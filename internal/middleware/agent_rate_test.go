package middleware

import (
	"testing"
	"time"
)

// TestAgentRateLimiter_Burst verifies the bucket starts at capacity
// and exactly capacity calls succeed before throttling.
func TestAgentRateLimiter_Burst(t *testing.T) {
	t.Parallel()
	l := newAgentRateLimiter(5, 0.0001, time.Hour) // very slow refill
	for i := 0; i < 5; i++ {
		ok, _ := l.allow("k1")
		if !ok {
			t.Fatalf("call %d denied; expected first 5 to pass", i)
		}
	}
	ok, retry := l.allow("k1")
	if ok {
		t.Fatal("6th call should be denied")
	}
	if retry < 1 {
		t.Errorf("retryAfter=%d, want >= 1", retry)
	}
}

// TestAgentRateLimiter_Refill confirms tokens refill over time.
func TestAgentRateLimiter_Refill(t *testing.T) {
	t.Parallel()
	l := newAgentRateLimiter(2, 100, time.Hour) // 100 tokens/sec
	for i := 0; i < 2; i++ {
		ok, _ := l.allow("k2")
		if !ok {
			t.Fatalf("burst denied at i=%d", i)
		}
	}
	if ok, _ := l.allow("k2"); ok {
		t.Fatal("3rd call should be denied immediately")
	}
	// Wait long enough for at least one token to refill.
	time.Sleep(50 * time.Millisecond)
	if ok, _ := l.allow("k2"); !ok {
		t.Fatal("call after refill window should succeed")
	}
}

// TestAgentRateLimiter_PerKeyIsolation: separate keys have separate
// buckets , exhausting one doesn't block another.
func TestAgentRateLimiter_PerKeyIsolation(t *testing.T) {
	t.Parallel()
	l := newAgentRateLimiter(1, 0.0001, time.Hour)
	if ok, _ := l.allow("a"); !ok {
		t.Fatal("first call for a denied")
	}
	if ok, _ := l.allow("a"); ok {
		t.Fatal("second call for a allowed")
	}
	// Different key, fresh bucket.
	if ok, _ := l.allow("b"); !ok {
		t.Fatal("call for b should succeed; per-key isolation broken")
	}
}

// TestAgentRateLimiter_EmptyKey is a free pass (matches consent
// limiter conventions). Used by tests that don't inject an
// AgentIdentity.
func TestAgentRateLimiter_EmptyKey(t *testing.T) {
	t.Parallel()
	l := newAgentRateLimiter(1, 0.0001, time.Hour)
	for i := 0; i < 100; i++ {
		if ok, _ := l.allow(""); !ok {
			t.Fatalf("empty key throttled at i=%d", i)
		}
	}
}

// TestAgentRateLimiter_IdleGC: a key not seen for idleAfter is
// pruned to keep the buckets map bounded.
func TestAgentRateLimiter_IdleGC(t *testing.T) {
	t.Parallel()
	l := newAgentRateLimiter(2, 0.0001, 10*time.Millisecond)
	l.allow("kgc")
	if len(l.buckets) != 1 {
		t.Fatalf("setup: expected 1 bucket, got %d", len(l.buckets))
	}
	time.Sleep(15 * time.Millisecond)
	// Trigger sweep via any allow() call.
	l.allow("other")
	if _, present := l.buckets["kgc"]; present {
		t.Error("kgc bucket should have been GC'd after idleAfter")
	}
}
