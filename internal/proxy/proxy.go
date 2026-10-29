package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rejisterjack/keywarden/internal/approval"
	"github.com/rejisterjack/keywarden/internal/canonical"
	"github.com/rejisterjack/keywarden/internal/fail"
	"github.com/rejisterjack/keywarden/internal/mcp"
	"github.com/rejisterjack/keywarden/internal/policy"
	"github.com/rejisterjack/keywarden/internal/redact"
	"github.com/rejisterjack/keywarden/internal/requestid"
	"github.com/rejisterjack/keywarden/internal/store"
	"github.com/rejisterjack/keywarden/internal/upstream"
	"github.com/rejisterjack/keywarden/internal/version"
)

// Proxy is the MCP data plane.
type Proxy struct {
	Store *store.Store
	Up    *upstream.Dispatcher
}

type taskKey struct{}

// WithTask stores the authenticated task on ctx.
func WithTask(ctx context.Context, task store.Task) context.Context {
	return context.WithValue(ctx, taskKey{}, task)
}

func taskFrom(ctx context.Context) (store.Task, bool) {
	task, ok := ctx.Value(taskKey{}).(store.Task)
	return task, ok
}

// ServeMCP handles one Streamable HTTP MCP request.
func (p *Proxy) ServeMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.Error(w, "sse is not offered", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodDelete {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	task, ok := taskFrom(r.Context())
	if !ok {
		writeRPCError(w, nil, -32001, "unauthorized")
		return
	}
	raw, err := readBody(r)
	if err != nil {
		writeRPCError(w, nil, -32700, "invalid request")
		return
	}
	req, err := mcp.ParseRequest(raw)
	if err != nil {
		writeRPCError(w, nil, -32700, "invalid request")
		return
	}
	if req.IsNotification() {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	switch req.Method {
	case "initialize":
		writeResult(w, req.ID, map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "keywarden", "version": version.Version},
		})
	case "ping":
		writeResult(w, req.ID, map[string]any{})
	case "tools/list":
		tools, err := p.list(r.Context(), task)
		if err != nil {
			writeRPCError(w, req.ID, -32000, publicErr(err))
			return
		}
		writeResult(w, req.ID, map[string]any{"tools": tools})
	case "tools/call":
		result, err := p.call(r.Context(), task, req.Params)
		if err != nil {
			writeRPCError(w, req.ID, -32000, publicErr(err))
			return
		}
		writeResult(w, req.ID, result)
	default:
		writeRPCError(w, req.ID, -32601, "method not found")
	}
}

func (p *Proxy) list(ctx context.Context, task store.Task) ([]mcp.Tool, error) {
	doc, err := p.Store.ActiveDocument(ctx)
	if err != nil {
		return nil, err
	}
	tools, err := p.Store.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	out := []mcp.Tool{}
	for _, pub := range tools {
		if !task.Allows(pub.ID) {
			continue
		}
		full, err := p.Store.GetTool(ctx, pub.ID)
		if err != nil {
			return nil, err
		}
		secretRaw, err := p.Store.OpenSecret(full)
		if err != nil {
			return nil, err
		}
		listed, err := p.Up.List(ctx, full, secretRaw)
		if err != nil {
			return nil, err
		}
		for _, tool := range listed {
			if !doc.Visible(task.AgentID, full.ID, tool.Name) {
				continue
			}
			if tool.InputSchema == nil {
				tool.InputSchema = json.RawMessage(`{"type":"object"}`)
			}
			tool.Name = full.ID + "__" + tool.Name
			out = append(out, tool)
		}
	}
	return out, nil
}

func (p *Proxy) call(ctx context.Context, task store.Task, params json.RawMessage) (mcp.ToolResult, error) {
	var body struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &body); err != nil || body.Name == "" {
		return mcp.ToolResult{}, fail.New(400, "tools/call requires a name")
	}
	toolID, name, ok := split(body.Name)
	if !ok {
		return mcp.ToolResult{}, fail.New(400, "tool name must be toolId__name")
	}
	argsHash, redacted, original, value, err := canonArgs(body.Arguments)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	base := map[string]any{
		"request_id":    requestid.From(ctx),
		"task_id":       task.ID.String(),
		"agent_id":      task.AgentID,
		"tool_id":       toolID,
		"mcp_tool":      name,
		"args_hash":     hex.EncodeToString(argsHash),
		"args_redacted": value,
	}
	if !task.Allows(toolID) {
		return p.refuse(ctx, task, toolID, name, redacted, argsHash, "", "task does not include this tool", base)
	}
	tool, err := p.Store.GetTool(ctx, toolID)
	if err != nil {
		if fail.IsStatus(err, 404) {
			return p.refuse(ctx, task, toolID, name, redacted, argsHash, "", "unknown tool", base)
		}
		return mcp.ToolResult{}, err
	}
	sqlClass := ""
	if tool.Kind == "postgres" {
		sqlText, err := sqlText(original)
		if err != nil {
			return mcp.ToolResult{}, err
		}
		if name != "query" {
			return p.refuse(ctx, task, toolID, name, redacted, argsHash, "", "unknown tool", base)
		}
		sqlClass = policy.Classify(sqlText)
		base["sql_class"] = sqlClass
	}
	doc, err := p.Store.ActiveDocument(ctx)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	decision := doc.Evaluate(policy.Input{AgentID: task.AgentID, ToolID: toolID, Name: name, SQLClass: sqlClass})
	base["rule_id"] = decision.RuleID
	base["reason"] = decision.Reason
	base["decision"] = decision.Effect
	switch decision.Effect {
	case policy.Deny:
		return p.refuse(ctx, task, toolID, name, redacted, argsHash, decision.RuleID, decision.Reason, base)
	case policy.Allow:
		return p.forward(ctx, task, tool, name, body.Arguments, redacted, argsHash, "allow", decision.RuleID, decision.Reason, base)
	case policy.RequireApproval:
		return p.approveThenForward(ctx, task, tool, name, body.Arguments, redacted, argsHash, decision, base)
	default:
		return p.refuse(ctx, task, toolID, name, redacted, argsHash, decision.RuleID, "denied", base)
	}
}

func (p *Proxy) refuse(ctx context.Context, task store.Task, toolID, name string, redacted, argsHash []byte, rule, reason string, audit map[string]any) (mcp.ToolResult, error) {
	audit["decision"] = "deny"
	audit["rule_id"] = rule
	audit["reason"] = reason
	_, err := p.Store.RecordCall(ctx, store.CallInput{
		TaskID: task.ID, AgentID: task.AgentID, ToolID: toolID, MCPToolName: name,
		ArgsRedacted: redacted, ArgsHash: argsHash, Decision: "deny", RuleID: rule, Reason: reason,
		RequestID: requestid.From(ctx), UpstreamStatus: "not_called",
		AuditType: "tool_call.denied", Audit: audit,
	})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	return mcp.TextResult("denied: "+reason, true), nil
}

func (p *Proxy) approveThenForward(ctx context.Context, task store.Task, tool store.Tool, name string, args, redacted, argsHash []byte, decision policy.Decision, audit map[string]any) (mcp.ToolResult, error) {
	audit["decision"] = "pending"
	call, appr, err := p.Store.RecordPending(ctx, store.CallInput{
		TaskID: task.ID, AgentID: task.AgentID, ToolID: tool.ID, MCPToolName: name,
		ArgsRedacted: redacted, ArgsHash: argsHash, RuleID: decision.RuleID, Reason: decision.Reason,
		RequestID: requestid.From(ctx), Audit: audit,
	}, decision.Timeout)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	status, err := approval.Wait(ctx, p.Store, appr.ID)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	if ctx.Err() != nil {
		return mcp.TextResult("denied: request cancelled before the call was forwarded", true), nil
	}
	if status != "approved" {
		completed := cloneAudit(audit)
		completed["decision"] = status
		completed["tool_call_id"] = call.ID.String()
		_ = p.Store.FinishCall(ctx, call.ID, "not_called", "tool_call.completed", completed)
		return mcp.TextResult("denied: approval "+status, true), nil
	}
	fresh, err := p.Store.GetTool(ctx, tool.ID)
	if err != nil {
		return mcp.ToolResult{}, err
	}
	return p.execute(ctx, fresh, call.ID, name, args, audit, "approved")
}

func (p *Proxy) forward(ctx context.Context, task store.Task, tool store.Tool, name string, args, redacted, argsHash []byte, decision, rule, reason string, audit map[string]any) (mcp.ToolResult, error) {
	audit["decision"] = decision
	call, err := p.Store.RecordCall(ctx, store.CallInput{
		TaskID: task.ID, AgentID: task.AgentID, ToolID: tool.ID, MCPToolName: name,
		ArgsRedacted: redacted, ArgsHash: argsHash, Decision: decision, RuleID: rule, Reason: reason,
		RequestID: requestid.From(ctx), UpstreamStatus: "",
		AuditType: "tool_call.allowed", Audit: audit,
	})
	if err != nil {
		return mcp.ToolResult{}, err
	}
	return p.execute(ctx, tool, call.ID, name, args, audit, decision)
}

func (p *Proxy) execute(ctx context.Context, tool store.Tool, callID uuid.UUID, name string, args []byte, audit map[string]any, decision string) (mcp.ToolResult, error) {
	if err := p.Store.MarkRunning(ctx, callID); err != nil {
		return mcp.TextResult("denied: call was not forwarded", true), nil
	}
	secretRaw, err := p.Store.OpenSecret(tool)
	if err != nil {
		completed := cloneAudit(audit)
		completed["decision"] = decision
		completed["tool_call_id"] = callID.String()
		completed["error"] = "secret unavailable"
		_ = p.Store.FinishCall(context.Background(), callID, "error", "tool_call.completed", completed)
		return mcp.ToolResult{}, err
	}
	result, err := p.Up.Call(ctx, tool, secretRaw, name, args)
	completed := cloneAudit(audit)
	completed["decision"] = decision
	completed["tool_call_id"] = callID.String()
	if err != nil {
		completed["error"] = publicErr(err)
		_ = p.Store.FinishCall(context.Background(), callID, "error", "tool_call.completed", completed)
		return mcp.TextResult("upstream error: "+publicErr(err), true), nil
	}
	completed["upstream_status"] = "ok"
	if err := p.Store.FinishCall(ctx, callID, "ok", "tool_call.completed", completed); err != nil {
		return mcp.ToolResult{}, err
	}
	if tool.Kind != "postgres" {
		var parsed mcp.ToolResult
		if err := json.Unmarshal([]byte(result), &parsed); err == nil && parsed.Content != nil {
			return parsed, nil
		}
	}
	return mcp.TextResult(result, false), nil
}

func canonArgs(raw json.RawMessage) (hash []byte, redacted json.RawMessage, original, redactedValue any, err error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		raw = []byte(`{}`)
	}
	c, err := canonical.FromJSON(raw)
	if err != nil {
		return nil, nil, nil, nil, fail.New(400, "arguments must be JSON")
	}
	var v any
	dec := json.NewDecoder(strings.NewReader(string(c)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, nil, nil, nil, fail.New(400, "arguments must be JSON")
	}
	if _, ok := v.(map[string]any); !ok {
		return nil, nil, nil, nil, fail.New(400, "arguments must be a JSON object")
	}
	sum := sha256.Sum256(c)
	red := redact.Value(v)
	rb, err := canonical.Bytes(red)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return sum[:], rb, v, red, nil
}

func sqlText(v any) (string, error) {
	m, _ := v.(map[string]any)
	s, _ := m["sql"].(string)
	if strings.TrimSpace(s) == "" {
		return "", fail.New(400, "sql is required")
	}
	if len(s) > 10000 {
		return "", fail.New(400, "sql is too long")
	}
	return s, nil
}

func split(public string) (string, string, bool) {
	i := strings.Index(public, "__")
	if i <= 0 || i+2 >= len(public) {
		return "", "", false
	}
	id, name := public[:i], public[i+2:]
	if !okSlug(id) || len(name) > 128 {
		return "", "", false
	}
	return id, name, true
}

func okSlug(id string) bool {
	if id == "" || len(id) > 63 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '_', c == '-':
		default:
			return false
		}
	}
	if id[0] < 'a' || id[0] > 'z' {
		return false
	}
	return !strings.Contains(id, "__")
}

func cloneAudit(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+2)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func publicErr(err error) string {
	var fe *fail.Error
	if errors.As(err, &fe) {
		return fe.Error()
	}
	msg := redact.String(err.Error())
	if msg == "" {
		return "request failed"
	}
	return msg
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	buf, err := io.ReadAll(io.LimitReader(r.Body, 1<<20+1))
	if err != nil {
		return nil, err
	}
	if len(buf) > 1<<20 {
		return nil, fail.New(400, "body too large")
	}
	return buf, nil
}

func writeResult(w http.ResponseWriter, id json.RawMessage, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mcp.Result{JSONRPC: "2.0", ID: id, Result: result})
}

func writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	if id == nil {
		id = json.RawMessage(`null`)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mcp.Failure{JSONRPC: "2.0", ID: id, Error: mcp.ErrorBody{Code: code, Message: message}})
}
