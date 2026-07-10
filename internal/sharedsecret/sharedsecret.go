// Package sharedsecret implements HMAC-SHA256 signing + verification with a
// dual-secret accept window. Mirror of the brightcrm-side package: same wire
// format, same dual-secret semantics, so Dockyard's rotation engine can roll
// the BrightCRM <-> slab shared secret without taking the loop down.
//
// During a rotation grace window, primary holds the new value and secondary
// holds the previous one. Outbound signing always uses primary. Inbound
// verification accepts either. Once the grace window closes, secondary is
// cleared.
//
// Update() and Signer.Update() let the /admin/reload-secrets handler hot-swap
// the values, RWMutex-guarded so concurrent verify/sign requests stay safe.
package sharedsecret

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// Verifier accepts HMAC-SHA256 signatures keyed by either the primary or
// the secondary secret. Constant-time compare against both.
type Verifier struct {
	mu        sync.RWMutex
	primary   []byte
	secondary []byte
}

func NewVerifier(primary, secondary string) *Verifier {
	v := &Verifier{}
	if primary != "" {
		v.primary = []byte(primary)
	}
	if secondary != "" {
		v.secondary = []byte(secondary)
	}
	return v
}

func (v *Verifier) Configured() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.primary) > 0
}

func (v *Verifier) HasSecondary() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.secondary) > 0
}

// Update hot-swaps the primary and secondary values.
func (v *Verifier) Update(primary, secondary string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if primary == "" {
		v.primary = nil
	} else {
		v.primary = []byte(primary)
	}
	if secondary == "" {
		v.secondary = nil
	} else {
		v.secondary = []byte(secondary)
	}
}

func (v *Verifier) Verify(body []byte, hexSig string) bool {
	v.mu.RLock()
	primary := v.primary
	secondary := v.secondary
	v.mu.RUnlock()
	if len(primary) == 0 {
		return false
	}
	if signedEqual(primary, body, hexSig) {
		return true
	}
	if len(secondary) > 0 && signedEqual(secondary, body, hexSig) {
		return true
	}
	return false
}

// Signer always uses the current (primary) secret. Never sign with previous.
type Signer struct {
	mu      sync.RWMutex
	current []byte
}

func NewSigner(current string) *Signer {
	s := &Signer{}
	if current != "" {
		s.current = []byte(current)
	}
	return s
}

func (s *Signer) Configured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.current) > 0
}

// Update hot-swaps the current secret.
func (s *Signer) Update(current string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current == "" {
		s.current = nil
	} else {
		s.current = []byte(current)
	}
}

func (s *Signer) Sign(body []byte) string {
	s.mu.RLock()
	current := s.current
	s.mu.RUnlock()
	if len(current) == 0 {
		return ""
	}
	mac := hmac.New(sha256.New, current)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func signedEqual(key, body []byte, hexSig string) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(hexSig), []byte(expected))
}
