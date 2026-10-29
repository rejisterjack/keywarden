package policy

import (
	"strings"
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		sql  string
		want string
	}{
		{"SELECT id FROM demo.orders", "read"},
		{"select 1", "read"},
		{"WITH a AS (SELECT 1) SELECT * FROM a", "read"},
		{"EXPLAIN SELECT 1", "read"},
		{"UPDATE demo.orders SET status = 'x' WHERE id = 1", "mutating"},
		{"DELETE FROM demo.orders WHERE id = 1", "mutating"},
		{"INSERT INTO demo.orders (id, status, customer) VALUES (1, 'a', 'b')", "mutating"},
		{"WITH a AS (DELETE FROM demo.orders RETURNING *) SELECT * FROM a", "mutating"},
		{"DROP TABLE demo.orders", "destructive"},
		{"TRUNCATE demo.orders", "destructive"},
		{"ALTER TABLE demo.orders ADD COLUMN x int", "destructive"},
		{"GRANT SELECT ON demo.orders TO PUBLIC", "destructive"},
		{"SELECT 1; DROP TABLE demo.orders", "destructive"},
		{"SELECT 'DROP TABLE demo.orders'", "read"},
		{"SELECT drop_column FROM demo.orders", "read"},
		{"SELECT * INTO demo.copy FROM demo.orders", "destructive"},
		{"", "destructive"},
		{"SELECT pg_read_file('/etc/passwd')", "read"},
	}
	for _, tc := range cases {
		if got := Classify(tc.sql); got != tc.want {
			t.Errorf("Classify(%q) = %s, want %s", tc.sql, got, tc.want)
		}
	}
}

func sample() Document {
	raw := []byte(`{
		"version": 1,
		"rules": [
			{"id": "orders-read", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "read"}, "effect": "allow"},
			{"id": "orders-mutate", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "mutating"}, "effect": "require_approval", "timeout_seconds": 120},
			{"id": "orders-destroy", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "destructive"}, "effect": "deny"},
			{"id": "catalog-lookup", "match": {"agent": "support", "tool": "catalog", "name": "lookup_order"}, "effect": "allow"},
			{"id": "catalog-refund", "match": {"agent": "support", "tool": "catalog", "name": "refund_order"}, "effect": "require_approval"},
			{"id": "catalog-delete", "match": {"agent": "support", "tool": "catalog", "name": "delete_catalog"}, "effect": "deny"}
		]
	}`)
	doc, _, err := Parse(raw)
	if err != nil {
		panic(err)
	}
	return doc
}

func TestEvaluateMatrix(t *testing.T) {
	doc := sample()
	cases := []struct {
		in     Input
		effect string
		rule   string
	}{
		{Input{"support", "orders", "query", "read"}, Allow, "orders-read"},
		{Input{"support", "orders", "query", "mutating"}, RequireApproval, "orders-mutate"},
		{Input{"support", "orders", "query", "destructive"}, Deny, "orders-destroy"},
		{Input{"other", "orders", "query", "read"}, Deny, ""},
		{Input{"support", "catalog", "lookup_order", ""}, Allow, "catalog-lookup"},
		{Input{"support", "catalog", "refund_order", ""}, RequireApproval, "catalog-refund"},
		{Input{"support", "catalog", "delete_catalog", ""}, Deny, "catalog-delete"},
		{Input{"support", "catalog", "unknown", ""}, Deny, ""},
	}
	for _, tc := range cases {
		got := doc.Evaluate(tc.in)
		if got.Effect != tc.effect || got.RuleID != tc.rule {
			t.Errorf("%+v -> %+v", tc.in, got)
		}
	}
	if d := doc.Evaluate(Input{"support", "orders", "query", "mutating"}); d.Timeout != 120*time.Second {
		t.Fatal(d.Timeout)
	}
	if d := doc.Evaluate(Input{"support", "catalog", "refund_order", ""}); d.Timeout != 120*time.Second {
		t.Fatal(d.Timeout)
	}
}

func TestVisibleHidesDeniedOnlyTools(t *testing.T) {
	doc := sample()
	if !doc.Visible("support", "orders", "query") {
		t.Fatal("query should be visible")
	}
	if !doc.Visible("support", "catalog", "lookup_order") || !doc.Visible("support", "catalog", "refund_order") {
		t.Fatal("catalog tools should be visible")
	}
	if doc.Visible("support", "catalog", "delete_catalog") {
		t.Fatal("delete_catalog is deny-only")
	}
	if doc.Visible("other", "orders", "query") {
		t.Fatal("other agent")
	}
}

func TestParseRejectsAllowDefault(t *testing.T) {
	_, _, err := Parse([]byte(`{"version":1,"default":"allow","rules":[]}`))
	if err == nil || !strings.Contains(err.Error(), "deny") {
		t.Fatal(err)
	}
}
