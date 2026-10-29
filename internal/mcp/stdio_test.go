package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "stdio-fixture" {
		runStdioFixture()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestStdioDoesNotInheritProxyEnv(t *testing.T) {
	const leak = "keywarden-stdio-leak-value"
	t.Setenv("KEYWARDEN_VAULT_KEY", leak)
	ctx := context.Background()
	raw, err := (Stdio{}).Call(ctx, os.Args[0], []string{"stdio-fixture"}, map[string]string{"FIXTURE_TOKEN": "fake"}, "env", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(leak)) {
		t.Fatalf("child saw the proxy environment: %s", raw)
	}
	if !bytes.Contains(raw, []byte("FIXTURE_TOKEN=fake")) {
		t.Fatalf("child missed its own env: %s", raw)
	}
}

func runStdioFixture() {
	in := bufio.NewReader(os.Stdin)
	for {
		line, err := in.ReadBytes('\n')
		if err != nil {
			return
		}
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(line, &req); err != nil || len(req.ID) == 0 || string(req.ID) == "null" {
			continue
		}
		switch req.Method {
		case "initialize":
			writeLine(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{
				"protocolVersion": "2025-03-26",
				"capabilities":    map[string]any{},
				"serverInfo":      map[string]any{"name": "fixture", "version": "0"},
			}})
		case "tools/call":
			var params struct {
				Name string `json:"name"`
			}
			_ = json.Unmarshal(req.Params, &params)
			text := "ok"
			if params.Name == "env" {
				text = strings.Join(os.Environ(), "\n")
			}
			writeLine(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": TextResult(text, false)})
		default:
			writeLine(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}})
		}
	}
}

func writeLine(v any) {
	raw, _ := json.Marshal(v)
	raw = append(raw, '\n')
	_, _ = os.Stdout.Write(raw)
}
