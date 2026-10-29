package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rejisterjack/keywarden/internal/redact"
	"github.com/rejisterjack/keywarden/internal/version"
)

// Client speaks Streamable HTTP MCP to upstream servers.
type Client struct {
	HTTP     *http.Client
	mu       sync.Mutex
	sessions map[string]string
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// List returns the upstream tool list.
func (c *Client) List(ctx context.Context, endpoint string, headers map[string]string) ([]Tool, error) {
	body, err := c.call(ctx, endpoint, headers, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var out struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("upstream tools/list")
	}
	if out.Tools == nil {
		out.Tools = []Tool{}
	}
	return out.Tools, nil
}

// Call invokes one upstream tool and returns the raw result JSON.
func (c *Client) Call(ctx context.Context, endpoint string, headers map[string]string, name string, args json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(args)) == 0 {
		args = []byte(`{}`)
	}
	return c.call(ctx, endpoint, headers, "tools/call", map[string]any{
		"name":      name,
		"arguments": json.RawMessage(args),
	})
}

func (c *Client) call(ctx context.Context, endpoint string, headers map[string]string, method string, params any) (json.RawMessage, error) {
	session, err := c.session(ctx, endpoint, headers)
	if err != nil {
		return nil, err
	}
	return c.roundTrip(ctx, endpoint, headers, session, method, params, false)
}

func (c *Client) session(ctx context.Context, endpoint string, headers map[string]string) (string, error) {
	c.mu.Lock()
	if c.sessions == nil {
		c.sessions = map[string]string{}
	}
	if id := c.sessions[endpoint]; id != "" {
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()
	_, session, err := c.post(ctx, endpoint, headers, "", "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "keywarden", "version": version.Version},
	}, false)
	if err != nil {
		return "", err
	}
	if session == "" {
		session = "stateless"
	}
	_, _, err = c.post(ctx, endpoint, headers, session, "notifications/initialized", map[string]any{}, true)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.sessions[endpoint] = session
	c.mu.Unlock()
	return session, nil
}

func (c *Client) roundTrip(ctx context.Context, endpoint string, headers map[string]string, session, method string, params any, note bool) (json.RawMessage, error) {
	raw, _, err := c.post(ctx, endpoint, headers, session, method, params, note)
	return raw, err
}

func (c *Client) post(ctx context.Context, endpoint string, headers map[string]string, session, method string, params any, note bool) (json.RawMessage, string, error) {
	payload := map[string]any{"jsonrpc": "2.0", "method": method, "params": params}
	if !note {
		payload["id"] = 1
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	if session != "" && session != "stateless" {
		req.Header.Set("Mcp-Session-Id", session)
	}
	for k, v := range headers {
		if strings.EqualFold(k, "host") || strings.EqualFold(k, "content-length") {
			continue
		}
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("upstream unavailable")
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, "", fmt.Errorf("upstream read failed")
	}
	if note && (resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent) {
		return nil, resp.Header.Get("Mcp-Session-Id"), nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	msg := respBody
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		msg, err = lastEvent(respBody)
		if err != nil {
			return nil, "", err
		}
	}
	var decoded struct {
		Result json.RawMessage `json:"result"`
		Error  *ErrorBody      `json:"error"`
	}
	if err := json.Unmarshal(msg, &decoded); err != nil {
		return nil, "", fmt.Errorf("upstream response")
	}
	if decoded.Error != nil {
		return nil, "", fmt.Errorf("upstream: %s", redact.String(decoded.Error.Message))
	}
	if len(decoded.Result) == 0 {
		return nil, "", fmt.Errorf("upstream returned no result")
	}
	return decoded.Result, resp.Header.Get("Mcp-Session-Id"), nil
}

func lastEvent(body []byte) (json.RawMessage, error) {
	var last []byte
	for _, line := range bytes.Split(body, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if bytes.HasPrefix(line, []byte("data:")) {
			data := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
			if len(data) > 0 && !bytes.Equal(data, []byte("[DONE]")) {
				last = data
			}
		}
	}
	if len(last) == 0 {
		return nil, fmt.Errorf("upstream event stream was empty")
	}
	return last, nil
}
