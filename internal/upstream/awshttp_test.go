package upstream

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/rejisterjack/keywarden/internal/secret"
)

func awsConfig(base string) []byte {
	cfg, _ := json.Marshal(map[string]any{
		"service":         "execute-api",
		"region":          "us-east-1",
		"base_url":        base,
		"allowed_methods": []string{"GET", "POST"},
		"path_prefix":     "/v1",
	})
	return cfg
}

func awsStaticSecret() []byte {
	return []byte(`{"access_key_id":"AKIAIOSFODNN7EXAMPLE","secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`)
}

func TestAWSHTTPSignsWithStaticCredentials(t *testing.T) {
	var gotAuth, gotDate, gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotDate = r.Header.Get("X-Amz-Date")
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	a := AWSHTTP{}
	out, err := a.Call(context.Background(), awsConfig(srv.URL), awsStaticSecret(), []byte(`{"method":"GET","path":"/v1/items"}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.HasPrefix(gotAuth, "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/") {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if !strings.Contains(gotAuth, "SignedHeaders=") || !strings.Contains(gotAuth, "Signature=") {
		t.Fatalf("Authorization missing parts: %q", gotAuth)
	}
	if gotDate == "" {
		t.Fatal("missing X-Amz-Date")
	}
	if gotMethod != "GET" || gotPath != "/v1/items" {
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

func TestAWSHTTPAssumeRoleUsesSessionCredentials(t *testing.T) {
	var gotAuth, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotToken = r.Header.Get("X-Amz-Security-Token")
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	a := AWSHTTP{
		AssumeRole: func(ctx context.Context, keys secret.AWSSecret, cfg secret.AWSConfig) (aws.Credentials, error) {
			if keys.RoleArn == "" {
				t.Fatal("expected role_arn")
			}
			return aws.Credentials{
				AccessKeyID:     "ASIASESSIONKEY",
				SecretAccessKey: "session-secret",
				SessionToken:    "session-token-value",
				Source:          "AssumeRole",
				CanExpire:       true,
				Expires:         time.Now().Add(time.Hour),
			}, nil
		},
	}
	secretRaw := []byte(`{
		"access_key_id":"AKIAIOSFODNN7EXAMPLE",
		"secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"role_arn":"arn:aws:iam::123456789012:role/KeywardenDemo"
	}`)
	if _, err := a.Call(context.Background(), awsConfig(srv.URL), secretRaw, []byte(`{"method":"POST","path":"/v1/items","body":"{}"}`)); err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(gotAuth, "Credential=ASIASESSIONKEY/") {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotToken != "session-token-value" {
		t.Fatalf("X-Amz-Security-Token = %q", gotToken)
	}
}

func TestAWSHTTPRejectsDisallowedMethod(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()
	a := AWSHTTP{}
	_, err := a.Call(context.Background(), awsConfig(srv.URL), awsStaticSecret(), []byte(`{"method":"DELETE","path":"/v1/x"}`))
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected method rejection, got %v", err)
	}
	if called {
		t.Fatal("disallowed method reached upstream")
	}
}

func TestAWSHTTPListSingleTool(t *testing.T) {
	a := AWSHTTP{}
	tools, err := a.List(context.Background(), awsConfig("https://example.execute-api.us-east-1.amazonaws.com"), awsStaticSecret())
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Name != "request" {
		t.Fatalf("tools = %+v", tools)
	}
}
