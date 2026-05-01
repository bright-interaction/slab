package handlers

import (
	"testing"
	"time"
)

func TestLoginRateLimiter_AllowsBelowThreshold(t *testing.T) {
	l := newLoginRateLimiter(5, 15*time.Minute, time.Hour)
	for i := 0; i < 4; i++ {
		ok, _ := l.allow("e:test@example.com")
		if !ok {
			t.Errorf("attempt %d: expected allowed, got blocked", i+1)
		}
		l.recordFailure("e:test@example.com")
	}
}

func TestLoginRateLimiter_LocksAtThreshold(t *testing.T) {
	l := newLoginRateLimiter(3, 15*time.Minute, time.Hour)
	key := "e:bad@example.com"
	// 3 failures should trigger lockout on the 4th attempt.
	for i := 0; i < 3; i++ {
		l.allow(key)
		l.recordFailure(key)
	}
	ok, retry := l.allow(key)
	if ok {
		t.Error("expected blocked after 3 failures, got allowed")
	}
	// Retry should be in the ballpark of 15 minutes (~900s) but allow
	// some clock skew.
	if retry < 800 || retry > 1000 {
		t.Errorf("retryAfter = %d seconds; want ~900", retry)
	}
}

func TestLoginRateLimiter_SuccessClearsCounter(t *testing.T) {
	l := newLoginRateLimiter(5, 15*time.Minute, time.Hour)
	key := "e:user@example.com"
	// Two failures, then a success: counter should reset, allowing
	// the user another full window of attempts.
	l.recordFailure(key)
	l.recordFailure(key)
	l.recordSuccess(key)
	for i := 0; i < 4; i++ {
		ok, _ := l.allow(key)
		if !ok {
			t.Errorf("after success, attempt %d should be allowed", i+1)
		}
		l.recordFailure(key)
	}
}

func TestLoginRateLimiter_EmptyKeyAlwaysAllowed(t *testing.T) {
	l := newLoginRateLimiter(1, 15*time.Minute, time.Hour)
	// Even after recording many failures with empty key, allow stays true.
	for i := 0; i < 100; i++ {
		l.recordFailure("")
	}
	ok, _ := l.allow("")
	if !ok {
		t.Error("empty key should never lock; proxy-misconfig that strips IPs must not block legitimate logins")
	}
}

func TestLoginRateLimiter_LockoutExpires(t *testing.T) {
	// Use a tiny lockout window to test expiry without sleeping for 15 minutes.
	l := newLoginRateLimiter(2, 50*time.Millisecond, time.Hour)
	key := "e:expiry@example.com"
	l.recordFailure(key)
	l.recordFailure(key)
	if ok, _ := l.allow(key); ok {
		t.Fatal("expected lockout after 2 failures")
	}
	time.Sleep(100 * time.Millisecond)
	if ok, _ := l.allow(key); !ok {
		t.Error("lockout should have expired after 100ms; got still blocked")
	}
}

func TestLoginRateLimiter_TwoAxesIndependent(t *testing.T) {
	// An attacker spraying credentials from one IP across many emails
	// should hit the per-IP lockout, not exhaust per-email counters.
	l := newLoginRateLimiter(5, 15*time.Minute, time.Hour)
	ipKey := "i:198.51.100.5"
	for i := 0; i < 5; i++ {
		l.allow(ipKey)
		l.recordFailure(ipKey)
	}
	if ok, _ := l.allow(ipKey); ok {
		t.Error("per-IP key should be locked after 5 failures")
	}
	// Per-email key for the same attacks should NOT be locked, because
	// the attacker only sent one attempt against any given account.
	if ok, _ := l.allow("e:victim@example.com"); !ok {
		t.Error("per-email key should remain unlocked when only IP was abused")
	}
}

func TestLoginRateLimiter_Sweep(t *testing.T) {
	l := newLoginRateLimiter(5, 15*time.Minute, 50*time.Millisecond)
	l.recordFailure("e:stale@example.com")
	if len(l.failures) != 1 {
		t.Fatalf("setup: failures = %d, want 1", len(l.failures))
	}
	time.Sleep(100 * time.Millisecond)
	l.sweep()
	if len(l.failures) != 0 {
		t.Errorf("after sweep: failures = %d, want 0 (idle entries should evict)", len(l.failures))
	}
}
