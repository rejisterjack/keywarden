package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func task(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: keywarden task create|revoke")
	}
	switch args[0] {
	case "create":
		return taskCreate(args[1:])
	case "revoke":
		if len(args) != 2 {
			return fmt.Errorf("usage: keywarden task revoke <task-id>")
		}
		return api(http.MethodPost, "/v1/tasks/"+args[1]+"/revoke", nil, nil)
	default:
		return fmt.Errorf("unknown task command %q", args[0])
	}
}

func taskCreate(args []string) error {
	agent := ""
	ttl := ""
	var tools []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			i++
			if i >= len(args) {
				return fmt.Errorf("--agent requires a value")
			}
			agent = args[i]
		case "--ttl":
			i++
			if i >= len(args) {
				return fmt.Errorf("--ttl requires a value")
			}
			ttl = args[i]
		case "--tool":
			i++
			if i >= len(args) {
				return fmt.Errorf("--tool requires a value")
			}
			tools = append(tools, args[i])
		default:
			return fmt.Errorf("unknown argument %q", args[i])
		}
	}
	if agent == "" {
		return fmt.Errorf("--agent is required")
	}
	body := map[string]any{"agent_id": agent}
	if ttl != "" {
		d, err := time.ParseDuration(ttl)
		if err != nil {
			return fmt.Errorf("--ttl: %w", err)
		}
		body["ttl_seconds"] = int(d.Seconds())
	}
	if len(tools) > 0 {
		body["tool_ids"] = tools
	}
	var out map[string]any
	if err := api(http.MethodPost, "/v1/tasks", body, &out); err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func policyCmd(args []string) error {
	if len(args) != 2 || args[0] != "apply" {
		return fmt.Errorf("usage: keywarden policy apply <file>")
	}
	raw, err := os.ReadFile(args[1])
	if err != nil {
		return err
	}
	var out map[string]any
	if err := apiRaw(http.MethodPost, "/v1/policies", raw, &out); err != nil {
		return err
	}
	fmt.Printf("applied policy version %v\n", out["version"])
	return nil
}

func auditCmd(args []string) error {
	if len(args) != 1 || args[0] != "verify" {
		return fmt.Errorf("usage: keywarden audit verify")
	}
	var out struct {
		OK     bool   `json:"ok"`
		Events int    `json:"events"`
		Reason string `json:"reason"`
	}
	if err := api(http.MethodGet, "/v1/audit/verify", nil, &out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("audit chain broken after %d events: %s", out.Events, out.Reason)
	}
	fmt.Printf("audit chain ok (%d events)\n", out.Events)
	return nil
}

func api(method, path string, body any, dst any) error {
	var raw []byte
	var err error
	if body != nil {
		raw, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	return apiRaw(method, path, raw, dst)
}

func apiRaw(method, path string, raw []byte, dst any) error {
	base := strings.TrimRight(os.Getenv("KEYWARDEN_URL"), "/")
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	token := os.Getenv("KEYWARDEN_ADMIN_TOKEN")
	if token == "" {
		return fmt.Errorf("KEYWARDEN_ADMIN_TOKEN is required")
	}
	var r io.Reader
	if raw != nil {
		r = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, base+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if raw != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	resp, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %s", method, path, strings.TrimSpace(string(resp)))
	}
	if dst == nil || res.StatusCode == http.StatusNoContent || len(bytes.TrimSpace(resp)) == 0 {
		return nil
	}
	return json.Unmarshal(resp, dst)
}
