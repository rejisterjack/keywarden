package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rejisterjack/keywarden/internal/config"
	"github.com/rejisterjack/keywarden/internal/fail"
	"github.com/rejisterjack/keywarden/internal/limit"
	"github.com/rejisterjack/keywarden/internal/proxy"
	"github.com/rejisterjack/keywarden/internal/redact"
	"github.com/rejisterjack/keywarden/internal/requestid"
	"github.com/rejisterjack/keywarden/internal/store"
	"github.com/rejisterjack/keywarden/internal/version"
)

// Server is the control plane and the MCP endpoint.
type Server struct {
	Store   *store.Store
	Proxy   *proxy.Proxy
	Cfg     config.Config
	Log     *slog.Logger
	Limit   *limit.Limiter
	Fixture http.Handler
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("GET /v1/meta", s.meta)
	mux.HandleFunc("/mcp", s.mcp)
	if s.Cfg.Demo && s.Fixture != nil {
		mux.Handle("/demo/mcp", s.Fixture)
	}
	mux.HandleFunc("GET /v1/agents", s.listAgents)
	mux.HandleFunc("POST /v1/agents", s.createAgent)
	mux.HandleFunc("DELETE /v1/agents/{id}", s.deleteAgent)
	mux.HandleFunc("GET /v1/tools", s.listTools)
	mux.HandleFunc("POST /v1/tools", s.createTool)
	mux.HandleFunc("DELETE /v1/tools/{id}", s.deleteTool)
	mux.HandleFunc("GET /v1/policies", s.listPolicies)
	mux.HandleFunc("GET /v1/policies/active", s.activePolicy)
	mux.HandleFunc("POST /v1/policies", s.applyPolicy)
	mux.HandleFunc("GET /v1/tasks", s.listTasks)
	mux.HandleFunc("POST /v1/tasks", s.createTask)
	mux.HandleFunc("POST /v1/tasks/{id}/revoke", s.revokeTask)
	mux.HandleFunc("GET /v1/approvals", s.listApprovals)
	mux.HandleFunc("POST /v1/approvals/{id}", s.decideApproval)
	mux.HandleFunc("GET /v1/audit", s.listAudit)
	mux.HandleFunc("GET /v1/audit/verify", s.verifyAudit)
	return s.wrap(mux)
}

func (s *Server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" || len(id) > 128 || strings.ContainsAny(id, " \r\n") {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		r = r.WithContext(requestid.With(r.Context(), id))
		sw := &statusWriter{ResponseWriter: w, code: 200}
		if strings.HasPrefix(r.URL.Path, "/mcp") || strings.HasPrefix(r.URL.Path, "/demo/mcp") {
			if s.Limit != nil && !s.Limit.Allow(clientIP(r, s.Cfg.TrustProxy)) {
				sw.Header().Set("Retry-After", "1")
				writeError(sw, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
		}
		if needsAdmin(r.URL.Path) {
			orgID, err := s.Store.AuthenticateAdmin(r.Context(), bearer(r), s.Cfg.AdminToken)
			if err != nil {
				writeError(sw, http.StatusUnauthorized, "unauthorized")
				return
			}
			r = r.WithContext(store.WithOrg(r.Context(), orgID))
		}
		if s.locked(r) {
			writeError(sw, http.StatusForbidden, "the hosted demo does not accept this change")
			return
		}
		next.ServeHTTP(sw, r)
		if s.Log != nil && r.URL.Path != "/healthz" && r.URL.Path != "/readyz" {
			s.Log.Info("request", "method", r.Method, "path", r.URL.Path, "status", sw.code, "request_id", id)
		}
	})
}

func (s *Server) locked(r *http.Request) bool {
	if !s.Cfg.DemoLock || r.Method == http.MethodGet || r.Method == http.MethodHead {
		return false
	}
	switch {
	case strings.HasPrefix(r.URL.Path, "/v1/agents"),
		strings.HasPrefix(r.URL.Path, "/v1/tools"),
		strings.HasPrefix(r.URL.Path, "/v1/policies"):
		return true
	default:
		return false
	}
}

func needsAdmin(path string) bool {
	switch path {
	case "/healthz", "/readyz", "/v1/meta", "/mcp", "/demo/mcp":
		return false
	default:
		return true
	}
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) meta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":          "keywarden",
		"version":       version.Version,
		"demo":          s.Cfg.Demo,
		"demo_lock":     s.Cfg.DemoLock,
		"single_tenant": s.Cfg.SingleTenant,
		"default_org":   s.Store.DefaultOrg().String(),
	})
}

func (s *Server) mcp(w http.ResponseWriter, r *http.Request) {
	task, err := s.Store.Authenticate(r.Context(), bearer(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := store.WithOrg(r.Context(), task.OrgID)
	s.Proxy.ServeMCP(w, r.WithContext(proxy.WithTask(ctx, task)))
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListAgents(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": items})
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if !decode(w, r, &body) {
		return
	}
	item, err := s.Store.CreateAgent(r.Context(), body.ID, body.Name)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) deleteAgent(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DeleteAgent(r.Context(), r.PathValue("id")); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listTools(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListTools(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tools": items})
}

func (s *Server) createTool(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID     string          `json:"id"`
		Name   string          `json:"name"`
		Kind   string          `json:"kind"`
		Config json.RawMessage `json:"config"`
		Secret json.RawMessage `json:"secret"`
	}
	if !decode(w, r, &body) {
		return
	}
	item, err := s.Store.CreateTool(r.Context(), store.CreateToolInput{
		ID: body.ID, Name: body.Name, Kind: body.Kind, Config: body.Config, Secret: body.Secret,
	})
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) deleteTool(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DeleteTool(r.Context(), r.PathValue("id")); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listPolicies(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListPolicies(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"policies": items})
}

func (s *Server) activePolicy(w http.ResponseWriter, r *http.Request) {
	item, err := s.Store.ActivePolicy(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) applyPolicy(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid policy")
		return
	}
	item, err := s.Store.ApplyPolicy(r.Context(), body)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListTasks(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": items})
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AgentID    string    `json:"agent_id"`
		TTLSeconds int       `json:"ttl_seconds"`
		ToolIDs    *[]string `json:"tool_ids"`
	}
	if !decode(w, r, &body) {
		return
	}
	var tools []string
	if body.ToolIDs != nil {
		tools = *body.ToolIDs
	}
	task, raw, err := s.Store.IssueTask(r.Context(), body.AgentID, tools, time.Duration(body.TTLSeconds)*time.Second)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"task": task, "token": raw})
}

func (s *Server) revokeTask(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	if err := s.Store.RevokeTask(r.Context(), id); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listApprovals(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	if status == "all" {
		status = ""
	}
	items, err := s.Store.ListApprovals(r.Context(), status)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approvals": items})
}

func (s *Server) decideApproval(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid approval id")
		return
	}
	var body struct {
		Decision string `json:"decision"`
		Comment  string `json:"comment"`
	}
	if !decode(w, r, &body) {
		return
	}
	approve, ok := approvalDecision(body.Decision)
	if !ok {
		writeError(w, http.StatusBadRequest, "decision must be approve or deny")
		return
	}
	item, err := s.Store.Decide(r.Context(), id, approve, body.Comment, "admin")
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.Store.ListAudit(r.Context(), after, limit)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": items})
}

func (s *Server) verifyAudit(w http.ResponseWriter, r *http.Request) {
	res, err := s.Store.VerifyAudit(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	var fe *fail.Error
	if errors.As(err, &fe) {
		writeError(w, fe.Status, fe.Error())
		return
	}
	if s.Log != nil {
		s.Log.Error("request failed", "err", redact.String(err.Error()))
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}

func approvalDecision(v string) (approve bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "approve", "approved":
		return true, true
	case "deny", "rejected":
		return false, true
	default:
		return false, false
	}
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func bearer(r *http.Request) string {
	scheme, rest, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(rest)
}

func clientIP(r *http.Request, trust bool) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if trust {
		if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
			first, _, _ := strings.Cut(fwd, ",")
			first = strings.TrimSpace(first)
			if first != "" {
				return first
			}
		}
	}
	return host
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (s *statusWriter) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}
