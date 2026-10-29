package redact

import "testing"

func TestStringMasksSecrets(t *testing.T) {
	in := "postgres://demo:demo-not-a-secret@localhost/db Authorization: Bearer abc.def token kwt_abcdefghijklmnopqrstuvwxyz0123456789 and sk-live-secretvalue"
	got := String(in)
	for _, leak := range []string{"demo-not-a-secret", "abc.def", "kwt_abcdefghijklmnopqrstuvwxyz0123456789", "sk-live-secretvalue"} {
		if contains(got, leak) {
			t.Fatalf("leaked %s in %s", leak, got)
		}
	}
}

func TestValueRedactsKeys(t *testing.T) {
	got := Value(map[string]any{
		"sql":      "select 1",
		"password": "hunter2",
		"nested":   map[string]any{"api_key": "k"},
	}).(map[string]any)
	if got["password"] != "[redacted]" {
		t.Fatal(got["password"])
	}
	nested := got["nested"].(map[string]any)
	if nested["api_key"] != "[redacted]" {
		t.Fatal(nested)
	}
	if got["sql"] != "select 1" {
		t.Fatal(got["sql"])
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
