package policy

import "testing"

func BenchmarkEvaluateRules(b *testing.B) {
	doc := sample()
	in := Input{AgentID: "support", ToolID: "orders", Name: "query", SQLClass: "read"}
	b.ReportAllocs()
	for b.Loop() {
		if doc.Evaluate(in).Effect != Allow {
			b.Fatal("expected allow")
		}
	}
}

func BenchmarkEvaluateRego(b *testing.B) {
	doc := Document{Version: 1, Engine: "rego", Rego: demoRego}
	in := Input{AgentID: "support", ToolID: "orders", Name: "query", SQLClass: "read"}
	if err := compileRego(doc.Rego); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if doc.Evaluate(in).Effect != Allow {
			b.Fatal("expected allow")
		}
	}
}
