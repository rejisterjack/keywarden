package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rejisterjack/keywarden/internal/redact"
	"github.com/rejisterjack/keywarden/internal/version"
)

// Stdio runs one MCP server as a child process.
// The child does not inherit the proxy environment. PATH is the only host variable copied.
type Stdio struct{}

// List starts the command, lists tools, and stops it.
func (Stdio) List(ctx context.Context, command string, args []string, env map[string]string) ([]Tool, error) {
	raw, err := run(ctx, command, args, env, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var out struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("stdio tools/list")
	}
	if out.Tools == nil {
		out.Tools = []Tool{}
	}
	return out.Tools, nil
}

// Call starts the command, calls one tool, and stops it.
func (Stdio) Call(ctx context.Context, command string, args []string, env map[string]string, name string, arguments json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		arguments = []byte(`{}`)
	}
	return run(ctx, command, args, env, "tools/call", map[string]any{
		"name":      name,
		"arguments": json.RawMessage(arguments),
	})
}

func run(ctx context.Context, command string, args []string, env map[string]string, method string, params any) (json.RawMessage, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("stdio command failed to start")
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	if err := writeMsg(stdin, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "keywarden", "version": version.Version},
	}, 1); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(stdout)
	if _, err := readResult(reader); err != nil {
		return nil, err
	}
	if err := writeMsg(stdin, "notifications/initialized", map[string]any{}, 0); err != nil {
		return nil, err
	}
	if err := writeMsg(stdin, method, params, 2); err != nil {
		return nil, err
	}
	return readResult(reader)
}

func writeMsg(w io.Writer, method string, params any, id int) error {
	payload := map[string]any{"jsonrpc": "2.0", "method": method, "params": params}
	if id > 0 {
		payload["id"] = id
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	_, err = w.Write(raw)
	return err
}

func readResult(r *bufio.Reader) (json.RawMessage, error) {
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("stdio closed")
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var decoded struct {
			ID     json.RawMessage `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *ErrorBody      `json:"error"`
		}
		if err := json.Unmarshal(line, &decoded); err != nil {
			continue
		}
		if len(decoded.ID) == 0 {
			continue
		}
		if decoded.Error != nil {
			return nil, fmt.Errorf("stdio: %s", redact.String(decoded.Error.Message))
		}
		if len(decoded.Result) == 0 {
			return nil, fmt.Errorf("stdio returned no result")
		}
		return decoded.Result, nil
	}
}
