package policy

import (
	"encoding/json"
	"testing"
	"time"
)

const demoRego = `package keywarden

read := {"effect": "allow", "rule_id": "orders-read"}

mutate := {"effect": "require_approval", "rule_id": "orders-mutate", "timeout_seconds": 60}

destroy := {"effect": "deny", "rule_id": "orders-destroy"}

decision := read {
	input.tool_id == "orders"
	input.name == "query"
	input.sql_class == "read"
}

decision := mutate {
	input.tool_id == "orders"
	input.name == "query"
	input.sql_class == "mutating"
}

decision := destroy {
	input.tool_id == "orders"
	input.name == "query"
	input.sql_class == "destructive"
}

visible {
	input.tool_id == "orders"
	input.name == "query"
}
`

func TestParseRegoDocument(t *testing.T) {
	raw := []byte(`{"version":1,"engine":"rego","rego":` + jsonString(t, demoRego) + `}`)
	doc, _, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse rego policy: %v", err)
	}
	if doc.Engine != "rego" {
		t.Fatalf("engine = %q", doc.Engine)
	}
}

func TestParseRegoRejectsInvalidModule(t *testing.T) {
	raw := []byte(`{"version":1,"engine":"rego","rego":"this is not rego"}`)
	if _, _, err := Parse(raw); err == nil {
		t.Fatal("expected invalid rego to fail parse")
	}
}

func TestParseRegoRejectsRules(t *testing.T) {
	raw := []byte(`{"version":1,"engine":"rego","rego":"package keywarden\n\nimport rego.v1\n\ndecision := {\"effect\": \"deny\"}","rules":[{"id":"x","match":{},"effect":"allow"}]}`)
	if _, _, err := Parse(raw); err == nil {
		t.Fatal("expected rules with rego engine to fail parse")
	}
}

func TestParseRejectsRegoWithoutEngine(t *testing.T) {
	raw := []byte(`{"version":1,"rego":"package keywarden"}`)
	if _, _, err := Parse(raw); err == nil {
		t.Fatal("expected rego source without engine to fail parse")
	}
}

func TestEvaluateRegoAllowDenyApproval(t *testing.T) {
	doc := Document{Version: 1, Engine: "rego", Rego: demoRego}

	allow := doc.Evaluate(Input{AgentID: "support", ToolID: "orders", Name: "query", SQLClass: "read"})
	if allow.Effect != Allow || allow.RuleID != "orders-read" {
		t.Fatalf("read = %+v", allow)
	}

	mutate := doc.Evaluate(Input{AgentID: "support", ToolID: "orders", Name: "query", SQLClass: "mutating"})
	if mutate.Effect != RequireApproval || mutate.Timeout != 60*time.Second {
		t.Fatalf("mutating = %+v", mutate)
	}

	destroy := doc.Evaluate(Input{AgentID: "support", ToolID: "orders", Name: "query", SQLClass: "destructive"})
	if destroy.Effect != Deny || destroy.RuleID != "orders-destroy" {
		t.Fatalf("destructive = %+v", destroy)
	}

	other := doc.Evaluate(Input{AgentID: "support", ToolID: "catalog", Name: "lookup"})
	if other.Effect != Deny {
		t.Fatalf("unmatched = %+v", other)
	}
}

func TestEvaluateRegoMalformedDecisionDenies(t *testing.T) {
	doc := Document{Version: 1, Engine: "rego", Rego: "package keywarden\n\nimport rego.v1\n\ndecision := {\"effect\": \"maybe\"}\n"}
	got := doc.Evaluate(Input{AgentID: "a", ToolID: "b", Name: "c"})
	if got.Effect != Deny {
		t.Fatalf("unknown effect = %+v", got)
	}
}

func TestVisibleRego(t *testing.T) {
	doc := Document{Version: 1, Engine: "rego", Rego: demoRego}
	if !doc.Visible("support", "orders", "query") {
		t.Fatal("orders query should be visible")
	}
	if doc.Visible("support", "catalog", "delete_catalog") {
		t.Fatal("catalog delete should not be visible")
	}
}

func jsonString(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}
