package config

import (
	"encoding/base64"
	"testing"
)

func TestListenAddressFollowsPort(t *testing.T) {
	t.Setenv("PORT", "10000")
	t.Setenv("KEYWARDEN_HTTP_ADDR", "")
	t.Setenv("KEYWARDEN_FIXTURE_URL", "")
	t.Setenv("KEYWARDEN_DATABASE_URL", "postgres://example")
	t.Setenv("KEYWARDEN_ADMIN_TOKEN", "0123456789abcdef")
	t.Setenv("KEYWARDEN_VAULT_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != "0.0.0.0:10000" {
		t.Fatalf("addr %q", cfg.HTTPAddr)
	}
	if cfg.FixtureURL != "http://127.0.0.1:10000/demo/mcp" {
		t.Fatalf("fixture %q", cfg.FixtureURL)
	}
}

func TestExplicitListenAddressWins(t *testing.T) {
	t.Setenv("PORT", "10000")
	t.Setenv("KEYWARDEN_HTTP_ADDR", ":8080")
	t.Setenv("KEYWARDEN_FIXTURE_URL", "http://127.0.0.1:8080/demo/mcp")
	t.Setenv("KEYWARDEN_DATABASE_URL", "postgres://example")
	t.Setenv("KEYWARDEN_ADMIN_TOKEN", "0123456789abcdef")
	t.Setenv("KEYWARDEN_VAULT_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("addr %q", cfg.HTTPAddr)
	}
}
