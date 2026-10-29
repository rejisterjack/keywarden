package secret

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/rejisterjack/keywarden/internal/fail"
)

// Secret is the sealed JSON stored for a tool. It is never returned by the API.
type Secret struct {
	KWDemo  string            `json:"kw_demo,omitempty"`
	DSN     string            `json:"dsn,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// HTTPConfig is the public configuration of an mcp_http tool.
type HTTPConfig struct {
	URL string `json:"url"`
}

// PostgresConfig is the public configuration of a postgres tool.
type PostgresConfig struct {
	MaxRows int `json:"max_rows,omitempty"`
}

// StdioConfig is the public configuration of an mcp_stdio tool.
type StdioConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

var (
	envKey  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	command = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)
	maxRows = 500
)

// Parse decodes a secret document.
func Parse(raw []byte) (Secret, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return Secret{}, fail.New(400, "secret is required")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var sec Secret
	if err := dec.Decode(&sec); err != nil {
		return Secret{}, fail.New(400, "secret must be a JSON object")
	}
	return sec, nil
}

// IsFake reports whether the secret is part of the demo set.
func IsFake(raw []byte) bool {
	sec, err := Parse(raw)
	return err == nil && sec.KWDemo == "fake"
}

// Validate checks a tool config and secret. In demo mode the secret must be
// marked fake, and the upstream must be the configured fixture or orders database.
func Validate(kind string, config, secretRaw []byte, demo bool, fixtureURL, ordersDSN string) error {
	sec, err := Parse(secretRaw)
	if err != nil {
		return err
	}
	if demo && sec.KWDemo != "fake" {
		return fail.New(400, "demo mode only accepts secrets marked kw_demo=fake")
	}
	switch kind {
	case "mcp_http":
		return validateHTTP(config, sec, demo, fixtureURL)
	case "postgres":
		return validatePostgres(config, sec, demo, ordersDSN)
	case "mcp_stdio":
		if demo {
			return fail.New(400, "stdio tools are disabled in demo mode")
		}
		return validateStdio(config, sec)
	default:
		return fail.New(400, "unknown tool kind")
	}
}

func validateHTTP(config []byte, sec Secret, demo bool, fixtureURL string) error {
	var cfg HTTPConfig
	if err := decodeConfig(config, &cfg); err != nil {
		return err
	}
	u, err := url.Parse(cfg.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fail.New(400, "mcp_http config url must be http or https")
	}
	if u.User != nil {
		return fail.New(400, "mcp_http url must not contain credentials")
	}
	if demo {
		if err := demoHeaders(sec.Headers); err != nil {
			return err
		}
		if !sameEndpoint(cfg.URL, fixtureURL) {
			return fail.New(400, "demo mode only accepts the seeded MCP fixture")
		}
	}
	return nil
}

func validatePostgres(config []byte, sec Secret, demo bool, ordersDSN string) error {
	var cfg PostgresConfig
	if len(bytes.TrimSpace(config)) == 0 {
		config = []byte(`{}`)
	}
	if err := decodeConfig(config, &cfg); err != nil {
		return err
	}
	if cfg.MaxRows < 0 || cfg.MaxRows > maxRows {
		return fail.New(400, "max_rows must be between 1 and 500")
	}
	if strings.TrimSpace(sec.DSN) == "" {
		return fail.New(400, "postgres secret requires dsn")
	}
	if _, err := dsnParts(sec.DSN); err != nil {
		return fail.New(400, "postgres dsn is invalid")
	}
	if demo {
		if ordersDSN == "" {
			return fail.New(400, "demo mode has no orders database configured")
		}
		if !sameDatabase(sec.DSN, ordersDSN) {
			return fail.New(400, "demo mode only accepts the seeded orders database")
		}
	}
	if sec.DSN != "" && len(sec.Headers) > 0 {
		return fail.New(400, "postgres secret does not take headers")
	}
	return nil
}

func validateStdio(config []byte, sec Secret) error {
	var cfg StdioConfig
	if err := decodeConfig(config, &cfg); err != nil {
		return err
	}
	if !command.MatchString(cfg.Command) {
		return fail.New(400, "stdio command must be a bare executable name or path")
	}
	if len(cfg.Args) > 32 {
		return fail.New(400, "stdio tool has too many arguments")
	}
	for _, arg := range cfg.Args {
		if len(arg) == 0 || len(arg) > 256 {
			return fail.New(400, "stdio arguments must be 1 to 256 characters")
		}
	}
	for k := range sec.Env {
		if !envKey.MatchString(k) || strings.EqualFold(k, "PATH") {
			return fail.New(400, "stdio env keys must be identifiers and cannot replace PATH")
		}
	}
	return nil
}

func demoHeaders(h map[string]string) error {
	for _, v := range h {
		if v != "" && v != "fake" {
			return fail.New(400, "demo mode header values must be empty or fake")
		}
	}
	return nil
}

func decodeConfig(raw []byte, dst any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fail.New(400, "config is required")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fail.New(400, "config must be a JSON object")
	}
	return nil
}

func sameEndpoint(a, b string) bool {
	ua, err1 := url.Parse(a)
	ub, err2 := url.Parse(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(ua.Hostname(), ub.Hostname()) && ua.Port() == ub.Port()
}

func sameDatabase(a, b string) bool {
	pa, err1 := dsnParts(a)
	pb, err2 := dsnParts(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return pa == pb
}

type parts struct {
	user string
	host string
	port string
	db   string
}

func dsnParts(dsn string) (parts, error) {
	u, err := url.Parse(dsn)
	if err != nil || u.Host == "" || u.User == nil || u.User.Username() == "" {
		return parts{}, fmt.Errorf("dsn")
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	db := strings.TrimPrefix(u.Path, "/")
	if db == "" {
		return parts{}, fmt.Errorf("database")
	}
	return parts{user: u.User.Username(), host: strings.ToLower(u.Hostname()), port: port, db: db}, nil
}

// MaxRows returns the configured cap, or 200.
func MaxRows(config []byte) int {
	var cfg PostgresConfig
	if err := decodeConfig(config, &cfg); err != nil || cfg.MaxRows == 0 {
		return 200
	}
	return cfg.MaxRows
}
