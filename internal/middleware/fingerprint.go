package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/config"
)

// FingerprintCookieName is the HttpOnly cookie used to identify a returning
// visitor across requests on a built site. The value is 16 hex chars derived
// from a hash of (client-ip, ua, accept-language, AnalyticsSalt) on first
// contact, then echoed back on every subsequent request. It does NOT identify
// a person: pre-consent it is anonymous, post-consent the analytics layer
// upgrades it to an identified visitor by writing visit_sessions.visitor_id.
const FingerprintCookieName = "atomicsite_fp"

// FingerprintMaxAge is the cookie lifetime: 1 year. Anything shorter would
// fragment "returning visitor" stats; anything longer is meaningless because
// browsers cap at 400 days and most clear cookies sooner.
const FingerprintMaxAge = 31536000 // 1 year in seconds

// fingerprintContextKey is the ctx key under which Fingerprint() exposes the
// 16-hex-char fingerprint that the handler should use for visit_sessions.
type fingerprintContextKey struct{}

// GetFingerprint returns the fingerprint set on the request context by
// FingerprintMiddleware. Returns "" if the middleware did not run.
func GetFingerprint(r *http.Request) string {
	v, _ := r.Context().Value(fingerprintContextKey{}).(string)
	return v
}

// FingerprintMiddleware ensures every request has an atomicsite_fp cookie. If
// the client already sent one, we trust it and pass it through. Otherwise we
// derive a stable hash from request signals + the server-side AnalyticsSalt,
// truncate to 16 hex chars (64 bits of entropy, plenty for visitor uniqueness)
// and set the cookie HttpOnly + Secure (in prod) + SameSite=Lax + 1y.
//
// The cookie is NOT cryptographic: it's a stable identifier, not a session
// token. SameSite=Lax is fine because the only thing it gates is the analytics
// receiver, which has no auth anyway.
//
// Mount this on the /t/* group only. Putting it on the whole app would set
// fingerprint cookies on every API request, which both pollutes admin sessions
// and tortures any future caching layer.
func FingerprintMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	secure := !strings.HasPrefix(cfg.BaseURL, "http://localhost")
	salt := cfg.AnalyticsSalt
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fp := readFingerprintCookie(r)
			if fp == "" {
				fp = deriveFingerprint(r, salt)
				http.SetCookie(w, &http.Cookie{
					Name:     FingerprintCookieName,
					Value:    fp,
					Path:     "/",
					MaxAge:   FingerprintMaxAge,
					HttpOnly: true,
					Secure:   secure,
					SameSite: http.SameSiteLaxMode,
				})
			}
			ctx := context.WithValue(r.Context(), fingerprintContextKey{}, fp)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// readFingerprintCookie returns the existing fingerprint cookie value if it
// looks like one we set (16 lowercase hex chars). Anything else gets ignored
// and we mint a fresh value, so we never trust client-supplied junk.
func readFingerprintCookie(r *http.Request) string {
	c, err := r.Cookie(FingerprintCookieName)
	if err != nil {
		return ""
	}
	v := c.Value
	if len(v) != 16 {
		return ""
	}
	for i := 0; i < len(v); i++ {
		ch := v[i]
		isHex := (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')
		if !isHex {
			return ""
		}
	}
	return v
}

// deriveFingerprint hashes the client IP, UA, accept-language, and salt into a
// 16-hex-char identifier. Two visitors behind the same NAT with identical
// browser strings collide, which is acceptable: the goal is "stable enough to
// stitch a session" not "globally unique".
func deriveFingerprint(r *http.Request, salt string) string {
	ip := clientIP(r)
	ua := r.Header.Get("User-Agent")
	al := r.Header.Get("Accept-Language")
	h := sha256.New()
	h.Write([]byte(ip))
	h.Write([]byte{0})
	h.Write([]byte(ua))
	h.Write([]byte{0})
	h.Write([]byte(al))
	h.Write([]byte{0})
	h.Write([]byte(salt))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:8])
}

// clientIP best-effort extracts the client IP. We honour X-Forwarded-For
// because chi's RealIP middleware (mounted earlier in the stack) sets
// r.RemoteAddr from it. If neither is present, RemoteAddr survives as the
// connection peer.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
