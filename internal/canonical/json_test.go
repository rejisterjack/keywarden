package canonical

import (
	"bytes"
	"testing"
)

func TestObjectKeyOrder(t *testing.T) {
	a, err := FromJSON([]byte(`{"b":1,"a":{"d":true,"c":"x"}}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := FromJSON([]byte(`{"a":{"c":"x","d":true},"b":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("%s vs %s", a, b)
	}
	if string(a) != `{"a":{"c":"x","d":true},"b":1}` {
		t.Fatal(string(a))
	}
}

func TestHashStable(t *testing.T) {
	h1, err := Hash(map[string]any{"sql": "select 1", "n": 2})
	if err != nil {
		t.Fatal(err)
	}
	h2, err := Hash(map[string]any{"n": 2, "sql": "select 1"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(h1, h2) {
		t.Fatal("hash changed with key order")
	}
}
