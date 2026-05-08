package shield

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// encrypt encrypts plaintext with AES-256-GCM and returns base64
// ciphertext. Nonce is prepended to the ciphertext (standard GCM
// pattern). Empty plaintext is passed through.
func encrypt(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if len(key) != 32 {
		return "", fmt.Errorf("shield: encryption key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("shield: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("shield: create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("shield: nonce: %w", err)
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// decrypt reverses encrypt. Empty input returns empty.
func decrypt(encoded string, key []byte) (string, error) {
	if encoded == "" {
		return "", nil
	}
	if len(key) != 32 {
		return "", fmt.Errorf("shield: decryption key must be 32 bytes, got %d", len(key))
	}
	ct, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("shield: decode base64: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("shield: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("shield: create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ct) < nonceSize {
		return "", fmt.Errorf("shield: ciphertext too short")
	}
	nonce, body := ct[:nonceSize], ct[nonceSize:]
	pt, err := gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("shield: decrypt: %w", err)
	}
	return string(pt), nil
}

// newTokenID returns "tok_<8hex>" using crypto/rand. Collision space is
// 2^32 per session which is fine for the token vault scope (one MCP
// connection, max ~minutes lifetime). Two collisions in the same
// session would surface as a CreateShieldToken primary-key conflict
// the caller can retry.
func newTokenID() (string, error) {
	b := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return "tok_" + hex.EncodeToString(b), nil
}
