// Package atrest encrypts tenant secrets at rest (app-install
// credentials, TOTP seeds) with AES-256-GCM under the same key as the
// shield tokenizer (ATOMICSITE_SHIELD_KEY).
//
// Stored values carry an "enc:v1:" prefix so ciphertext and legacy
// plaintext coexist during rollout:
//
//   - Encrypt prepends the prefix; empty input passes through unchanged.
//   - Decrypt returns prefixed values decrypted and everything else
//     (legacy plaintext, empty) verbatim.
//
// A deploy without a 32-byte key gets a passthrough Cipher (Encrypt and
// Decrypt are the identity), so single-tenant self-hosters that never
// set ATOMICSITE_SHIELD_KEY keep working and are never locked out. When
// the key is set, new writes are encrypted and existing plaintext rows
// upgrade lazily on their next save.
package atrest

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const prefix = "enc:v1:"

// Cipher encrypts/decrypts at-rest secret values. The zero value is a
// valid passthrough cipher; use New to build one from a key.
type Cipher struct{ key []byte }

// New returns a Cipher. A key that is not exactly 32 bytes yields a
// passthrough Cipher rather than an error, so a missing/empty
// ATOMICSITE_SHIELD_KEY degrades to storing plaintext (the prior
// behavior) instead of refusing to operate.
func New(key []byte) *Cipher {
	if len(key) != 32 {
		return &Cipher{}
	}
	return &Cipher{key: key}
}

// Enabled reports whether a usable 32-byte key is configured.
func (c *Cipher) Enabled() bool { return c != nil && len(c.key) == 32 }

// IsEncrypted reports whether a stored value is atrest ciphertext.
func IsEncrypted(stored string) bool { return strings.HasPrefix(stored, prefix) }

// Encrypt returns the prefixed ciphertext for plaintext. Empty input and
// the passthrough (no key) cipher return the input unchanged.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if !c.Enabled() || plaintext == "" {
		return plaintext, nil
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("atrest: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("atrest: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("atrest: nonce: %w", err)
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt reverses Encrypt. Values without the prefix (legacy plaintext,
// empty) are returned verbatim so old rows keep working. A prefixed
// value with no key configured is an error (the operator removed the key
// after encrypting), surfaced rather than silently returning ciphertext.
func (c *Cipher) Decrypt(stored string) (string, error) {
	if !strings.HasPrefix(stored, prefix) {
		return stored, nil
	}
	if !c.Enabled() {
		return "", errors.New("atrest: encrypted value present but no key configured")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, prefix))
	if err != nil {
		return "", fmt.Errorf("atrest: decode: %w", err)
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("atrest: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("atrest: gcm: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("atrest: ciphertext too short")
	}
	nonce, body := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("atrest: decrypt: %w", err)
	}
	return string(pt), nil
}
