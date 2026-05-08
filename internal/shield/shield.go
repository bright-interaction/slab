// Package shield is the LLM-boundary tokenizer for atomicsite's MCP server.
//
// When the MCP server returns data to an external agent (Claude on Anthropic,
// Groq, etc.), every PII field gets replaced with a marker of the form
//
//	[shield:<kind>:tok_<hex>:<hint>]
//
// The agent reasons over the marker plus its hint metadata (e.g. email
// domain, name initials, value length) without ever seeing the raw value.
// When the agent calls a write tool with marker-bearing arguments, the
// server resolves markers back to plaintext locally before executing the
// underlying logic. The agent + the upstream LLM provider never see the
// real value.
//
// Token format: "[shield:<kind>:tok_<hex>:<hint>]". <kind> is one of the
// constants in this file. tok_ + 8 hex chars is the lookup key. <hint>
// is a comma-separated key=value list whose keys must be whitelisted per
// kind (HintRules in this file). Hints are domain-truthful but never
// leak the raw value.
//
// Storage: per-MCP-connection rows in shield_sessions + shield_tokens.
// Ciphertext = base64(AES-256-GCM(value, ATOMICSITE_SHIELD_KEY)).
//
// Privacy invariant: when shield_enabled is on, no MCP tool response
// leaves this server with a tagged PII field in plaintext, and no MCP
// tool argument runs against the store with an unresolved marker.
package shield

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Kind is a PII category. Add new kinds by extending this list and
// updating HintRules + redactPatterns where applicable.
type Kind string

const (
	KindEmail        Kind = "email"
	KindName         Kind = "name"
	KindPhone        Kind = "phone"
	KindCompany      Kind = "company"
	KindPersonnummer Kind = "personnummer"
	KindIBAN         Kind = "iban"
	KindOrgNumber    Kind = "orgnr"
	KindFreeform     Kind = "freeform"
)

// HintRules is the whitelist of metadata keys allowed per kind. Any key
// not present in this map is rejected at Shield time so mistakes like
// hint="value=anna@example.com" cannot ship.
var HintRules = map[Kind]map[string]bool{
	KindEmail:        {"domain": true, "len": true},
	KindName:         {"initials": true, "len": true},
	KindPhone:        {"country": true, "len": true},
	KindCompany:      {"industry": true, "country": true, "len": true},
	KindPersonnummer: {"century": true},
	KindIBAN:         {"country": true},
	KindOrgNumber:    {"country": true},
	KindFreeform:     {"len": true},
}

// MarkerPattern matches a [shield:...] marker for unshield + redact
// passes. Groups: 1 = kind, 2 = token id (tok_<hex>), 3 = hint (or empty).
var MarkerPattern = regexp.MustCompile(`\[shield:([a-z]+):(tok_[0-9a-f]+)(?::([^\]]*))?\]`)

// FormatMarker produces "[shield:<kind>:<tokenID>:<hint>]" with hint
// omitted entirely when empty. tokenID must include the "tok_" prefix.
func FormatMarker(kind Kind, tokenID, hint string) string {
	if hint == "" {
		return fmt.Sprintf("[shield:%s:%s]", kind, tokenID)
	}
	return fmt.Sprintf("[shield:%s:%s:%s]", kind, tokenID, hint)
}

// validateHint enforces HintRules for the given kind. Returns nil when
// hint is empty or all key=value pairs fall in the whitelist.
func validateHint(kind Kind, hint string) error {
	if hint == "" {
		return nil
	}
	rules, ok := HintRules[kind]
	if !ok {
		return fmt.Errorf("shield: unknown kind %q", kind)
	}
	for _, kv := range strings.Split(hint, ",") {
		k, _, found := strings.Cut(strings.TrimSpace(kv), "=")
		if !found || k == "" {
			return fmt.Errorf("shield: malformed hint pair %q", kv)
		}
		if !rules[k] {
			return fmt.Errorf("shield: hint key %q not allowed for kind %q", k, kind)
		}
	}
	return nil
}

// Common errors returned by the Session API.
var (
	ErrSessionExpired = errors.New("shield: session expired")
	ErrTokenNotFound  = errors.New("shield: token not found in session")
	ErrUnknownKind    = errors.New("shield: unknown kind")
)
