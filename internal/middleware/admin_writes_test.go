package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newAdminWriteHandler(t *testing.T, lim *AdminWriteLimiter) http.Handler {
	t.Helper()
	pass := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return AdminWriteRateLimit(lim)(pass)
}

func authedReq(method, path, userID, remote string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.RemoteAddr = remote
	if userID != "" {
		ctx := context.WithValue(r.Context(), UserContextKey, &AuthUser{ID: userID})
		r = r.WithContext(ctx)
	}
	return r
}

func TestAdminWriteRateLimit_NilLimiterIsPassthrough(t *testing.T) {
	h := AdminWriteRateLimit(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
		if w.Code != http.StatusOK {
			t.Fatalf("nil limiter should pass everything; iter %d got %d", i, w.Code)
		}
	}
}

func TestAdminWriteRateLimit_ReadsBypass(t *testing.T) {
	// Tiny capacity; if the limiter were applied to reads, the second
	// GET would 429.
	lim := NewAdminWriteLimiter(1, 0.001, time.Minute)
	h := newAdminWriteHandler(t, lim)
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, authedReq("GET", "/x", "user-a", "1.2.3.4:1"))
		if w.Code != http.StatusOK {
			t.Fatalf("GET should bypass write limiter; iter %d got %d", i, w.Code)
		}
	}
	for _, m := range []string{"HEAD", "OPTIONS"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, authedReq(m, "/x", "user-a", "1.2.3.4:1"))
		if w.Code != http.StatusOK {
			t.Fatalf("%s should bypass; got %d", m, w.Code)
		}
	}
}

func TestAdminWriteRateLimit_BurstThen429(t *testing.T) {
	// capacity 3, refill effectively never (in this test horizon).
	lim := NewAdminWriteLimiter(3, 0.001, time.Minute)
	h := newAdminWriteHandler(t, lim)

	// First 3 writes pass.
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
		if w.Code != http.StatusOK {
			t.Fatalf("burst write %d should pass; got %d", i, w.Code)
		}
	}
	// Fourth is throttled.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("over-burst write should 429; got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("429 response must set Retry-After header")
	}
}

func TestAdminWriteRateLimit_PerUserIsolation(t *testing.T) {
	lim := NewAdminWriteLimiter(2, 0.001, time.Minute)
	h := newAdminWriteHandler(t, lim)

	// User A burns its 2-token budget.
	for i := 0; i < 2; i++ {
		h.ServeHTTP(httptest.NewRecorder(), authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
	}
	wA := httptest.NewRecorder()
	h.ServeHTTP(wA, authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
	if wA.Code != http.StatusTooManyRequests {
		t.Fatalf("user A should be throttled; got %d", wA.Code)
	}
	// User B from the same IP still has its own bucket.
	wB := httptest.NewRecorder()
	h.ServeHTTP(wB, authedReq("POST", "/x", "user-b", "1.2.3.4:1"))
	if wB.Code != http.StatusOK {
		t.Fatalf("user B should not share user A's bucket; got %d", wB.Code)
	}
}

func TestAdminWriteRateLimit_PerIPIsolation(t *testing.T) {
	lim := NewAdminWriteLimiter(2, 0.001, time.Minute)
	h := newAdminWriteHandler(t, lim)

	// Same user, IP-1 burns budget.
	for i := 0; i < 2; i++ {
		h.ServeHTTP(httptest.NewRecorder(), authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
	}
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
	if w1.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on IP-1; got %d", w1.Code)
	}
	// Same user from a different IP still has its own bucket (this is
	// intentional: a shared-NAT office with two devices keeps working
	// even when one is busy).
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, authedReq("POST", "/x", "user-a", "5.6.7.8:1"))
	if w2.Code != http.StatusOK {
		t.Fatalf("different IP should have its own bucket; got %d", w2.Code)
	}
}

func TestAdminWriteRateLimit_UnauthAnonKey(t *testing.T) {
	lim := NewAdminWriteLimiter(1, 0.001, time.Minute)
	h := newAdminWriteHandler(t, lim)

	// No user in context: bucket key is "anon|<ip>". First passes,
	// second from same IP throttles.
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, authedReq("POST", "/x", "", "1.2.3.4:1"))
	if w1.Code != http.StatusOK {
		t.Fatalf("first anon write should pass; got %d", w1.Code)
	}
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, authedReq("POST", "/x", "", "1.2.3.4:1"))
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second anon write from same IP should throttle; got %d", w2.Code)
	}
}

func TestAdminWriteLimiter_Sweep(t *testing.T) {
	// maxIdle 1ns: any sleep guarantees the bucket is older than the
	// cutoff and Sweep() reaps it.
	lim := NewAdminWriteLimiter(2, 0.001, time.Nanosecond)
	lim.allow("user-a|1.1.1.1")
	lim.allow("user-b|2.2.2.2")
	if got := len(lim.buckets); got != 2 {
		t.Fatalf("expected 2 buckets, got %d", got)
	}
	time.Sleep(2 * time.Millisecond)
	lim.Sweep()
	if got := len(lim.buckets); got != 0 {
		t.Fatalf("Sweep should evict idle buckets; got %d", got)
	}
}

func TestAdminWriteLimiter_RefillEventuallyAllows(t *testing.T) {
	// 5 tokens/sec: in 50ms we should refill ~1 token after burst empty.
	lim := NewAdminWriteLimiter(1, 5, time.Minute)
	h := newAdminWriteHandler(t, lim)

	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
	if w1.Code != http.StatusOK {
		t.Fatalf("first write should pass; got %d", w1.Code)
	}
	// Second immediately: empty.
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second write should 429; got %d", w2.Code)
	}
	// Wait long enough for one token to refill (5 tokens/s = 200ms/token,
	// give 300ms slack).
	time.Sleep(300 * time.Millisecond)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, authedReq("POST", "/x", "user-a", "1.2.3.4:1"))
	if w3.Code != http.StatusOK {
		t.Fatalf("after refill the third write should pass; got %d", w3.Code)
	}
}
