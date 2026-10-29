package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Tool is one MCP tool definition.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Request is a JSON-RPC 2.0 message.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IsNotification reports whether the request must not receive a body.
func (r Request) IsNotification() bool {
	return len(bytes.TrimSpace(r.ID)) == 0 || string(r.ID) == "null"
}

// Result is a JSON-RPC success.
type Result struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result"`
}

// Failure is a JSON-RPC error.
type Failure struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   ErrorBody       `json:"error"`
}

// ErrorBody is the JSON-RPC error object.
type ErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolResult is the MCP tools/call result.
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError"`
}

// Content is one text block returned to the agent.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// TextResult builds a tools/call result.
func TextResult(text string, isError bool) ToolResult {
	return ToolResult{Content: []Content{{Type: "text", Text: text}}, IsError: isError}
}

// ParseRequest decodes one JSON-RPC request and rejects batches.
func ParseRequest(raw []byte) (Request, error) {
	trim := bytes.TrimSpace(raw)
	if len(trim) == 0 {
		return Request{}, fmt.Errorf("empty request")
	}
	if trim[0] == '[' {
		return Request{}, fmt.Errorf("batches are not supported")
	}
	var req Request
	if err := json.Unmarshal(trim, &req); err != nil {
		return Request{}, err
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return Request{}, fmt.Errorf("invalid json-rpc request")
	}
	return req, nil
}
