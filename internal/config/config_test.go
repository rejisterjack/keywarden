package config

import (
	"encoding/base64"
	"testing"
)

func TestLoadRequiresSecrets(t *testing.T) {
	t.Setenv("KEYWARDEN_DATABASE_URL", "")
	t.Setenv("KEYWARDEN_ADMIN_TOKEN", "")
	t.Setenv("KEYWARDEN_VAULT_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadAcceptsVaultKey(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	t.Setenv("KEYWARDEN_DATABASE_URL", "postgres://localhost/keywarden")
	t.Setenv("KEYWARDEN_ADMIN_TOKEN", "local-dev-admin-token")
	t.Setenv("KEYWARDEN_VAULT_KEY", base64.StdEncoding.EncodeToString(key))
	t.Setenv("KEYWARDEN_DEMO", "1")
	t.Setenv("KEYWARDEN_ALLOW_STDIO", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Demo {
		t.Fatal("demo flag")
	}
	if !cfg.SingleTenant {
		t.Fatal("single tenant should default on")
	}
	if len(cfg.VaultKey) != 32 || cfg.VaultKey[0] != 1 {
		t.Fatalf("key = %v", cfg.VaultKey)
	}
}

func TestRequireDemoWithoutDemoFails(t *testing.T) {
	key := make([]byte, 32)
	t.Setenv("KEYWARDEN_DATABASE_URL", "postgres://localhost/keywarden")
	t.Setenv("KEYWARDEN_ADMIN_TOKEN", "local-dev-admin-token")
	t.Setenv("KEYWARDEN_VAULT_KEY", base64.StdEncoding.EncodeToString(key))
	t.Setenv("KEYWARDEN_DEMO", "")
	t.Setenv("KEYWARDEN_REQUIRE_DEMO", "1")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}

func TestDemoRejectsStdio(t *testing.T) {
	key := make([]byte, 32)
	t.Setenv("KEYWARDEN_DATABASE_URL", "postgres://localhost/keywarden")
	t.Setenv("KEYWARDEN_ADMIN_TOKEN", "local-dev-admin-token")
	t.Setenv("KEYWARDEN_VAULT_KEY", base64.StdEncoding.EncodeToString(key))
	t.Setenv("KEYWARDEN_DEMO", "1")
	t.Setenv("KEYWARDEN_REQUIRE_DEMO", "")
	t.Setenv("KEYWARDEN_ALLOW_STDIO", "1")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}
