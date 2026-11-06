package upstream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cloudConfig(base string) []byte {
	cfg, _ := json.Marshal(map[string]any{
		"base_url":        base,
		"allowed_methods": []string{"GET", "POST"},
		"path_prefix":     "/v1",
	})
	return cfg
}

func TestGCPHTTPInjectsBearerToken(t *testing.T) {
	var gotAuth, gotExtra, gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotExtra = r.Header.Get("X-Goog-User-Project")
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	g := GCPHTTP{}
	secret := []byte(`{"access_token":"ya29.gcp-token","headers":{"X-Goog-User-Project":"demo-proj"}}`)
	out, err := g.Call(context.Background(), cloudConfig(srv.URL), secret, []byte(`{"method":"GET","path":"/v1/projects"}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if gotAuth != "Bearer ya29.gcp-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotExtra != "demo-proj" {
		t.Fatalf("extra header = %q", gotExtra)
	}
	if gotMethod != "GET" || gotPath != "/v1/projects" {
		t.Fatalf("method/path = %s %s", gotMethod, gotPath)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["status"] != float64(200) {
		t.Fatalf("output = %s", out)
	}
}

func TestAzureHTTPInjectsBearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(204)
	}))
	defer srv.Close()

	a := AzureHTTP{}
	secret := []byte(`{"access_token":"eyJazure"}`)
	out, err := a.Call(context.Background(), cloudConfig(srv.URL), secret, []byte(`{"method":"POST","path":"/v1/resource","body":"{}"}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if gotAuth != "Bearer eyJazure" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["status"] != float64(204) {
		t.Fatalf("output = %s", out)
	}
}

func TestGCPHTTPRejectsMissingToken(t *testing.T) {
	g := GCPHTTP{}
	_, err := g.Call(context.Background(), cloudConfig("https://example.com"), []byte(`{"headers":{}}`), []byte(`{"method":"GET","path":"/v1/x"}`))
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected missing token to fail, got %v", err)
	}
}
