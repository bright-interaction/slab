package handlers

import (
	"testing"
	"time"
)

func TestConsentRateLimiter_BurstThenThrottle(t *testing.T) {
	// Capacity 5, refill 1/sec. The first 5 calls should pass; the 6th
	// should be denied; after time advances enough to refill one token
	// the next call should pass again.
	rl := newConsentRateLimiter(5, 1, time.Minute)
	for i := 0; i < 5; i++ {
		if !rl.allow("1.2.3.4") {
			t.Fatalf("burst call %d should be allowed", i+1)
		}
	}
	if rl.allow("1.2.3.4") {
		t.Error("6th call should be denied (burst exhausted)")
	}
}

func TestConsentRateLimiter_PerIPIsolation(t *testing.T) {
	rl := newConsentRateLimiter(2, 1, time.Minute)
	if !rl.allow("1.1.1.1") || !rl.allow("1.1.1.1") {
		t.Fatal("first two calls from IP A should pass")
	}
	if rl.allow("1.1.1.1") {
		t.Fatal("third call from IP A should be denied")
	}
	// Different IP gets its own bucket.
	if !rl.allow("2.2.2.2") {
		t.Error("IP B must not be affected by IP A's bucket")
	}
}

func TestConsentRateLimiter_EmptyKeyAlwaysAllowed(t *testing.T) {
	rl := newConsentRateLimiter(1, 1, time.Minute)
	for i := 0; i < 100; i++ {
		if !rl.allow("") {
			t.Fatalf("empty key should never be throttled (proxy-misconfig fallback); failed at i=%d", i)
		}
	}
}

func TestConsentRateLimiter_Sweep(t *testing.T) {
	rl := newConsentRateLimiter(5, 1, 10*time.Millisecond)
	rl.allow("1.1.1.1")
	if len(rl.buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(rl.buckets))
	}
	time.Sleep(20 * time.Millisecond)
	rl.sweep()
	if len(rl.buckets) != 0 {
		t.Errorf("sweep should evict idle bucket, %d remain", len(rl.buckets))
	}
}
