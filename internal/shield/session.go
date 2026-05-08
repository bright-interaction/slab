package shield

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Store is the minimal logical surface Session needs. Each calling
// service supplies an implementation: shield.SQLStore wraps database/sql
// (SQLite) for atomicsite + dockyard; brightcrm's mcp package provides a
// pgx-backed implementation. The interface keeps the shield package
// pure logic + crypto so the same code ships unchanged across services.
type Store interface {
	// InsertSession creates one row in shield_sessions. id is the MCP
	// connection identifier; expiresAt is absolute UTC.
	InsertSession(ctx context.Context, id string, expiresAt time.Time) error

	// GetSession returns the stored expiry. found=false when the id is
	// unknown. err is non-nil only on driver-level failure.
	GetSession(ctx context.Context, id string) (expiresAt time.Time, found bool, err error)

	// UpdateSessionExpiry refreshes last_seen + expires_at on an
	// existing session. No-op when id is unknown (caller treats that
	// case as create).
	UpdateSessionExpiry(ctx context.Context, id string, expiresAt time.Time) error

	// SweepExpiredSessions deletes every session whose expires_at is
	// strictly before the given cutoff. ON DELETE CASCADE removes the
	// associated tokens.
	SweepExpiredSessions(ctx context.Context, before time.Time) error

	// InsertToken records one new token. token is the "tok_<hex>" id
	// (no marker brackets); kind/hint are the metadata; ciphertext is
	// base64 AES-256-GCM(value, key).
	InsertToken(ctx context.Context, sessionID, token, kind, hint, ciphertext string) error

	// GetTokenCiphertext returns the encrypted blob for the token if
	// it is bound to sessionID. found=false when the token is unknown
	// or belongs to a different session (the caller treats both as
	// ErrTokenNotFound for replay protection).
	GetTokenCiphertext(ctx context.Context, sessionID, token string) (ciphertext string, found bool, err error)
}

// Session is one MCP connection's token vault. Safe for concurrent
// use; the Store implementation is responsible for its own locking.
type Session struct {
	ID    string
	store Store
	key   []byte
	ttl   time.Duration
}

// NewSession looks up an existing session and touches expires_at, or
// creates a fresh row when the id is unknown. Returns an active
// Session bound to the given store + key. Caller is expected to have
// authenticated the MCP request (X-Agent-Key, etc.) before calling.
func NewSession(ctx context.Context, store Store, sessionID string, key []byte, ttl time.Duration) (*Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("shield: empty session id")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("shield: key must be 32 bytes, got %d", len(key))
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	now := time.Now().UTC()
	newExpiry := now.Add(ttl)

	expiresAt, found, err := store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("shield: lookup session: %w", err)
	}
	if found {
		if expiresAt.Before(now) {
			return nil, ErrSessionExpired
		}
		if err := store.UpdateSessionExpiry(ctx, sessionID, newExpiry); err != nil {
			return nil, fmt.Errorf("shield: touch session: %w", err)
		}
	} else {
		if err := store.InsertSession(ctx, sessionID, newExpiry); err != nil {
			return nil, fmt.Errorf("shield: create session: %w", err)
		}
	}
	return &Session{ID: sessionID, store: store, key: key, ttl: ttl}, nil
}

// SweepExpired deletes session rows past their expires_at. Cascading
// FK takes the tokens with them. Safe to call from a janitor goroutine.
func SweepExpired(ctx context.Context, store Store) error {
	return store.SweepExpiredSessions(ctx, time.Now().UTC())
}

// Tokenize records one new token and returns its full marker string.
// hint may be empty. Returns ErrUnknownKind for unsupported kinds and a
// wrapped error for non-whitelisted hint keys. The plaintext is
// AES-256-GCM-encrypted before hitting the row.
func (s *Session) Tokenize(ctx context.Context, kind Kind, value, hint string) (string, error) {
	if _, ok := HintRules[kind]; !ok {
		return "", ErrUnknownKind
	}
	if err := validateHint(kind, hint); err != nil {
		return "", err
	}
	tokenID, err := newTokenID()
	if err != nil {
		return "", fmt.Errorf("shield: generate token id: %w", err)
	}
	ct, err := encrypt(value, s.key)
	if err != nil {
		return "", err
	}
	if err := s.store.InsertToken(ctx, s.ID, tokenID, string(kind), hint, ct); err != nil {
		return "", fmt.Errorf("shield: insert token: %w", err)
	}
	return FormatMarker(kind, tokenID, hint), nil
}

// Resolve maps a "tok_<hex>" id back to its plaintext value. Returns
// ErrTokenNotFound when the id is unknown or scoped to another session
// (replay protection: tokens issued for session A cannot be redeemed
// in session B).
func (s *Session) Resolve(ctx context.Context, tokenID string) (string, error) {
	if !strings.HasPrefix(tokenID, "tok_") {
		return "", ErrTokenNotFound
	}
	ct, found, err := s.store.GetTokenCiphertext(ctx, s.ID, tokenID)
	if err != nil {
		return "", fmt.Errorf("shield: lookup token: %w", err)
	}
	if !found {
		return "", ErrTokenNotFound
	}
	return decrypt(ct, s.key)
}

// Shield walks v recursively and replaces every shield-tagged string
// field with a marker. v must be a pointer to a struct. Untagged
// fields, non-string types, and unexported fields pass through. Slices,
// maps, pointers, and interfaces are descended.
//
// Tag format: `shield:"tokenize,kind=email,hint=domain len"`. The kind=
// part is required; hint= is optional and lists the metadata keys to
// auto-extract from the value.
//
// `shield:"id"` and `shield:"-"` are explicit "leave alone" markers
// (the field is not tokenized but is still descended for nested types).
func (s *Session) Shield(ctx context.Context, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("shield: Shield requires a pointer, got %T", v)
	}
	return s.shieldValue(ctx, rv.Elem())
}

func (s *Session) shieldValue(ctx context.Context, rv reflect.Value) error {
	if !rv.IsValid() {
		return nil
	}
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return s.shieldValue(ctx, rv.Elem())
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if err := s.shieldValue(ctx, rv.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := rv.MapRange()
		for iter.Next() {
			elem := iter.Value()
			tmp := reflect.New(elem.Type()).Elem()
			tmp.Set(elem)
			if err := s.shieldValue(ctx, tmp); err != nil {
				return err
			}
			rv.SetMapIndex(iter.Key(), tmp)
		}
	case reflect.Struct:
		t := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := f.Tag.Get("shield")
			fv := rv.Field(i)
			if tag == "" || tag == "-" || tag == "id" {
				if err := s.shieldValue(ctx, fv); err != nil {
					return err
				}
				continue
			}
			kind, hintKeys, err := parseShieldTag(tag)
			if err != nil {
				return err
			}
			if err := s.shieldField(ctx, fv, kind, hintKeys); err != nil {
				return err
			}
		}
	}
	return nil
}

// shieldField replaces a single tagged field's value with its marker.
// String fields are tokenized; pointer-to-string is dereferenced and
// tokenized in place. Other types pass through (logged via the tag
// validator: a tag on a non-string field is a programmer error caught
// in the unit tests).
func (s *Session) shieldField(ctx context.Context, fv reflect.Value, kind Kind, hintKeys []string) error {
	if !fv.CanSet() {
		return fmt.Errorf("shield: tagged field is not settable (likely unexported)")
	}
	switch fv.Kind() {
	case reflect.String:
		raw := fv.String()
		if raw == "" {
			return nil
		}
		hint := buildHint(kind, raw, hintKeys)
		marker, err := s.Tokenize(ctx, kind, raw, hint)
		if err != nil {
			return err
		}
		fv.SetString(marker)
	case reflect.Ptr:
		if !fv.IsNil() {
			return s.shieldField(ctx, fv.Elem(), kind, hintKeys)
		}
	}
	return nil
}

// Unshield walks v recursively and replaces every shield marker back
// to its plaintext. Used by write-tool argument handlers before
// executing the underlying logic. Markers belonging to other sessions
// or expired tokens cause the call to fail with ErrTokenNotFound,
// which write handlers should surface as an MCP error to the agent.
func (s *Session) Unshield(ctx context.Context, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("shield: Unshield requires a pointer, got %T", v)
	}
	return s.unshieldValue(ctx, rv.Elem())
}

func (s *Session) unshieldValue(ctx context.Context, rv reflect.Value) error {
	if !rv.IsValid() {
		return nil
	}
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return s.unshieldValue(ctx, rv.Elem())
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if err := s.unshieldValue(ctx, rv.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := rv.MapRange()
		for iter.Next() {
			elem := iter.Value()
			tmp := reflect.New(elem.Type()).Elem()
			tmp.Set(elem)
			if err := s.unshieldValue(ctx, tmp); err != nil {
				return err
			}
			rv.SetMapIndex(iter.Key(), tmp)
		}
	case reflect.Struct:
		for i := 0; i < rv.NumField(); i++ {
			if err := s.unshieldValue(ctx, rv.Field(i)); err != nil {
				return err
			}
		}
	case reflect.String:
		if rv.CanSet() {
			out, err := s.UnshieldString(ctx, rv.String())
			if err != nil {
				return err
			}
			rv.SetString(out)
		}
	}
	return nil
}

// UnshieldString resolves every [shield:...] marker in the input back
// to plaintext. Strings without markers are returned unchanged.
func (s *Session) UnshieldString(ctx context.Context, in string) (string, error) {
	if !strings.Contains(in, "[shield:") {
		return in, nil
	}
	var firstErr error
	out := MarkerPattern.ReplaceAllStringFunc(in, func(match string) string {
		groups := MarkerPattern.FindStringSubmatch(match)
		if len(groups) < 3 {
			return match
		}
		tokenID := groups[2]
		pt, err := s.Resolve(ctx, tokenID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return match
		}
		return pt
	})
	return out, firstErr
}

// UnshieldJSON parses a JSON document, walks every string node, and
// resolves markers back to plaintext. Returns the re-marshaled JSON.
// Convenient for MCP tool-call argument blobs that arrive as
// json.RawMessage.
func (s *Session) UnshieldJSON(ctx context.Context, raw []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("shield: unmarshal: %w", err)
	}
	resolved, err := s.unshieldAny(ctx, v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(resolved)
}

// ShieldJSON parses a JSON document, walks every string node, runs the
// regex bank over each string value to replace any inline PII with
// markers, and returns the re-marshaled JSON. Used by the MCP
// dispatcher as a safety net so tool handlers that build raw JSON
// without typed structs still get redacted output. Strings that already
// contain markers are not touched (the redact pass is idempotent).
func (s *Session) ShieldJSON(ctx context.Context, raw []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("shield: unmarshal: %w", err)
	}
	redacted, err := s.shieldAny(ctx, v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(redacted)
}

func (s *Session) shieldAny(ctx context.Context, v any) (any, error) {
	switch t := v.(type) {
	case string:
		return s.RedactString(ctx, t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			r, err := s.shieldAny(ctx, item)
			if err != nil {
				return nil, err
			}
			out[i] = r
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			r, err := s.shieldAny(ctx, val)
			if err != nil {
				return nil, err
			}
			out[k] = r
		}
		return out, nil
	default:
		return v, nil
	}
}

func (s *Session) unshieldAny(ctx context.Context, v any) (any, error) {
	switch t := v.(type) {
	case string:
		return s.UnshieldString(ctx, t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			r, err := s.unshieldAny(ctx, item)
			if err != nil {
				return nil, err
			}
			out[i] = r
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			r, err := s.unshieldAny(ctx, val)
			if err != nil {
				return nil, err
			}
			out[k] = r
		}
		return out, nil
	default:
		return v, nil
	}
}

// parseShieldTag splits "tokenize,kind=email,hint=domain len" into
// kind + an ordered list of hint keys. The "tokenize" verb is
// currently the only one; the slot exists so future verbs (drop,
// hash) can plug in without changing the tag format.
func parseShieldTag(tag string) (Kind, []string, error) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) != "tokenize" {
		return "", nil, fmt.Errorf("shield: tag must start with 'tokenize', got %q", tag)
	}
	var kind Kind
	var hintKeys []string
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		k, v, found := strings.Cut(p, "=")
		if !found {
			return "", nil, fmt.Errorf("shield: malformed tag part %q", p)
		}
		switch k {
		case "kind":
			kind = Kind(v)
		case "hint":
			for _, hk := range strings.Split(v, " ") {
				hk = strings.TrimSpace(hk)
				if hk != "" {
					hintKeys = append(hintKeys, hk)
				}
			}
		default:
			return "", nil, fmt.Errorf("shield: unknown tag key %q", k)
		}
	}
	if kind == "" {
		return "", nil, errors.New("shield: tag missing kind")
	}
	if _, ok := HintRules[kind]; !ok {
		return "", nil, fmt.Errorf("shield: tag references unknown kind %q", kind)
	}
	return kind, hintKeys, nil
}

// buildHint extracts the hint keys named in the tag from the raw
// value. Unknown keys (not in HintRules[kind]) are skipped silently
// so a typo in a tag does not block the response. Returns "" when no
// keys produce a value.
func buildHint(kind Kind, raw string, keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	rules := HintRules[kind]
	var pairs []string
	for _, k := range keys {
		if !rules[k] {
			continue
		}
		v := extractHint(kind, raw, k)
		if v != "" {
			pairs = append(pairs, k+"="+v)
		}
	}
	return strings.Join(pairs, ",")
}

// extractHint returns the value of one whitelisted hint key for the
// given kind + raw value. Centralised so HintRules + this function
// are the only places to update when a new hint key is added.
func extractHint(kind Kind, raw, key string) string {
	switch key {
	case "len":
		return fmt.Sprintf("%d", len(raw))
	case "domain":
		if kind == KindEmail {
			if at := strings.LastIndex(raw, "@"); at >= 0 && at < len(raw)-1 {
				return raw[at+1:]
			}
		}
	case "initials":
		if kind == KindName {
			fields := strings.Fields(raw)
			var b strings.Builder
			for _, f := range fields {
				if f != "" {
					b.WriteByte(f[0])
				}
			}
			return strings.ToUpper(b.String())
		}
	case "country":
		switch kind {
		case KindPhone:
			if strings.HasPrefix(raw, "+46") {
				return "SE"
			}
			if strings.HasPrefix(raw, "+1") {
				return "US"
			}
			if strings.HasPrefix(raw, "+44") {
				return "GB"
			}
		case KindIBAN:
			if len(raw) >= 2 {
				return strings.ToUpper(raw[:2])
			}
		}
	case "century":
		if kind == KindPersonnummer && len(raw) >= 2 {
			if raw[0] == '1' || raw[0] == '2' {
				return raw[:2]
			}
		}
	}
	return ""
}
