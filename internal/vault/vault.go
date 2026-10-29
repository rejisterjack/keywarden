package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// Vault seals tool secrets with AES-256-GCM. The tool id is additional
// authenticated data, so a ciphertext cannot be moved onto another tool.
type Vault struct {
	key []byte
}

func New(key []byte) (*Vault, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("vault key must be 32 bytes")
	}
	cp := make([]byte, 32)
	copy(cp, key)
	return &Vault{key: cp}, nil
}

// Seal encrypts plaintext. The nonce is stored separately from the ciphertext.
func (v *Vault) Seal(toolID string, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, []byte(toolID))
	return ciphertext, nonce, nil
}

// Open decrypts a secret for the tool id it was sealed with.
func (v *Vault) Open(toolID string, ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid secret nonce")
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, []byte(toolID))
	if err != nil {
		return nil, fmt.Errorf("secret authentication failed")
	}
	return plain, nil
}
