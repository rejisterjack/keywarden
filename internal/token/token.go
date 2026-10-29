package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

// ErrUnauthorized is returned when a task token is missing or malformed.
var ErrUnauthorized = errors.New("unauthorized")

// NewToken returns a task token and the SHA-256 stored at rest.
func NewToken() (string, []byte, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}
	raw := "kwt_" + base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

// Hash returns the stored hash of a task token.
func Hash(raw string) ([]byte, error) {
	if len(raw) != len("kwt_")+43 || !strings.HasPrefix(raw, "kwt_") {
		return nil, ErrUnauthorized
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}
