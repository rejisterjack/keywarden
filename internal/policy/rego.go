package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-policy-agent/opa/rego"
)

// regoTimeout bounds one policy evaluation. A slow module denies the call
// rather than stalling the data plane.
const regoTimeout = 2 * time.Second

// compileRego parses a module so invalid policies fail at apply time.
func compileRego(module string) error {
	_, err := rego.New(
		rego.Query("data.keywarden.decision"),
		rego.Module("policy.rego", module),
	).PrepareForEval(context.Background())
	return err
}

// evaluateVisible runs data.keywarden.visible when the module defines it.
// That rule answers tools/list visibility with the argument predicates
// unknown. Modules without it fall back to evaluateRego with sql_class empty.
func evaluateVisible(module string, in Input) (bool, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), regoTimeout)
	defer cancel()
	q, err := rego.New(
		rego.Query("data.keywarden.visible"),
		rego.Module("policy.rego", module),
	).PrepareForEval(ctx)
	if err != nil {
		return false, false
	}
	rs, err := q.Eval(ctx, rego.EvalInput(inputMap(in)))
	if err != nil || len(rs) == 0 {
		return false, false
	}
	visible, ok := rs[0].Expressions[0].Value.(bool)
	return visible, ok
}

func inputMap(in Input) map[string]any {
	return map[string]any{
		"agent_id":  in.AgentID,
		"tool_id":   in.ToolID,
		"name":      in.Name,
		"sql_class": in.SQLClass,
	}
}

// evaluateRego runs the compiled module with the call input.
//
//	{"effect": "allow"|"deny"|"require_approval", "rule_id": "...", "timeout_seconds": n}
//
// An undefined or malformed decision denies the call.
func evaluateRego(module string, in Input) Decision {
	ctx, cancel := context.WithTimeout(context.Background(), regoTimeout)
	defer cancel()
	q, err := rego.New(
		rego.Query("data.keywarden.decision"),
		rego.Module("policy.rego", module),
	).PrepareForEval(ctx)
	if err != nil {
		return Decision{Effect: Deny, Reason: "policy module failed to compile: " + err.Error()}
	}
	input := inputMap(in)
	rs, err := q.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return Decision{Effect: Deny, Reason: "policy evaluation failed: " + err.Error()}
	}
	if len(rs) == 0 {
		return Decision{Effect: Deny, Reason: "no rego decision"}
	}
	raw, ok := rs[0].Expressions[0].Value.(map[string]any)
	if !ok {
		return Decision{Effect: Deny, Reason: "rego decision must be an object"}
	}
	effect, _ := raw["effect"].(string)
	ruleID, _ := raw["rule_id"].(string)
	switch effect {
	case Allow, RequireApproval, Deny:
	default:
		return Decision{Effect: Deny, RuleID: ruleID, Reason: denyReason(effect, ruleID)}
	}
	if effect == Deny {
		reason := "rego decision"
		if ruleID != "" {
			reason = "matched rego rule " + ruleID
		}
		return Decision{Effect: Deny, RuleID: ruleID, Reason: reason}
	}
	decision := Decision{Effect: effect, RuleID: ruleID, Reason: "rego decision"}
	if effect == RequireApproval {
		decision.Timeout = 120 * time.Second
		if seconds, ok := raw["timeout_seconds"].(json.Number); ok {
			if n, err := seconds.Int64(); err == nil && n >= 1 && n <= 600 {
				decision.Timeout = time.Duration(n) * time.Second
			}
		} else if f, ok := raw["timeout_seconds"].(float64); ok && f >= 1 && f <= 600 {
			decision.Timeout = time.Duration(f) * time.Second
		}
	}
	return decision
}

func denyReason(effect, ruleID string) string {
	if effect == "" {
		if ruleID != "" {
			return fmt.Sprintf("rego rule %s returned no effect", ruleID)
		}
		return "rego decision has no effect"
	}
	return "rego returned unknown effect " + effect
}
