package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rejisterjack/keywarden/internal/config"
	"github.com/rejisterjack/keywarden/internal/limit"
	"github.com/rejisterjack/keywarden/internal/logging"
	"github.com/rejisterjack/keywarden/internal/mcp"
	"github.com/rejisterjack/keywarden/internal/proxy"
	"github.com/rejisterjack/keywarden/internal/store"
	"github.com/rejisterjack/keywarden/internal/testlock"
	"github.com/rejisterjack/keywarden/internal/upstream"
	"github.com/rejisterjack/keywarden/internal/vault"
)

func TestMain(m *testing.M) {
	url := os.Getenv("KEYWARDEN_TEST_DATABASE_URL")
	if url == "" {
		os.Exit(m.Run())
	}
	release, err := testlock.Acquire(url)
	if err != nil {
		panic(err)
	}
	code := m.Run()
	release()
	os.Exit(code)
}

type harness struct {
	t      *testing.T
	srv    *httptest.Server
	store  *store.Store
	dsn    string
	admin  string
	dials  atomic.Int32
	mcpHit atomic.Int32
}

func newHarness(t *testing.T, demo, lock bool) *harness {
	t.Helper()
	dsn := os.Getenv("KEYWARDEN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KEYWARDEN_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	key := bytes.Repeat([]byte{7}, 32)
	v, err := vault.New(key)
	if err != nil {
		t.Fatal(err)
	}
	orders := dsn
	fixtureURL := "http://127.0.0.1:9/mcp"
	st, err := store.Open(ctx, store.Options{
		DatabaseURL: dsn,
		Vault:       v,
		Demo:        demo,
		AllowStdio:  !demo,
		FixtureURL:  fixtureURL,
		OrdersDSN:   orders,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.ResetForTests(ctx); err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, store: st, dsn: dsn, admin: "test-admin-token-value"}
	mcpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mcpHit.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "test")
		body, _ := io.ReadAll(r.Body)
		if bytes.Contains(body, []byte(`"method":"notifications/initialized"`)) || bytes.Contains(body, []byte(`"method": "notifications/initialized"`)) {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"lookup_order","description":"lookup","inputSchema":{"type":"object"}},{"name":"delete_catalog","description":"no","inputSchema":{"type":"object"}}],"content":[{"type":"text","text":"shipped"}],"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"fake","version":"0"}}}`))
	}))
	t.Cleanup(mcpSrv.Close)
	hUp := &upstream.Dispatcher{
		Connect: func(ctx context.Context, got string) (*pgx.Conn, error) {
			h.dials.Add(1)
			return pgx.Connect(ctx, got)
		},
	}
	cfg := config.Config{AdminToken: h.admin, Demo: demo, DemoLock: lock, MCPRate: 100, MCPBurst: 100, FixtureURL: mcpSrv.URL, OrdersDSN: dsn}
	// The store was opened with a placeholder fixture URL. Demo tests that create tools
	// use cfg only for the lock flag; tool validation uses the store's fixture URL.
	api := &Server{
		Store: st,
		Proxy: &proxy.Proxy{Store: st, Up: hUp},
		Cfg:   cfg,
		Log:   logging.New(io.Discard),
		Limit: limit.New(100, 100),
	}
	h.srv = httptest.NewServer(api.Handler())
	t.Cleanup(h.srv.Close)
	if !demo {
		h.mustSQL(ctx, `CREATE SCHEMA IF NOT EXISTS demo_orders`)
		h.mustSQL(ctx, `CREATE TABLE IF NOT EXISTS demo_orders.shipments (id int PRIMARY KEY, status text, customer text)`)
		h.mustSQL(ctx, `CREATE TABLE IF NOT EXISTS demo_orders.scratch (id int)`)
		h.mustSQL(ctx, `INSERT INTO demo_orders.shipments (id, status, customer) VALUES (1042, 'shipped', 'Ada Lovelace') ON CONFLICT (id) DO NOTHING`)
		h.adminJSON(http.MethodPost, "/v1/agents", map[string]string{"id": "support", "name": "Support"})
		h.adminJSON(http.MethodPost, "/v1/tools", map[string]any{
			"id": "orders", "name": "Orders", "kind": "postgres", "config": map[string]any{},
			"secret": map[string]string{"dsn": dsn},
		})
		h.adminJSON(http.MethodPost, "/v1/tools", map[string]any{
			"id": "catalog", "name": "Catalog", "kind": "mcp_http",
			"config": map[string]string{"url": mcpSrv.URL},
			"secret": map[string]any{"headers": map[string]string{}},
		})
		h.adminRaw(http.MethodPost, "/v1/policies", []byte(flowPolicy))
	}
	return h
}

const flowPolicy = `{
  "version": 1,
  "rules": [
    {"id": "orders-read", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "read"}, "effect": "allow"},
    {"id": "orders-mutate", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "mutating"}, "effect": "require_approval", "timeout_seconds": 30},
    {"id": "orders-destroy", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "destructive"}, "effect": "deny"},
    {"id": "catalog-lookup", "match": {"agent": "support", "tool": "catalog", "name": "lookup_order"}, "effect": "allow"},
    {"id": "catalog-refund", "match": {"agent": "support", "tool": "catalog", "name": "refund_order"}, "effect": "require_approval", "timeout_seconds": 1},
    {"id": "catalog-delete", "match": {"agent": "support", "tool": "catalog", "name": "delete_catalog"}, "effect": "deny"}
  ]
}`

func TestAllowDenyAndApproval(t *testing.T) {
	h := newHarness(t, false, false)
	token := h.taskToken()
	listed := h.mcp(token, "tools/list", `{}`)
	if listed.err != "" || !strings.Contains(listed.raw, "orders__query") || !strings.Contains(listed.raw, "catalog__lookup_order") {
		t.Fatalf("list: %s %s", listed.err, listed.raw)
	}
	if strings.Contains(listed.raw, "delete_catalog") {
		t.Fatalf("deny-only tool was listed: %s", listed.raw)
	}

	dials := h.dials.Load()
	got := h.mcp(token, "tools/call", `{"name":"orders__query","arguments":{"sql":"SELECT status FROM demo_orders.shipments WHERE id = 1042"}}`)
	if got.isError || !strings.Contains(got.raw, "shipped") {
		t.Fatalf("allow: error=%v body=%s rpc=%s", got.isError, got.raw, got.err)
	}
	if h.dials.Load() != dials+1 {
		t.Fatalf("select dials = %d, want %d", h.dials.Load(), dials+1)
	}

	dials = h.dials.Load()
	hits := h.mcpHit.Load()
	got = h.mcp(token, "tools/call", `{"name":"orders__query","arguments":{"sql":"DROP TABLE demo_orders.scratch"}}`)
	if !got.isError || !strings.Contains(got.raw, "denied") {
		t.Fatalf("deny: error=%v body=%s", got.isError, got.raw)
	}
	if h.dials.Load() != dials {
		t.Fatal("denied call opened a database connection")
	}
	if h.mcpHit.Load() != hits {
		t.Fatal("denied call contacted the MCP upstream")
	}
	var still *string
	if err := h.store.Pool().QueryRow(context.Background(), `SELECT to_regclass('demo_orders.scratch')::text`).Scan(&still); err != nil || still == nil {
		t.Fatal("scratch table disappeared", err, still)
	}

	dials = h.dials.Load()
	done := make(chan mcpResult, 1)
	go func() {
		done <- h.mcp(token, "tools/call", `{"name":"orders__query","arguments":{"sql":"DELETE FROM demo_orders.shipments WHERE id = 1042"}}`)
	}()
	approvalID := h.waitPending()
	h.adminJSON(http.MethodPost, "/v1/approvals/"+approvalID, map[string]string{"decision": "approve", "comment": "customer asked"})
	h.adminJSON(http.MethodPost, "/v1/approvals/"+approvalID, map[string]string{"decision": "approve", "comment": "again"})
	res := h.adminStatus(http.MethodPost, "/v1/approvals/"+approvalID, map[string]string{"decision": "deny"})
	if res != http.StatusConflict {
		t.Fatalf("conflicting decision status = %d", res)
	}
	got = <-done
	if got.isError || !strings.Contains(got.raw, "DELETE") {
		t.Fatalf("approved delete: error=%v body=%s rpc=%s", got.isError, got.raw, got.err)
	}
	if h.dials.Load() != dials+1 {
		t.Fatalf("approved dials = %d", h.dials.Load())
	}

	verify := h.adminJSON(http.MethodGet, "/v1/audit/verify", nil)
	if verify["ok"] != true {
		t.Fatalf("verify: %#v", verify)
	}
	if _, err := h.store.Pool().Exec(context.Background(), `UPDATE audit_events SET payload = payload || 'x' WHERE seq = (SELECT max(seq) FROM audit_events)`); err != nil {
		t.Fatal(err)
	}
	verify = h.adminJSON(http.MethodGet, "/v1/audit/verify", nil)
	if verify["ok"] != false {
		t.Fatalf("tampered chain still verified: %#v", verify)
	}
}

func TestTimeoutAndRestartDoNotForward(t *testing.T) {
	h := newHarness(t, false, false)
	h.adminRaw(http.MethodPost, "/v1/policies", []byte(`{
	  "version": 1,
	  "rules": [
	    {"id": "orders-mutate", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "mutating"}, "effect": "require_approval", "timeout_seconds": 1},
	    {"id": "orders-destroy", "match": {"tool": "orders", "name": "query"}, "when": {"sql": "destructive"}, "effect": "deny"}
	  ]
	}`))
	token := h.taskToken()
	dials := h.dials.Load()
	got := h.mcp(token, "tools/call", `{"name":"orders__query","arguments":{"sql":"UPDATE demo_orders.shipments SET status = 'held' WHERE id = 1042"}}`)
	if !got.isError || !strings.Contains(got.raw, "expired") {
		t.Fatalf("timeout: error=%v body=%s", got.isError, got.raw)
	}
	if h.dials.Load() != dials {
		t.Fatal("timed-out call opened a database connection")
	}

	h.adminRaw(http.MethodPost, "/v1/policies", []byte(`{
	  "version": 1,
	  "rules": [
	    {"id": "orders-mutate", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "mutating"}, "effect": "require_approval", "timeout_seconds": 30}
	  ]
	}`))
	token = h.taskToken()
	dials = h.dials.Load()
	done := make(chan mcpResult, 1)
	go func() {
		done <- h.mcp(token, "tools/call", `{"name":"orders__query","arguments":{"sql":"UPDATE demo_orders.shipments SET status = 'held' WHERE id = 1042"}}`)
	}()
	_ = h.waitPending()
	n, err := h.store.InterruptInFlight(context.Background())
	if err != nil || n == 0 {
		t.Fatal(err, n)
	}
	got = <-done
	if !got.isError {
		t.Fatalf("restart forwarded the call: %s", got.raw)
	}
	if h.dials.Load() != dials {
		t.Fatal("interrupted call opened a database connection")
	}
}

func TestSecretsStayWriteOnly(t *testing.T) {
	h := newHarness(t, false, false)
	const leak = "vault-secret-should-not-leak"
	res := h.adminRaw(http.MethodPost, "/v1/tools", []byte(`{
		"id":"billing","name":"Billing","kind":"postgres","config":{},
		"secret":{"dsn":"postgres://bill:`+leak+`@127.0.0.1:5432/billing"}
	}`))
	if strings.Contains(string(res), leak) {
		t.Fatalf("create response leaked the secret: %s", res)
	}
	listed := h.adminRaw(http.MethodGet, "/v1/tools", nil)
	if strings.Contains(string(listed), leak) || strings.Contains(string(listed), "ciphertext") {
		t.Fatalf("list leaked the secret: %s", listed)
	}
}

func TestDemoLockAndFakeSecrets(t *testing.T) {
	h := newHarness(t, true, true)
	status := h.adminStatus(http.MethodPost, "/v1/agents", map[string]string{"id": "support", "name": "Support"})
	if status != http.StatusForbidden {
		t.Fatalf("demo lock status = %d", status)
	}
	if err := h.store.AssertDemoSecrets(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err := h.store.CreateTool(context.Background(), store.CreateToolInput{
		ID: "orders", Name: "Orders", Kind: "postgres", Config: []byte(`{}`),
		Secret: []byte(`{"dsn":"` + h.dsn + `"}`),
	})
	if err == nil {
		t.Fatal("demo mode accepted a secret without kw_demo=fake")
	}
}

type mcpResult struct {
	raw     string
	err     string
	isError bool
}

func (h *harness) mcp(token, method, params string) mcpResult {
	h.t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`
	req, err := http.NewRequest(http.MethodPost, h.srv.URL+"/mcp", strings.NewReader(body))
	if err != nil {
		h.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return mcpResult{err: string(raw)}
	}
	var decoded struct {
		Result mcp.ToolResult `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		h.t.Fatal(err)
	}
	if decoded.Error != nil {
		return mcpResult{err: decoded.Error.Message, raw: string(raw)}
	}
	text := ""
	if len(decoded.Result.Content) > 0 {
		text = decoded.Result.Content[0].Text
	}
	return mcpResult{raw: text + string(raw), isError: decoded.Result.IsError}
}

func (h *harness) taskToken() string {
	h.t.Helper()
	out := h.adminJSON(http.MethodPost, "/v1/tasks", map[string]any{"agent_id": "support", "ttl_seconds": 900})
	token, _ := out["token"].(string)
	if token == "" {
		h.t.Fatalf("no token: %#v", out)
	}
	return token
}

func (h *harness) waitPending() string {
	h.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		out := h.adminJSON(http.MethodGet, "/v1/approvals?status=pending", nil)
		items, _ := out["approvals"].([]any)
		if len(items) > 0 {
			row, _ := items[0].(map[string]any)
			id, _ := row["id"].(string)
			if id != "" {
				return id
			}
		}
		time.Sleep(30 * time.Millisecond)
	}
	h.t.Fatal("timed out waiting for a pending approval")
	return ""
}

func (h *harness) adminJSON(method, path string, body any) map[string]any {
	h.t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			h.t.Fatal(err)
		}
	}
	resp := h.adminRaw(method, path, raw)
	if len(bytes.TrimSpace(resp)) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(resp, &out); err != nil {
		h.t.Fatalf("%s %s: %s", method, path, resp)
	}
	if msg, ok := out["error"].(string); ok && method != http.MethodGet {
		h.t.Fatalf("%s %s: %s", method, path, msg)
	}
	return out
}

func (h *harness) adminRaw(method, path string, body []byte) []byte {
	h.t.Helper()
	status, raw := h.do(method, path, body)
	if status >= 300 {
		h.t.Fatalf("%s %s -> %d %s", method, path, status, raw)
	}
	return raw
}

func (h *harness) adminStatus(method, path string, body any) int {
	h.t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	status, _ := h.do(method, path, raw)
	return status
}

func (h *harness) do(method, path string, body []byte) (int, []byte) {
	h.t.Helper()
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, h.srv.URL+path, r)
	if err != nil {
		h.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+h.admin)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

func (h *harness) mustSQL(ctx context.Context, sql string) {
	h.t.Helper()
	if _, err := h.store.Pool().Exec(ctx, sql); err != nil {
		h.t.Fatal(err)
	}
}
