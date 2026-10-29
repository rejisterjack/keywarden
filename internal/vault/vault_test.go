package vault

import (
	"bytes"
	"testing"
)

func TestRoundTripAndAAD(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	v, err := vaultNew(key)
	if err != nil {
		t.Fatal(err)
	}
	ct, nonce, err := v.Seal("orders", []byte(`{"dsn":"postgres://example"}`))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := v.Open("orders", ct, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, []byte(`{"dsn":"postgres://example"}`)) {
		t.Fatalf("plain = %s", plain)
	}
	if _, err := v.Open("other", ct, nonce); err == nil {
		t.Fatal("expected AAD mismatch to fail")
	}
	ct[0] ^= 0xff
	if _, err := v.Open("orders", ct, nonce); err == nil {
		t.Fatal("expected tampered ciphertext to fail")
	}
}

func vaultNew(key []byte) (*Vault, error) { return New(key) }

func TestRejectsShortKey(t *testing.T) {
	if _, err := New([]byte("short")); err == nil {
		t.Fatal("expected error")
	}
}
