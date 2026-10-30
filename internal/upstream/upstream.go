package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/rejisterjack/keywarden/internal/fail"
	"github.com/rejisterjack/keywarden/internal/mcp"
	"github.com/rejisterjack/keywarden/internal/policy"
	"github.com/rejisterjack/keywarden/internal/redact"
	"github.com/rejisterjack/keywarden/internal/secret"
	"github.com/rejisterjack/keywarden/internal/store"
)

// Dispatcher forwards an already-authorized call to the upstream.
type Dispatcher struct {
	HTTP       *mcp.Client
	Stdio      mcp.Stdio
	Plain      PlainHTTP
	AWS        AWSHTTP
	Connect    func(ctx context.Context, dsn string) (*pgx.Conn, error)
	AllowStdio bool
}

// List returns upstream tool definitions.
func (d *Dispatcher) List(ctx context.Context, tool store.Tool, secretRaw []byte) ([]mcp.Tool, error) {
	switch tool.Kind {
	case "postgres":
		return []mcp.Tool{{
			Name:        "query",
			Description: "Run one SQL statement. Keywarden classifies it before it reaches the database.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"sql":{"type":"string"}},"required":["sql"],"additionalProperties":false}`),
		}}, nil
	case "mcp_http":
		sec, cfg, err := httpParts(tool.Config, secretRaw)
		if err != nil {
			return nil, err
		}
		return d.http().List(ctx, cfg.URL, sec.Headers)
	case "mcp_stdio":
		if !d.AllowStdio {
			return nil, fail.New(400, "stdio tools are disabled")
		}
		sec, cfg, err := stdioParts(tool.Config, secretRaw)
		if err != nil {
			return nil, err
		}
		return d.Stdio.List(ctx, cfg.Command, cfg.Args, sec.Env)
	case "http":
		return d.Plain.List(ctx, tool.Config, secretRaw)
	case "aws_http":
		return d.AWS.List(ctx, tool.Config, secretRaw)
	default:
		return nil, fail.New(400, "unknown tool kind")
	}
}

// Call executes one tool. Postgres connections are opened here, not before.
func (d *Dispatcher) Call(ctx context.Context, tool store.Tool, secretRaw []byte, name string, args json.RawMessage) (string, error) {
	switch tool.Kind {
	case "postgres":
		if name != "query" {
			return "", fail.New(404, "unknown tool")
		}
		return d.query(ctx, tool, secretRaw, args)
	case "mcp_http":
		sec, cfg, err := httpParts(tool.Config, secretRaw)
		if err != nil {
			return "", err
		}
		raw, err := d.http().Call(ctx, cfg.URL, sec.Headers, name, args)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	case "mcp_stdio":
		if !d.AllowStdio {
			return "", fail.New(400, "stdio tools are disabled")
		}
		sec, cfg, err := stdioParts(tool.Config, secretRaw)
		if err != nil {
			return "", err
		}
		raw, err := d.Stdio.Call(ctx, cfg.Command, cfg.Args, sec.Env, name, args)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	case "http":
		if name != "request" {
			return "", fail.New(404, "unknown tool")
		}
		return d.Plain.Call(ctx, tool.Config, secretRaw, args)
	case "aws_http":
		if name != "request" {
			return "", fail.New(404, "unknown tool")
		}
		return d.AWS.Call(ctx, tool.Config, secretRaw, args)
	default:
		return "", fail.New(400, "unknown tool kind")
	}
}

func (d *Dispatcher) http() *mcp.Client {
	if d.HTTP != nil {
		return d.HTTP
	}
	return &mcp.Client{}
}

func (d *Dispatcher) query(ctx context.Context, tool store.Tool, secretRaw, args []byte) (string, error) {
	sec, err := secret.Parse(secretRaw)
	if err != nil {
		return "", err
	}
	sqlText, err := sqlArg(args)
	if err != nil {
		return "", err
	}
	if policy.Statements(sqlText) != 1 {
		return "", fail.New(400, "postgres tool accepts one statement")
	}
	connect := d.Connect
	if connect == nil {
		connect = pgx.Connect
	}
	conn, err := connect(ctx, sec.DSN)
	if err != nil {
		return "", fmt.Errorf("database unavailable")
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "SET statement_timeout = '5s'"); err != nil {
		return "", fmt.Errorf("database unavailable")
	}
	rows, err := conn.Query(ctx, sqlText)
	if err != nil {
		return "", fmt.Errorf("query failed: %s", redact.String(err.Error()))
	}
	defer rows.Close()
	fields := rows.FieldDescriptions()
	cols := make([]string, len(fields))
	for i, f := range fields {
		cols[i] = f.Name
	}
	limit := secret.MaxRows(tool.Config)
	out := make([][]any, 0)
	truncated := false
	var used int
	for rows.Next() {
		if len(out) >= limit {
			truncated = true
			break
		}
		vals, err := rows.Values()
		if err != nil {
			return "", fmt.Errorf("query failed")
		}
		row := make([]any, len(vals))
		for i, v := range vals {
			row[i] = jsonValue(v)
		}
		raw, err := json.Marshal(row)
		if err != nil {
			return "", fmt.Errorf("query failed")
		}
		if used+len(raw) > 256*1024 {
			truncated = true
			break
		}
		used += len(raw)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("query failed: %s", redact.String(err.Error()))
	}
	tag := rows.CommandTag()
	payload := map[string]any{
		"columns":       cols,
		"rows":          out,
		"row_count":     len(out),
		"rows_affected": tag.RowsAffected(),
		"command":       tag.String(),
		"truncated":     truncated,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("query failed")
	}
	return string(raw), nil
}

func sqlArg(args []byte) (string, error) {
	var body struct {
		SQL string `json:"sql"`
	}
	if err := json.Unmarshal(args, &body); err != nil {
		return "", fail.New(400, "arguments must be a JSON object")
	}
	if len(body.SQL) == 0 || len(body.SQL) > 10000 {
		return "", fail.New(400, "sql is required")
	}
	return body.SQL, nil
}

func jsonValue(v any) any {
	switch t := v.(type) {
	case []byte:
		if utf8.Valid(t) {
			return string(t)
		}
		return fmt.Sprintf("%x", t)
	default:
		return v
	}
}

func httpParts(config, secretRaw []byte) (secret.Secret, secret.HTTPConfig, error) {
	sec, err := secret.Parse(secretRaw)
	if err != nil {
		return secret.Secret{}, secret.HTTPConfig{}, err
	}
	var cfg secret.HTTPConfig
	if err := json.Unmarshal(config, &cfg); err != nil || cfg.URL == "" {
		return secret.Secret{}, secret.HTTPConfig{}, fail.New(500, "tool config is invalid")
	}
	return sec, cfg, nil
}

func stdioParts(config, secretRaw []byte) (secret.Secret, secret.StdioConfig, error) {
	sec, err := secret.Parse(secretRaw)
	if err != nil {
		return secret.Secret{}, secret.StdioConfig{}, err
	}
	var cfg secret.StdioConfig
	if err := json.Unmarshal(config, &cfg); err != nil || cfg.Command == "" {
		return secret.Secret{}, secret.StdioConfig{}, fail.New(500, "tool config is invalid")
	}
	return sec, cfg, nil
}
