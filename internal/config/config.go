package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is the process configuration. Secrets come from the environment,
// never from the database URL's query string in logs.
type Config struct {
	DatabaseURL string
	AdminToken  string
	VaultKey    []byte
	HTTPAddr    string
	Demo        bool
	RequireDemo bool
	AllowStdio  bool
	TrustProxy  bool
	MCPRate     float64
	MCPBurst    float64
	FixtureAddr string
	FixtureURL  string
	PublicURL   string
	Seed        bool
	DemoLock    bool
	OrdersDSN   string
}

// Load reads configuration from the environment. A local .env file fills in
// keys that are not already set.
func Load() (Config, error) {
	loadDotEnv()
	cfg := Config{
		DatabaseURL: strings.TrimSpace(os.Getenv("KEYWARDEN_DATABASE_URL")),
		AdminToken:  strings.TrimSpace(os.Getenv("KEYWARDEN_ADMIN_TOKEN")),
		HTTPAddr:    envDefault("KEYWARDEN_HTTP_ADDR", ":8080"),
		Demo:        envBool("KEYWARDEN_DEMO"),
		RequireDemo: envBool("KEYWARDEN_REQUIRE_DEMO"),
		AllowStdio:  envBool("KEYWARDEN_ALLOW_STDIO"),
		TrustProxy:  envBool("KEYWARDEN_TRUST_PROXY"),
		MCPRate:     envFloat("KEYWARDEN_MCP_RPS", 30),
		MCPBurst:    envFloat("KEYWARDEN_MCP_BURST", 60),
		FixtureAddr: envDefault("KEYWARDEN_FIXTURE_ADDR", ""),
		Seed:        envBool("KEYWARDEN_SEED"),
		DemoLock:    envBool("KEYWARDEN_DEMO_LOCK"),
		OrdersDSN:   strings.TrimSpace(os.Getenv("KEYWARDEN_ORDERS_DSN")),
	}
	cfg.PublicURL = strings.TrimRight(envDefault("KEYWARDEN_PUBLIC_URL", defaultPublicURL(cfg.HTTPAddr)), "/")
	if v := strings.TrimSpace(os.Getenv("KEYWARDEN_FIXTURE_URL")); v != "" {
		cfg.FixtureURL = strings.TrimRight(v, "/")
	} else {
		cfg.FixtureURL = cfg.PublicURL + "/demo/mcp"
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("KEYWARDEN_DATABASE_URL is required")
	}
	if len(cfg.AdminToken) < 16 {
		return Config{}, fmt.Errorf("KEYWARDEN_ADMIN_TOKEN must be at least 16 characters")
	}
	key, err := decodeKey(strings.TrimSpace(os.Getenv("KEYWARDEN_VAULT_KEY")))
	if err != nil {
		return Config{}, err
	}
	cfg.VaultKey = key
	if cfg.RequireDemo && !cfg.Demo {
		return Config{}, fmt.Errorf("KEYWARDEN_REQUIRE_DEMO is set but KEYWARDEN_DEMO is not")
	}
	if cfg.Demo && cfg.AllowStdio {
		return Config{}, fmt.Errorf("stdio tools are disabled in demo mode")
	}
	if cfg.DemoLock && !cfg.Demo {
		return Config{}, fmt.Errorf("KEYWARDEN_DEMO_LOCK requires KEYWARDEN_DEMO")
	}
	if cfg.MCPRate <= 0 || cfg.MCPBurst < 1 {
		return Config{}, fmt.Errorf("MCP rate limit must be positive")
	}
	return cfg, nil
}

func decodeKey(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, fmt.Errorf("KEYWARDEN_VAULT_KEY is required")
	}
	var key []byte
	var err error
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding} {
		key, err = enc.DecodeString(encoded)
		if err == nil {
			break
		}
	}
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("KEYWARDEN_VAULT_KEY must be 32 bytes, base64-encoded")
	}
	return key, nil
}

func envDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envFloat(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return n
}

func defaultPublicURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	return "http://" + addr
}

func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		_ = os.Setenv(key, val)
	}
}
