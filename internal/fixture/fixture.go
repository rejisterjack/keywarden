package fixture

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/rejisterjack/keywarden/internal/mcp"
)

// Handler is a fake MCP server with three catalog tools and no real credentials.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.Error(w, "sse is not offered", http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeErr(w, nil, -32700, "invalid request")
			return
		}
		req, err := mcp.ParseRequest(raw)
		if err != nil {
			writeErr(w, nil, -32700, "invalid request")
			return
		}
		if req.IsNotification() {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Mcp-Session-Id", "demo")
		switch req.Method {
		case "initialize":
			writeOK(w, req.ID, map[string]any{
				"protocolVersion": "2025-03-26",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "keywarden-fixture", "version": "0.1.0"},
			})
		case "ping":
			writeOK(w, req.ID, map[string]any{})
		case "tools/list":
			writeOK(w, req.ID, map[string]any{"tools": []mcp.Tool{
				{Name: "lookup_order", Description: "Look up a fictional order.", InputSchema: obj("order_id")},
				{Name: "refund_order", Description: "Refund a fictional order.", InputSchema: obj("order_id")},
				{Name: "delete_catalog", Description: "Destroy the fictional catalog.", InputSchema: json.RawMessage(`{"type":"object"}`)},
			}})
		case "tools/call":
			writeOK(w, req.ID, call(req.Params))
		default:
			writeErr(w, req.ID, -32601, "method not found")
		}
	})
}

func call(params json.RawMessage) mcp.ToolResult {
	var body struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	_ = json.Unmarshal(params, &body)
	var args struct {
		OrderID string `json:"order_id"`
	}
	_ = json.Unmarshal(body.Arguments, &args)
	switch body.Name {
	case "lookup_order":
		if args.OrderID == "" {
			args.OrderID = "1042"
		}
		raw, _ := json.Marshal(map[string]string{"order_id": args.OrderID, "status": "shipped", "customer": "Ada Lovelace"})
		return mcp.TextResult(string(raw), false)
	case "refund_order":
		raw, _ := json.Marshal(map[string]string{"order_id": args.OrderID, "status": "refunded"})
		return mcp.TextResult(string(raw), false)
	case "delete_catalog":
		return mcp.TextResult(`{"deleted":true}`, false)
	default:
		return mcp.TextResult("unknown tool", true)
	}
}

func obj(field string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			field: map[string]string{"type": "string"},
		},
		"required": []string{field},
	})
	return raw
}

func writeOK(w http.ResponseWriter, id json.RawMessage, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mcp.Result{JSONRPC: "2.0", ID: id, Result: result})
}

func writeErr(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	if id == nil {
		id = json.RawMessage(`null`)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mcp.Failure{JSONRPC: "2.0", ID: id, Error: mcp.ErrorBody{Code: code, Message: msg}})
}
