package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rejisterjack/keywarden/internal/version"
)

func TestVersionCommand(t *testing.T) {
	out := captureStdout(t, func() {
		old := os.Args
		os.Args = []string{"keywarden", "version"}
		t.Cleanup(func() { os.Args = old })
		main()
	})
	if strings.TrimSpace(out) != version.Version {
		t.Fatalf("version = %q, want %s", out, version.Version)
	}
}

func TestCLICommandsCallAdminAPI(t *testing.T) {
	var sawAuth string
	var sawTaskBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tasks":
			sawTaskBody = string(body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"kwt_cli","task":{"id":"11111111-1111-4111-8111-111111111111","agent_id":"support"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tasks/11111111-1111-4111-8111-111111111111/revoke":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/policies":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"version":4}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/audit/verify":
			_, _ = w.Write([]byte(`{"ok":true,"events":7}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/orgs":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"organization":{"id":"22222222-2222-4222-8222-222222222222","slug":"acme"},"token":"kwa_first"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/orgs/22222222-2222-4222-8222-222222222222/tokens":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"kwa_ci"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("KEYWARDEN_URL", srv.URL)
	t.Setenv("KEYWARDEN_ADMIN_TOKEN", "test-admin-token-value")

	out := captureStdout(t, func() {
		if err := task([]string{"create", "--agent", "support", "--ttl", "15m", "--tool", "orders"}); err != nil {
			t.Fatal(err)
		}
	})
	if sawAuth != "Bearer test-admin-token-value" {
		t.Fatalf("auth = %q", sawAuth)
	}
	if !strings.Contains(sawTaskBody, `"agent_id":"support"`) || !strings.Contains(sawTaskBody, `"ttl_seconds":900`) || !strings.Contains(sawTaskBody, `"orders"`) {
		t.Fatalf("task body = %s", sawTaskBody)
	}
	if !strings.Contains(out, "kwt_cli") {
		t.Fatalf("stdout = %s", out)
	}
	if err := task([]string{"revoke", "11111111-1111-4111-8111-111111111111"}); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"version":1,"rules":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out = captureStdout(t, func() {
		if err := policyCmd([]string{"apply", policyPath}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "version 4") {
		t.Fatalf("policy stdout = %s", out)
	}
	out = captureStdout(t, func() {
		if err := auditCmd([]string{"verify"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "audit chain ok (7 events)") {
		t.Fatalf("audit stdout = %s", out)
	}
	out = captureStdout(t, func() {
		if err := orgCmd([]string{"create", "--slug", "acme", "--name", "Acme"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "kwa_first") {
		t.Fatalf("org stdout = %s", out)
	}
	out = captureStdout(t, func() {
		if err := orgCmd([]string{"token", "--org", "22222222-2222-4222-8222-222222222222", "--label", "ci"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "kwa_ci") {
		t.Fatalf("token stdout = %s", out)
	}
}

func TestCLIRejectsBadUsage(t *testing.T) {
	if err := task(nil); err == nil {
		t.Fatal("expected usage error")
	}
	if err := policyCmd([]string{"nope"}); err == nil {
		t.Fatal("expected usage error")
	}
	if err := auditCmd(nil); err == nil {
		t.Fatal("expected usage error")
	}
	if err := orgCmd([]string{"create", "--slug", "acme"}); err == nil {
		t.Fatal("expected missing name")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
