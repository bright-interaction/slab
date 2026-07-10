// Package middleware - mfa_enforce.go.
//
// EnforceMFAEnrollment refuses POST/PUT/PATCH/DELETE traffic for any
// authenticated user the policy says must enroll in TOTP and who has
// not yet enrolled. Reads pass through unchanged so the dashboard can
// render the enroll-MFA screen + the existing settings the user
// needs to navigate.
//
// Whitelisted endpoints (always allowed regardless of enrollment
// status) are the TOTP setup flow itself plus log-out + change-
// password. Without these the user could log in but never get out
// of the enroll loop.
//
// The policy itself lives in cfg.MustEnrollMFA(role); see
// internal/config/config.go. Process-wide so the auth path and this
// middleware share one source of truth.

package middleware

import (
	"net/http"
	"strings"

	"github.com/bright-interaction/atomicsite/internal/config"
	"github.com/bright-interaction/atomicsite/internal/store"
)

// mfaEnrollmentLookup loads totp_enrolled_at for a user. The
// production wiring uses *store.Queries.GetUserByID; tests can
// inject a fake to skip the DB layer.
type mfaEnrollmentLookup func(r *http.Request, userID string) (totpEnrolledAt string, ok bool)

// EnforceMFAEnrollment returns the middleware. Pass cfg + the
// queries handle (used to look up the live totp_enrolled_at value;
// JWT claims don't carry it because rotating without a re-issue
// would be silent).
func EnforceMFAEnrollment(cfg *config.Config, queries *store.Queries) func(http.Handler) http.Handler {
	return enforceMFAEnrollment(cfg, func(r *http.Request, userID string) (string, bool) {
		row, err := queries.GetUserByID(r.Context(), userID)
		if err != nil {
			return "", false
		}
		return row.TotpEnrolledAt, true
	})
}

func enforceMFAEnrollment(cfg *config.Config, lookup mfaEnrollmentLookup) func(http.Handler) http.Handler {
	if cfg == nil || cfg.RequireMFA == config.MFAOptional {
		// Policy off: pass-through. Saves the per-request DB lookup
		// when the deployment doesn't enforce MFA at all.
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isWriteMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			if isMFAEnrollmentWhitelisted(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			user := GetUser(r)
			if user == nil {
				// Not authenticated: let auth middleware handle the
				// 401. (We don't normally get called pre-auth, but
				// being defensive doesn't hurt.)
				next.ServeHTTP(w, r)
				return
			}
			if !cfg.MustEnrollMFA(user.Role) {
				next.ServeHTTP(w, r)
				return
			}
			enrolledAt, ok := lookup(r, user.ID)
			if !ok {
				// DB miss: fail closed. The user shouldn't slip
				// through MFA enforcement just because we couldn't
				// confirm their state.
				writeMFAEnrollError(w)
				return
			}
			// TrimSpace before the empty-check so a whitespace-only
			// totp_enrolled_at (corrupt write or upstream bug) can't
			// be treated as "enrolled". The DB column is supposed to
			// hold an ISO 8601 timestamp; anything that whitespaces
			// to empty is by definition not a real enrollment.
			if strings.TrimSpace(enrolledAt) == "" {
				writeMFAEnrollError(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isMFAEnrollmentWhitelisted returns true for the small set of write
// endpoints that must remain reachable when an unenrolled user is
// trying to enroll: TOTP setup + verify, change-password (so a fresh
// invite can rotate the seeded password), and logout. Everything
// else 403s with error_code=mfa_enroll_required.
func isMFAEnrollmentWhitelisted(p string) bool {
	switch p {
	case "/api/auth/totp/setup",
		"/api/auth/totp/verify",
		"/api/auth/totp/disable",
		"/api/auth/change-password",
		"/api/auth/logout",
		"/api/auth/sign-out-everywhere":
		return true
	}
	// Allow any /api/auth/totp/* path so future MFA flow additions
	// don't accidentally lock users out.
	return strings.HasPrefix(p, "/api/auth/totp/")
}

func writeMFAEnrollError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"MFA enrollment required before performing this action","error_code":"mfa_enroll_required"}`))
}
