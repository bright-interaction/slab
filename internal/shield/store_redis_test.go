package shield

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newRedisStoreForTest spins up a fresh miniredis instance and returns a
// RedisStore that talks to it. The miniredis instance is torn down via
// t.Cleanup so each test gets isolated state.
func newRedisStoreForTest(t *testing.T) *RedisStore {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRedisStore(client)
}

func TestRedisStore_InsertAndGetSession(t *testing.T) {
	store := newRedisStoreForTest(t)
	ctx := context.Background()
	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	if err := store.InsertSession(ctx, "sess1", exp); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, ok, err := store.GetSession(ctx, "sess1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("session not found")
	}
	if !got.Equal(exp) {
		t.Errorf("expiry roundtrip: got %v want %v", got, exp)
	}
}

func TestRedisStore_GetMissingSessionReturnsNotFound(t *testing.T) {
	store := newRedisStoreForTest(t)
	_, ok, err := store.GetSession(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ok {
		t.Error("missing session should not be found")
	}
}

func TestRedisStore_UpdateSessionExpiryExtendsTTL(t *testing.T) {
	store := newRedisStoreForTest(t)
	ctx := context.Background()
	exp1 := time.Now().Add(10 * time.Minute).UTC().Truncate(time.Second)
	exp2 := exp1.Add(time.Hour)
	if err := store.InsertSession(ctx, "sess2", exp1); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := store.UpdateSessionExpiry(ctx, "sess2", exp2); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, ok, _ := store.GetSession(ctx, "sess2")
	if !ok || !got.Equal(exp2) {
		t.Errorf("expected refreshed expiry %v, got %v (ok=%v)", exp2, got, ok)
	}
}

func TestRedisStore_UpdateMissingSessionIsNoOp(t *testing.T) {
	store := newRedisStoreForTest(t)
	err := store.UpdateSessionExpiry(context.Background(), "no-such-session", time.Now().Add(time.Hour))
	if err != nil {
		t.Errorf("update on missing session should be silent no-op; got %v", err)
	}
}

func TestRedisStore_TokenRoundTrip(t *testing.T) {
	store := newRedisStoreForTest(t)
	ctx := context.Background()
	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	if err := store.InsertSession(ctx, "sess3", exp); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if err := store.InsertToken(ctx, "sess3", "tok_abc123", "email", "domain=example.com", "ciphertext-blob"); err != nil {
		t.Fatalf("insert token: %v", err)
	}
	ct, ok, err := store.GetTokenCiphertext(ctx, "sess3", "tok_abc123")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("token not found")
	}
	if ct != "ciphertext-blob" {
		t.Errorf("ciphertext: got %q want %q", ct, "ciphertext-blob")
	}
}

func TestRedisStore_TokenScopedToSession(t *testing.T) {
	// Replay protection: a token issued in session A must look "not found"
	// when queried under session B's id. Otherwise an agent could lift
	// tokens across sessions.
	store := newRedisStoreForTest(t)
	ctx := context.Background()
	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	for _, id := range []string{"sessA", "sessB"} {
		if err := store.InsertSession(ctx, id, exp); err != nil {
			t.Fatalf("insert session %s: %v", id, err)
		}
	}
	if err := store.InsertToken(ctx, "sessA", "tok_shared", "email", "", "plain-A"); err != nil {
		t.Fatalf("insert token: %v", err)
	}
	if _, ok, _ := store.GetTokenCiphertext(ctx, "sessB", "tok_shared"); ok {
		t.Error("token must not resolve under a different session id")
	}
	if _, ok, _ := store.GetTokenCiphertext(ctx, "sessA", "tok_shared"); !ok {
		t.Error("token must resolve under its own session id")
	}
}

func TestRedisStore_GetMissingTokenReturnsNotFound(t *testing.T) {
	store := newRedisStoreForTest(t)
	_, ok, err := store.GetTokenCiphertext(context.Background(), "sess", "tok_nope")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ok {
		t.Error("missing token should not be found")
	}
}

func TestRedisStore_SweepIsNoOp(t *testing.T) {
	// Redis garbage-collects via TTL; SweepExpiredSessions should be a
	// silent no-op so the shared sweep job remains compatible.
	store := newRedisStoreForTest(t)
	if err := store.SweepExpiredSessions(context.Background(), time.Now()); err != nil {
		t.Errorf("sweep should be a silent no-op; got %v", err)
	}
}

func TestRedisStore_PrefixIsolatesTenants(t *testing.T) {
	// Two tenants sharing one Redis with different prefixes must not
	// see each other's sessions.
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	tenantA := NewRedisStore(client)
	tenantA.Prefix = "tenA:shield:"
	tenantB := NewRedisStore(client)
	tenantB.Prefix = "tenB:shield:"
	ctx := context.Background()
	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	if err := tenantA.InsertSession(ctx, "shared-id", exp); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := tenantB.GetSession(ctx, "shared-id"); ok {
		t.Error("tenant B must not see tenant A's session")
	}
}

func TestNewRedisStoreFromURL_RejectsGarbage(t *testing.T) {
	_, err := NewRedisStoreFromURL("not-a-real-url")
	if err == nil {
		t.Error("NewRedisStoreFromURL should reject garbage URLs")
	}
}

func TestParseRedisTokenJSON(t *testing.T) {
	encoded := redisTokenJSON("sess", "email", "domain=x.com", "ct-blob")
	sessID, kind, hint, ct, ok := parseRedisTokenJSON(encoded)
	if !ok {
		t.Fatal("round-trip parse failed")
	}
	if sessID != "sess" || kind != "email" || hint != "domain=x.com" || ct != "ct-blob" {
		t.Errorf("round-trip mismatch: %s | %s | %s | %s", sessID, kind, hint, ct)
	}
	if _, _, _, _, ok := parseRedisTokenJSON("only-one-part"); ok {
		t.Error("malformed payload should not parse ok")
	}
}
