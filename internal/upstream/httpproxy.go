package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rejisterjack/keywarden/internal/fail"
	"github.com/rejisterjack/keywarden/internal/mcp"
	"github.com/rejisterjack/keywarden/internal/secret"
)

// httpMaxBody caps request and response bodies at 1 MiB.
const httpMaxBody = 1 << 20

// plainHTTPToolSchema is the single MCP tool an http upstream exposes.
const plainHTTPToolSchema = `{"type":"object","properties":{"method":{"type":"string","enum":["GET","POST","PUT","PATCH","DELETE","HEAD"]},"path":{"type":"string"},"headers":{"type":"object","additionalProperties":{"type":"string"}},"body":{"type":"string"}},"required":["method","path"],"additionalProperties":false}`

// PlainHTTP forwards one governed call to a plain HTTP API.
type PlainHTTP struct {
	// HTTP is injectable for tests.
	HTTP *http.Client
}

// List returns the single request tool.
func (p PlainHTTP) List(ctx context.Context, config, secretRaw []byte) ([]mcp.Tool, error) {
	if _, _, err := secret.PlainHTTP(config, secretRaw); err != nil {
		return nil, err
	}
	return []mcp.Tool{{
		Name:        "request",
		Description: "Call the configured HTTP API. The real credentials stay in Keywarden.",
		InputSchema: json.RawMessage(plainHTTPToolSchema),
	}}, nil
}

// Call executes one request. Method allow-list, path prefix, and the sealed
// headers are enforced here, before any dial.
func (p PlainHTTP) Call(ctx context.Context, config, secretRaw, args []byte) (string, error) {
	cfg, sec, err := secret.PlainHTTP(config, secretRaw)
	if err != nil {
		return "", err
	}
	var body struct {
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Headers map[string]string `json:"headers,omitempty"`
		Body    string            `json:"body,omitempty"`
	}
	if err := strictUnmarshal(args, &body); err != nil {
		return "", err
	}
	if body.Method == "" || body.Path == "" {
		return "", fail.New(400, "request requires method and path")
	}
	method := strings.ToUpper(body.Method)
	if !methodAllowed(cfg.AllowedMethods, method) {
		return "", fail.New(400, "method is not allowed for this tool")
	}
	target, err := joinPath(cfg.BaseURL, cfg.PathPrefix, body.Path)
	if err != nil {
		return "", err
	}
	var payload io.Reader
	if body.Body != "" {
		if len(body.Body) > httpMaxBody {
			return "", fail.New(400, "request body is too large")
		}
		payload = strings.NewReader(body.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, payload)
	if err != nil {
		return "", fail.New(400, "request is invalid")
	}
	for k, v := range sec.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range body.Headers {
		if strings.EqualFold(k, "Host") || strings.EqualFold(k, "Content-Length") {
			return "", fail.New(400, "request headers cannot override host or length")
		}
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "keywarden")
	}
	client := p.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upstream unavailable")
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, httpMaxBody+1))
	if err != nil {
		return "", fmt.Errorf("upstream unavailable")
	}
	if len(raw) > httpMaxBody {
		return "", fail.New(400, "response body is too large")
	}
	out := map[string]any{
		"status": res.StatusCode,
		"body":   string(raw),
	}
	enc, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(enc), nil
}

func methodAllowed(allowed []string, method string) bool {
	for _, m := range allowed {
		if strings.ToUpper(m) == method {
			return true
		}
	}
	return false
}

func joinPath(baseURL, prefix, callPath string) (string, error) {
	if !strings.HasPrefix(callPath, "/") {
		callPath = "/" + callPath
	}
	if strings.Contains(callPath, "..") || strings.ContainsAny(callPath, "?#") {
		// Query strings travel in the path the agent chose; split them off
		// before the prefix check and reattach after.
		u, err := url.Parse(callPath)
		if err != nil {
			return "", fail.New(400, "path is invalid")
		}
		if strings.Contains(u.Path, "..") {
			return "", fail.New(400, "path must not traverse")
		}
		callPath = u.Path
		if u.RawQuery != "" {
			callPath += "?" + u.RawQuery
		}
	}
	pathOnly, query, _ := strings.Cut(callPath, "?")
	if prefix != "" && !strings.HasPrefix(pathOnly, prefix) {
		return "", fail.New(400, "path is outside the configured prefix")
	}
	trimmed := strings.TrimSuffix(baseURL, "/") + pathOnly
	if query != "" {
		trimmed += "?" + query
	}
	return trimmed, nil
}

func strictUnmarshal(raw []byte, dst any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fail.New(400, "arguments must match the request schema")
	}
	return nil
}
