package limit

import "testing"

func TestBurstThenDeny(t *testing.T) {
	l := New(0.01, 2)
	if !l.Allow("a") {
		t.Fatal("first call should be allowed")
	}
	if !l.Allow("a") {
		t.Fatal("second call should be allowed")
	}
	if l.Allow("a") {
		t.Fatal("third call should be limited")
	}
	if !l.Allow("b") {
		t.Fatal("keys are independent")
	}
}
