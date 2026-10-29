package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogRedactsSecrets(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)
	log.Info("connect "+`postgres://keywarden_demo:demo-not-a-secret@localhost/db`,
		"dsn", "postgres://keywarden_demo:demo-not-a-secret@localhost/db",
		"agent", "support",
	)
	got := buf.String()
	if strings.Contains(got, "demo-not-a-secret") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "support") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatal(got)
	}
}
