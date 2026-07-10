package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/shield"
)

// testAgent is a stable server-resolved identity used by the shield
// session tests; beginShieldSession scopes the session to (SiteID, KeyID).
var testAgent = &authmw.AgentIdentity{SiteID: "site-1", KeyID: "key-1"}

// TestShieldHotSwap_ExistingSessionsKeepWorking is the safety invariant:
// rotating SHIELD_KEY must not invalidate an in-flight Session. The
// Session value-copies the key at NewSession time, so a swap on the
// server only affects subsequently-opened sessions. Token resolution
// for the original session must still succeed using the original key.
func TestShieldHotSwap_ExistingSessionsKeepWorking(t *testing.T) {
	store := shield.NewMemoryStore()
	const oldKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	const newKey = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"

	srv := NewServer(nil, nil)
	srv.WithShield(store, []byte(oldKey), 5*time.Minute, shield.HintFull)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Mcp-Session-Id", "sess-1")
	sess1, err := srv.beginShieldSession(r, testAgent)
	if err != nil {
		t.Fatalf("beginShieldSession #1: %v", err)
	}

	tok, err := sess1.Tokenize(context.Background(), shield.KindEmail, "alice@example.com", "domain=example.com")
	if err != nil {
		t.Fatalf("Tokenize before rotation: %v", err)
	}

	if err := srv.UpdateShieldKey([]byte(newKey)); err != nil {
		t.Fatalf("UpdateShieldKey: %v", err)
	}

	// sess1 still uses oldKey (snapshotted at creation). Resolve must
	// succeed because the token was encrypted with oldKey and sess1
	// still holds oldKey.
	plain, err := sess1.Resolve(context.Background(), tokenIDFromMarker(t, tok))
	if err != nil {
		t.Fatalf("Resolve after rotation on existing session: %v (this is the data-corruption scenario the design must prevent)", err)
	}
	if plain != "alice@example.com" {
		t.Fatalf("Resolve returned %q, want plaintext", plain)
	}

	// A NEW session opened post-rotation gets the new key.
	r2 := httptest.NewRequest(http.MethodPost, "/", nil)
	r2.Header.Set("Mcp-Session-Id", "sess-2")
	sess2, err := srv.beginShieldSession(r2, testAgent)
	if err != nil {
		t.Fatalf("beginShieldSession #2: %v", err)
	}
	tok2, err := sess2.Tokenize(context.Background(), shield.KindEmail, "bob@example.com", "domain=example.com")
	if err != nil {
		t.Fatalf("Tokenize on new session: %v", err)
	}
	plain2, err := sess2.Resolve(context.Background(), tokenIDFromMarker(t, tok2))
	if err != nil {
		t.Fatalf("Resolve on new session: %v", err)
	}
	if plain2 != "bob@example.com" {
		t.Fatalf("Resolve sess2 = %q, want plaintext", plain2)
	}
}

// TestShieldSessionIsTenantScoped is the C1 regression guard: two
// different agent identities that send the IDENTICAL Mcp-Session-Id
// header must NOT share a token vault. Before the fix the shield session
// id was taken straight from the client header, so tenant B could
// present tenant A's session id and resolve A's PII tokens. The session
// id is now derived from the server-resolved (SiteID, KeyID), so B's
// resolve of A's token must fail with ErrTokenNotFound.
func TestShieldSessionIsTenantScoped(t *testing.T) {
	store := shield.NewMemoryStore()
	srv := NewServer(nil, nil)
	srv.WithShield(store, []byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"), 5*time.Minute, shield.HintFull)

	tenantA := &authmw.AgentIdentity{SiteID: "site-A", KeyID: "key-A"}
	tenantB := &authmw.AgentIdentity{SiteID: "site-B", KeyID: "key-B"}

	// Both callers send the SAME Mcp-Session-Id header (the spoofing vector).
	rA := httptest.NewRequest(http.MethodPost, "/", nil)
	rA.Header.Set("Mcp-Session-Id", "shared-id")
	sessA, err := srv.beginShieldSession(rA, tenantA)
	if err != nil {
		t.Fatalf("beginShieldSession A: %v", err)
	}
	rB := httptest.NewRequest(http.MethodPost, "/", nil)
	rB.Header.Set("Mcp-Session-Id", "shared-id")
	sessB, err := srv.beginShieldSession(rB, tenantB)
	if err != nil {
		t.Fatalf("beginShieldSession B: %v", err)
	}

	if sessA.ID == sessB.ID {
		t.Fatalf("tenant A and B share session id %q despite different identities (cross-tenant vault)", sessA.ID)
	}

	marker, err := sessA.Tokenize(context.Background(), shield.KindEmail, "secret@tenant-a.com", "")
	if err != nil {
		t.Fatalf("tenant A Tokenize: %v", err)
	}
	tokID := tokenIDFromMarker(t, marker)

	// Tenant B must not be able to resolve tenant A's token.
	if _, err := sessB.Resolve(context.Background(), tokID); err != shield.ErrTokenNotFound {
		t.Fatalf("tenant B resolved tenant A's token (got err=%v); cross-tenant PII de-anonymization is still possible", err)
	}
	// Sanity: tenant A still resolves its own token.
	if got, err := sessA.Resolve(context.Background(), tokID); err != nil || got != "secret@tenant-a.com" {
		t.Fatalf("tenant A self-resolve = %q, err=%v; want own plaintext", got, err)
	}
}

func TestShieldHotSwap_RejectsInvalidLength(t *testing.T) {
	srv := NewServer(nil, nil)
	srv.WithShield(shield.NewMemoryStore(), []byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"), time.Minute, shield.HintFull)

	if err := srv.UpdateShieldKey([]byte("too-short")); err == nil {
		t.Fatal("UpdateShieldKey with 9-byte key returned nil, want length error")
	}
	if err := srv.UpdateShieldKey([]byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")); err != nil {
		t.Fatalf("UpdateShieldKey with key matching current must be a no-op (rotation engine sends the same value twice during dual-phase delivery), got err: %v", err)
	}
}

func TestShieldHotSwap_RejectsWhenDisabled(t *testing.T) {
	srv := NewServer(nil, nil)
	if err := srv.UpdateShieldKey([]byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")); err == nil {
		t.Fatal("UpdateShieldKey on disabled shield returned nil, want error")
	}
}

func TestShieldHotSwap_ConcurrentReadsWhileSwapping(t *testing.T) {
	srv := NewServer(nil, nil)
	srv.WithShield(shield.NewMemoryStore(), []byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"), time.Minute, shield.HintFull)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					r := httptest.NewRequest(http.MethodPost, "/", nil)
					r.Header.Set("Mcp-Session-Id", "concurrent")
					_, _ = srv.beginShieldSession(r, testAgent)
				}
			}
		}()
	}

	// Toggle keys for 50ms. With sync.RWMutex this should never race.
	for i := 0; i < 20; i++ {
		var k [32]byte
		for j := range k {
			k[j] = byte('A' + (i % 16))
		}
		_ = srv.UpdateShieldKey(k[:])
		time.Sleep(2 * time.Millisecond)
	}
	close(stop)
	wg.Wait()
}

// tokenIDFromMarker extracts the tok_<hex> id out of a [shield:...]
// marker so we can pass it to Session.Resolve. Failures here mean the
// marker shape changed and the test needs updating, not that the
// rotation logic is broken.
func tokenIDFromMarker(t *testing.T, marker string) string {
	t.Helper()
	m := shield.MarkerPattern.FindStringSubmatch(marker)
	if len(m) < 3 {
		t.Fatalf("marker %q did not match shield.MarkerPattern", marker)
	}
	return m[2]
}
