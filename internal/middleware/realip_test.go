package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustedProxyRealIP(t *testing.T) {
	t.Run("empty trust list is a no-op", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.7:55432"
		req.Header.Set("X-Forwarded-For", "1.2.3.4, 192.168.1.1")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "203.0.113.7:55432" {
			t.Fatalf("expected RemoteAddr untouched, got %q", seen)
		}
	})

	t.Run("trusted peer gets XFF resolved to leftmost untrusted", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("127.0.0.1/32, 10.0.0.0/8")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		// Right-most is the proxy nearest to us (trusted), then a
		// trusted LB, then the real client.
		req.Header.Set("X-Forwarded-For", "203.0.113.42, 10.0.0.4, 10.0.0.5")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "203.0.113.42:0" {
			t.Fatalf("expected client IP from XFF, got %q", seen)
		}
	})

	t.Run("untrusted peer cannot rewrite via XFF", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("127.0.0.1/32")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		// Direct internet peer trying to spoof XFF.
		req.RemoteAddr = "203.0.113.99:44444"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "203.0.113.99:44444" {
			t.Fatalf("untrusted peer's XFF should be ignored, got %q", seen)
		}
	})

	t.Run("XFF takes precedence over X-Real-IP", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("127.0.0.1/32")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "203.0.113.10")
		req.Header.Set("X-Real-IP", "9.9.9.9")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "203.0.113.10:0" {
			t.Fatalf("XFF should win, got %q", seen)
		}
	})

	t.Run("X-Real-IP fallback when XFF absent and peer is trusted", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("127.0.0.1/32")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Set("X-Real-IP", "203.0.113.55")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "203.0.113.55:0" {
			t.Fatalf("expected X-Real-IP, got %q", seen)
		}
	})

	t.Run("malformed XFF entries are skipped", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("127.0.0.1/32")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "not-an-ip, 203.0.113.1")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "203.0.113.1:0" {
			t.Fatalf("malformed entry should be skipped, got %q", seen)
		}
	})

	t.Run("bare IP in trust list works", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		// Bare IP, no /32.
		mw := TrustedProxyRealIP("127.0.0.1")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "203.0.113.77")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "203.0.113.77:0" {
			t.Fatalf("bare-IP trust entry should match exactly, got %q", seen)
		}
	})

	t.Run("IPv6 trust + IPv6 XFF round-trip", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("::1/128")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "[::1]:8080"
		req.Header.Set("X-Forwarded-For", "2001:db8::1")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "[2001:db8::1]:0" {
			t.Fatalf("IPv6 client IP round-trip failed, got %q", seen)
		}
	})

	t.Run("trailing whitespace in single-entry XFF still resolves", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("127.0.0.1/32")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "  203.0.113.99  ")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "203.0.113.99:0" {
			t.Fatalf("trailing whitespace single-entry must trim, got %q", seen)
		}
	})

	t.Run("multiple X-Forwarded-For headers are joined for trust-walk", func(t *testing.T) {
		// Defense against the "send a second XFF header to hide a hop"
		// attack. r.Header.Values gives us every value; we concatenate
		// before splitting so every hop is in the chain.
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("127.0.0.1/32, 10.0.0.0/8")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		// First header carries the real client; second header would
		// be the attacker's spoof. Both must reach the trust-walk.
		req.Header.Add("X-Forwarded-For", "203.0.113.50")
		req.Header.Add("X-Forwarded-For", "10.0.0.5")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		// Right-most (closest hop) is the trusted 10.0.0.5; one hop
		// further left is 203.0.113.50, the real client.
		if seen != "203.0.113.50:0" {
			t.Fatalf("expected real client through joined headers, got %q", seen)
		}
	})

	t.Run("multiple X-Real-IP headers fall back gracefully", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("127.0.0.1/32")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Add("X-Real-IP", "")
		req.Header.Add("X-Real-IP", "203.0.113.77")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "203.0.113.77:0" {
			t.Fatalf("X-Real-IP fallback should pick first valid value, got %q", seen)
		}
	})

	t.Run("fully-trusted XFF chain falls back to leftmost", func(t *testing.T) {
		var seen string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.RemoteAddr
		})
		mw := TrustedProxyRealIP("10.0.0.0/8, 127.0.0.1/32")(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
		mw.ServeHTTP(httptest.NewRecorder(), req)

		if seen != "10.0.0.1:0" {
			t.Fatalf("expected leftmost fallback, got %q", seen)
		}
	})
}
