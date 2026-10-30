package policy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rejisterjack/keywarden/internal/slug"
)

const (
	Allow           = "allow"
	Deny            = "deny"
	RequireApproval = "require_approval"
	defaultTimeout  = 120 * time.Second
	maxTimeout      = 10 * time.Minute
	minTimeout      = time.Second
)

// Document is one versioned policy. No matching rule means deny.
// Engine selects the evaluator: "" or "rules" uses the ordered rule list,
// "rego" evaluates the Rego module in Rego against the call input.
type Document struct {
	Version int    `json:"version"`
	Default string `json:"default,omitempty"`
	Engine  string `json:"engine,omitempty"`
	Rego    string `json:"rego,omitempty"`
	Rules   []Rule `json:"rules"`
}

// Rule is matched in order. The first match wins.
type Rule struct {
	ID             string `json:"id"`
	Match          Match  `json:"match"`
	When           When   `json:"when,omitempty"`
	Effect         string `json:"effect"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// Match compares agent, tool, and MCP tool name. Empty fields match anything.
type Match struct {
	Agent string `json:"agent,omitempty"`
	Tool  string `json:"tool,omitempty"`
	Name  string `json:"name,omitempty"`
}

// When narrows a match with predicates known at call time.
type When struct {
	SQL string `json:"sql,omitempty"`
}

// Input is one tool call.
type Input struct {
	AgentID  string
	ToolID   string
	Name     string
	SQLClass string
}

// Decision is the effect to enforce.
type Decision struct {
	Effect  string
	RuleID  string
	Timeout time.Duration
	Reason  string
}

// Parse validates a policy document and returns it in canonical form.
func Parse(raw []byte) (Document, []byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var doc Document
	if err := dec.Decode(&doc); err != nil {
		return Document{}, nil, fmt.Errorf("policy JSON: %w", err)
	}
	if err := doc.validate(); err != nil {
		return Document{}, nil, err
	}
	canon, err := json.Marshal(doc)
	if err != nil {
		return Document{}, nil, err
	}
	return doc, canon, nil
}

func (d Document) validate() error {
	if d.Version != 1 {
		return fmt.Errorf("policy version must be 1")
	}
	if d.Default != "" && d.Default != Deny {
		return fmt.Errorf("policy default must be deny")
	}
	if d.Engine != "" && d.Engine != "rules" && d.Engine != "rego" {
		return fmt.Errorf("policy engine must be rules or rego")
	}
	if d.Engine == "rego" {
		if strings.TrimSpace(d.Rego) == "" {
			return fmt.Errorf("engine rego requires a rego module")
		}
		if len(d.Rules) > 0 {
			return fmt.Errorf("engine rego does not accept rules")
		}
		return compileRego(d.Rego)
	}
	if d.Rego != "" {
		return fmt.Errorf("rego source requires engine rego")
	}
	seen := map[string]bool{}
	for i, rule := range d.Rules {
		if !slug.OK(rule.ID) {
			return fmt.Errorf("rule %d has an invalid id", i)
		}
		if seen[rule.ID] {
			return fmt.Errorf("duplicate rule id %s", rule.ID)
		}
		seen[rule.ID] = true
		switch rule.Effect {
		case Allow, Deny, RequireApproval:
		default:
			return fmt.Errorf("rule %s has an unknown effect", rule.ID)
		}
		if rule.When.SQL != "" && rule.When.SQL != "read" && rule.When.SQL != "mutating" && rule.When.SQL != "destructive" {
			return fmt.Errorf("rule %s has an unknown sql class", rule.ID)
		}
		if rule.TimeoutSeconds != 0 && rule.Effect != RequireApproval {
			return fmt.Errorf("rule %s sets a timeout without require_approval", rule.ID)
		}
		if rule.Effect == RequireApproval && rule.TimeoutSeconds != 0 {
			if rule.TimeoutSeconds < 1 || rule.TimeoutSeconds > int(maxTimeout/time.Second) {
				return fmt.Errorf("rule %s timeout must be between 1 and 600 seconds", rule.ID)
			}
		}
		if rule.Match.Agent != "" && !slug.OK(rule.Match.Agent) {
			return fmt.Errorf("rule %s has an invalid agent", rule.ID)
		}
		if rule.Match.Tool != "" && !slug.OK(rule.Match.Tool) {
			return fmt.Errorf("rule %s has an invalid tool", rule.ID)
		}
		if rule.Match.Name != "" && len(rule.Match.Name) > 128 {
			return fmt.Errorf("rule %s tool name is too long", rule.ID)
		}
	}
	return nil
}

// Evaluate returns the first matching rule, or deny when nothing matches.
// A rego-engine document is evaluated by its compiled module instead.
func (d Document) Evaluate(in Input) Decision {
	if d.Engine == "rego" {
		return evaluateRego(d.Rego, in)
	}
	for _, rule := range d.Rules {
		if !matchField(rule.Match.Agent, in.AgentID) || !matchField(rule.Match.Tool, in.ToolID) || !matchField(rule.Match.Name, in.Name) {
			continue
		}
		if rule.When.SQL != "" && rule.When.SQL != in.SQLClass {
			continue
		}
		timeout := time.Duration(0)
		if rule.Effect == RequireApproval {
			timeout = defaultTimeout
			if rule.TimeoutSeconds > 0 {
				timeout = time.Duration(rule.TimeoutSeconds) * time.Second
			}
			if timeout < minTimeout {
				timeout = minTimeout
			}
		}
		reason := "matched rule " + rule.ID
		return Decision{Effect: rule.Effect, RuleID: rule.ID, Timeout: timeout, Reason: reason}
	}
	reason := "no matching rule"
	if len(d.Rules) == 0 {
		reason = "no active policy"
	}
	return Decision{Effect: Deny, Reason: reason}
}

// Visible reports whether any allow or approval rule could match this tool name.
// Argument predicates are ignored because they are not known at list time.
// A rego-engine document is visible when the module allows or holds the call
// with the argument predicates unknown (sql_class empty).
func (d Document) Visible(agentID, toolID, name string) bool {
	if d.Engine == "rego" {
		if visible, ok := evaluateVisible(d.Rego, Input{AgentID: agentID, ToolID: toolID, Name: name}); ok {
			return visible
		}
		decision := evaluateRego(d.Rego, Input{AgentID: agentID, ToolID: toolID, Name: name})
		return decision.Effect == Allow || decision.Effect == RequireApproval
	}
	for _, rule := range d.Rules {
		if rule.Effect != Allow && rule.Effect != RequireApproval {
			continue
		}
		if matchField(rule.Match.Agent, agentID) && matchField(rule.Match.Tool, toolID) && matchField(rule.Match.Name, name) {
			return true
		}
	}
	return false
}

func matchField(want, got string) bool {
	return want == "" || want == got
}
