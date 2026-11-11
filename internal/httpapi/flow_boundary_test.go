package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMCPMethods(t *testing.T) {
	h := newHarness(t, false, false)
	token := h.taskToken()

	status, body := h.mcpStatus(http.MethodGet, token, "")
	if status != http.StatusMethodNotAllowed || !strings.Contains(body, "sse is not offered") {
		t.Fatalf("GET /mcp = %d %s", status, body)
	}
	status, body = h.mcpStatus(http.MethodDelete, token, "")
	if status != http.StatusNoContent {
		t.Fatalf("DELETE /mcp = %d %s", status, body)
	}
}

func TestAuthBoundaries(t *testing.T) {
	h := newHarness(t, false, false)

	status, _ := h.doAs(http.MethodGet, "/v1/agents", nil, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("missing bearer = %d", status)
	}
	status, _ = h.doAs(http.MethodGet, "/v1/agents", nil, "Token "+h.admin)
	if status != http.StatusUnauthorized {
		t.Fatalf("wrong scheme = %d", status)
	}

	out := h.adminJSON(http.MethodPost, "/v1/tasks", map[string]any{"agent_id": "support", "ttl_seconds": 60})
	task, _ := out["task"].(map[string]any)
	id, _ := task["id"].(string)
	if id == "" {
		t.Fatalf("task = %#v", out)
	}
	h.adminJSON(http.MethodPost, "/v1/tasks/"+id+"/revoke", map[string]any{})
	issued, _ := out["token"].(string)
	status, _ = h.mcpStatus(http.MethodPost, issued, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if status != http.StatusUnauthorized {
		t.Fatalf("revoked task = %d", status)
	}

	short := h.taskToken()
	if _, err := h.store.Pool().Exec(context.Background(), `UPDATE tasks SET expires_at = now() - interval '1 second' WHERE revoked_at IS NULL AND expires_at > now()`); err != nil {
		t.Fatal(err)
	}
	status, _ = h.mcpStatus(http.MethodPost, short, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if status != http.StatusUnauthorized {
		t.Fatalf("expired task = %d", status)
	}

	h.adminJSON(http.MethodPost, "/v1/agents", map[string]string{"id": "temp", "name": "Temp"})
	live := h.adminJSON(http.MethodPost, "/v1/tasks", map[string]any{"agent_id": "temp", "ttl_seconds": 60})
	liveToken, _ := live["token"].(string)
	status, body := h.do(http.MethodDelete, "/v1/agents/temp", nil)
	if status != http.StatusConflict || !strings.Contains(string(body), "referenced") {
		t.Fatalf("delete referenced agent = %d %s", status, body)
	}
	status, _ = h.mcpStatus(http.MethodPost, liveToken, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if status != http.StatusOK {
		t.Fatalf("task for a still-present agent = %d", status)
	}
}

func TestTaskListOmitsPlaintextToken(t *testing.T) {
	h := newHarness(t, false, false)
	out := h.adminJSON(http.MethodPost, "/v1/tasks", map[string]any{"agent_id": "support", "ttl_seconds": 60})
	token, _ := out["token"].(string)
	if !strings.HasPrefix(token, "kwt_") {
		t.Fatalf("token = %q", token)
	}
	listed := h.adminRaw(http.MethodGet, "/v1/tasks", nil)
	if strings.Contains(string(listed), token) || strings.Contains(string(listed), "token_hash") {
		t.Fatalf("task list leaked the credential: %s", listed)
	}
}

func TestPolicyBoundaries(t *testing.T) {
	h := newHarness(t, false, false)
	if status := h.adminStatus(http.MethodPost, "/v1/policies", []byte(`{"version":1,"rules":[{"id":"bad","match":{},"effect":"maybe"}]}`)); status != http.StatusBadRequest {
		t.Fatalf("unknown effect = %d", status)
	}
	if status := h.adminStatus(http.MethodPost, "/v1/policies", []byte(`{"version":1,"rules":[{"id":"slow","match":{},"effect":"require_approval","timeout_seconds":601}]}`)); status != http.StatusBadRequest {
		t.Fatalf("timeout 601 = %d", status)
	}
	if status := h.adminStatus(http.MethodPost, "/v1/policies", []byte(`{"version":1,"rules":[{"id":"slow","match":{},"effect":"allow","timeout_seconds":30}]}`)); status != http.StatusBadRequest {
		t.Fatalf("timeout without approval = %d", status)
	}

	h.adminRaw(http.MethodPost, "/v1/policies", []byte(`{"version":1,"rules":[]}`))
	token := h.taskToken()
	dials := h.dials.Load()
	got := h.mcp(token, "tools/call", `{"name":"orders__query","arguments":{"sql":"SELECT 1"}}`)
	if !got.isError || !strings.Contains(got.raw, "denied") {
		t.Fatalf("empty policy = error:%v %s", got.isError, got.raw)
	}
	if h.dials.Load() != dials {
		t.Fatal("empty policy opened a connection")
	}
}

func TestApprovalIsBoundToArgumentHash(t *testing.T) {
	h := newHarness(t, false, false)
	token := h.taskToken()
	doneDelete := make(chan mcpResult, 1)
	doneUpdate := make(chan mcpResult, 1)
	go func() {
		doneDelete <- h.mcp(token, "tools/call", `{"name":"orders__query","arguments":{"sql":"DELETE FROM demo_orders.shipments WHERE id = 1042"}}`)
	}()
	first := h.waitPending()
	go func() {
		doneUpdate <- h.mcp(token, "tools/call", `{"name":"orders__query","arguments":{"sql":"UPDATE demo_orders.shipments SET status = 'held' WHERE id = 1042"}}`)
	}()
	second := ""
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && second == "" {
		out := h.adminJSON(http.MethodGet, "/v1/approvals?status=pending", nil)
		items, _ := out["approvals"].([]any)
		for _, item := range items {
			row, _ := item.(map[string]any)
			id, _ := row["id"].(string)
			if id != "" && id != first {
				second = id
			}
		}
		if second == "" {
			time.Sleep(30 * time.Millisecond)
		}
	}
	if second == "" {
		t.Fatal("second approval did not appear")
	}
	h.adminJSON(http.MethodPost, "/v1/approvals/"+first, map[string]string{"decision": "approve", "comment": "only this hash"})
	deleted := <-doneDelete
	if deleted.isError {
		t.Fatalf("approved delete failed: %s", deleted.raw)
	}
	select {
	case updated := <-doneUpdate:
		t.Fatalf("different SQL was forwarded: %s", updated.raw)
	case <-time.After(200 * time.Millisecond):
	}
	h.adminJSON(http.MethodPost, "/v1/approvals/"+second, map[string]string{"decision": "deny", "comment": "other hash"})
	updated := <-doneUpdate
	if !updated.isError {
		t.Fatalf("denied update was forwarded: %s", updated.raw)
	}
}

func TestDecisionAfterDeadlineIsRejected(t *testing.T) {
	h := newHarness(t, false, false)
	h.adminRaw(http.MethodPost, "/v1/policies", []byte(`{
	  "version": 1,
	  "rules": [
	    {"id": "orders-mutate", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "mutating"}, "effect": "require_approval", "timeout_seconds": 1}
	  ]
	}`))
	token := h.taskToken()
	done := make(chan mcpResult, 1)
	go func() {
		done <- h.mcp(token, "tools/call", `{"name":"orders__query","arguments":{"sql":"DELETE FROM demo_orders.shipments WHERE id = 1042"}}`)
	}()
	id := h.waitPending()
	time.Sleep(1100 * time.Millisecond)
	status, body := h.do(http.MethodPost, "/v1/approvals/"+id, []byte(`{"decision":"approve","comment":"too late"}`))
	if status != http.StatusConflict || !strings.Contains(string(body), "expired") {
		t.Fatalf("late decision = %d %s", status, body)
	}
	got := <-done
	if !got.isError {
		t.Fatalf("late approval forwarded the call: %s", got.raw)
	}
}

func TestSQLBoundariesDoNotDial(t *testing.T) {
	h := newHarness(t, false, false)
	token := h.taskToken()
	cases := []string{
		`{"name":"orders__query","arguments":{"sql":""}}`,
		`{"name":"orders__query","arguments":{"sql":"-- just a comment"}}`,
		`{"name":"orders__query","arguments":{"sql":"SELECT 1; DROP TABLE demo_orders.scratch"}}`,
		`{"name":"orders__query","arguments":{"sql":"SELECT * INTO demo_orders.copy FROM demo_orders.shipments"}}`,
		`{"name":"orders__query","arguments":{"sql":"` + strings.Repeat(" ", 10001) + `SELECT 1"}}`,
	}
	for _, params := range cases {
		dials := h.dials.Load()
		got := h.mcp(token, "tools/call", params)
		if !got.isError && got.err == "" {
			t.Fatalf("accepted %s: %s", params, got.raw)
		}
		if h.dials.Load() != dials {
			t.Fatalf("dialed for %s", params)
		}
	}
}

func TestMCPBodyLimit(t *testing.T) {
	h := newHarness(t, false, false)
	token := h.taskToken()
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"pad":"` + strings.Repeat("a", 1<<20) + `"}}`
	status, raw := h.mcpStatus(http.MethodPost, token, body)
	if status == http.StatusOK && !strings.Contains(raw, "invalid request") {
		t.Fatalf("oversized body was accepted: %s", raw[:min(180, len(raw))])
	}
	if strings.Contains(raw, "orders__query") {
		t.Fatal("oversized body reached tools/list")
	}
}

func TestUpstreamFailureIsAudited(t *testing.T) {
	h := newHarness(t, false, false)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	t.Cleanup(api.Close)
	h.adminJSON(http.MethodPost, "/v1/tools", map[string]any{
		"id": "billingapi", "name": "Billing API", "kind": "http",
		"config": map[string]any{
			"base_url": api.URL, "allowed_methods": []string{"GET"}, "path_prefix": "/v1",
		},
		"secret": map[string]any{"headers": map[string]string{}},
	})
	h.adminRaw(http.MethodPost, "/v1/policies", []byte(`{
	  "version": 1,
	  "rules": [
	    {"id": "billing-get", "match": {"agent": "support", "tool": "billingapi", "name": "request"}, "effect": "allow"}
	  ]
	}`))
	before := h.adminJSON(http.MethodGet, "/v1/audit", nil)
	token := h.taskToken()
	got := h.mcp(token, "tools/call", `{"name":"billingapi__request","arguments":{"method":"GET","path":"/v1/ping"}}`)
	if !strings.Contains(got.raw, "500") {
		t.Fatalf("upstream 500 body = %s", got.raw)
	}
	after := h.adminJSON(http.MethodGet, "/v1/audit", nil)
	beforeN := countEvents(before)
	afterN := countEvents(after)
	if afterN <= beforeN {
		t.Fatalf("audit events before=%d after=%d", beforeN, afterN)
	}
	verify := h.adminJSON(http.MethodGet, "/v1/audit/verify", nil)
	if verify["ok"] != true {
		t.Fatalf("verify: %#v", verify)
	}
}

func countEvents(body map[string]any) int {
	items, _ := body["events"].([]any)
	return len(items)
}

func (h *harness) mcpStatus(method, token, body string) (int, string) {
	h.t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, h.srv.URL+"/mcp", r)
	if err != nil {
		h.t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(raw)
}

func (h *harness) doAs(method, path string, body []byte, auth string) (int, []byte) {
	h.t.Helper()
	var r io.Reader
	if body != nil {
		r = strings.NewReader(string(body))
	}
	req, err := http.NewRequest(method, h.srv.URL+path, r)
	if err != nil {
		h.t.Fatal(err)
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
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
