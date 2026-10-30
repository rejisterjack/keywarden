package upstream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func httpConfig(base, prefix string, methods ...string) []byte {
	cfg, _ := json.Marshal(map[string]any{
		"base_url":        base,
		"allowed_methods": methods,
		"path_prefix":     prefix,
	})
	return cfg
}

func TestPlainHTTPCallForwardsWithSealedHeaders(t *testing.T) {
	var gotAuth, gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		buf := make([]byte, 64)
		n, _ := r.Body.Read(buf)
		gotBody = strings.TrimSpace(string(buf[:n]))
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	p := PlainHTTP{}
	secret := []byte(`{"headers":{"Authorization":"Bearer sealed-token"}}`)
	args := []byte(`{"method":"POST","path":"/v1/orders","headers":{"X-Trace":"1"},"body":"{\"a\":1}"}`)
	out, err := p.Call(context.Background(), httpConfig(srv.URL, "/v1", "GET", "POST"), secret, args)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if gotAuth != "Bearer sealed-token" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if gotMethod != "POST" || gotPath != "/v1/orders" {
		t.Fatalf("method/path = %s %s", gotMethod, gotPath)
	}
	if gotBody == "" {
		t.Fatal("body was not forwarded")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output: %v", err)
	}
	if decoded["status"] != float64(201) || decoded["body"] != `{"ok":true}` {
		t.Fatalf("output = %s", out)
	}
}

func TestPlainHTTPRejectsDisallowedMethod(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()
	p := PlainHTTP{}
	secret := []byte(`{"headers":{}}`)
	_, err := p.Call(context.Background(), httpConfig(srv.URL, "", "GET"), secret, []byte(`{"method":"DELETE","path":"/x"}`))
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected method rejection, got %v", err)
	}
	if called {
		t.Fatal("disallowed method reached the upstream")
	}
}

func TestPlainHTTPRejectsPathOutsidePrefix(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()
	p := PlainHTTP{}
	secret := []byte(`{"headers":{}}`)
	_, err := p.Call(context.Background(), httpConfig(srv.URL, "/v1", "GET"), secret, []byte(`{"method":"GET","path":"/admin"}`))
	if err == nil || !strings.Contains(err.Error(), "prefix") {
		t.Fatalf("expected prefix rejection, got %v", err)
	}
	if called {
		t.Fatal("out-of-prefix path reached the upstream")
	}
}

func TestPlainHTTPRejectsTraversal(t *testing.T) {
	p := PlainHTTP{}
	secret := []byte(`{"headers":{}}`)
	_, err := p.Call(context.Background(), httpConfig("https://api.example.com", "/v1", "GET"), secret, []byte(`{"method":"GET","path":"/v1/../admin"}`))
	if err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestPlainHTTPListSingleTool(t *testing.T) {
	p := PlainHTTP{}
	tools, err := p.List(context.Background(), httpConfig("https://api.example.com", "", "GET"), []byte(`{"headers":{}}`))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "request" {
		t.Fatalf("tools = %+v", tools)
	}
}
