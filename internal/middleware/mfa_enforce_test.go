package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bright-interaction/slab/internal/config"
)

// fakeLookup returns the given totp_enrolled_at for any user. Lets
// tests skip the DB layer entirely.
func fakeLookup(enrolledAt string, ok bool) mfaEnrollmentLookup {
	return func(_ *http.Request, _ string) (string, bool) {
		return enrolledAt, ok
	}
}

func mfaReq(method, path, userID, role string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	if userID != "" {
		ctx := context.WithValue(r.Context(), UserContextKey, &AuthUser{ID: userID, Role: role})
		r = r.WithContext(ctx)
	}
	return r
}

func mfaPassthrough() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestEnforceMFAEnrollment_PolicyOffPassesThrough(t *testing.T) {
	cfg := &config.Config{RequireMFA: config.MFAOptional}
	h := enforceMFAEnrollment(cfg, fakeLookup("", true))(mfaPassthrough())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, mfaReq("POST", "/api/sites/x", "user-a", "admin"))
	if w.Code != http.StatusOK {
		t.Fatalf("policy off should pass through; got %d", w.Code)
	}
}

func TestEnforceMFAEnrollment_AdminPolicyEditorPasses(t *testing.T) {
	cfg := &config.Config{RequireMFA: config.MFAAdminsOnly}
	h := enforceMFAEnrollment(cfg, fakeLookup("", true))(mfaPassthrough())
	// Editor role: not gated under "admin" policy.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, mfaReq("POST", "/api/sites/x", "user-a", "editor"))
	if w.Code != http.StatusOK {
		t.Fatalf("editor under admin-only policy should pass; got %d", w.Code)
	}
}

func TestEnforceMFAEnrollment_AdminUnenrolledBlocked(t *testing.T) {
	cfg := &config.Config{RequireMFA: config.MFAAdminsOnly}
	h := enforceMFAEnrollment(cfg, fakeLookup("", true))(mfaPassthrough())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, mfaReq("POST", "/api/sites/x", "user-a", "admin"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("unenrolled admin should 403; got %d", w.Code)
	}
	body := w.Body.String()
	if !contains(body, "mfa_enroll_required") {
		t.Errorf("response body should carry error_code=mfa_enroll_required, got %s", body)
	}
}

func TestEnforceMFAEnrollment_AdminEnrolledPasses(t *testing.T) {
	cfg := &config.Config{RequireMFA: config.MFAAdminsOnly}
	h := enforceMFAEnrollment(cfg, fakeLookup("2026-05-01T00:00:00Z", true))(mfaPassthrough())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, mfaReq("POST", "/api/sites/x", "user-a", "admin"))
	if w.Code != http.StatusOK {
		t.Fatalf("enrolled admin should pass; got %d", w.Code)
	}
}

func TestEnforceMFAEnrollment_AllPolicyEditorBlocked(t *testing.T) {
	cfg := &config.Config{RequireMFA: config.MFAAllUsers}
	h := enforceMFAEnrollment(cfg, fakeLookup("", true))(mfaPassthrough())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, mfaReq("POST", "/api/pages", "user-a", "editor"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("editor under all-users policy without enrollment should 403; got %d", w.Code)
	}
}

func TestEnforceMFAEnrollment_ReadsAlwaysPass(t *testing.T) {
	cfg := &config.Config{RequireMFA: config.MFAAllUsers}
	h := enforceMFAEnrollment(cfg, fakeLookup("", true))(mfaPassthrough())
	for _, m := range []string{"GET", "HEAD", "OPTIONS"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, mfaReq(m, "/api/sites/x", "user-a", "admin"))
		if w.Code != http.StatusOK {
			t.Errorf("%s should pass; got %d", m, w.Code)
		}
	}
}

func TestEnforceMFAEnrollment_TOTPSetupAlwaysAllowed(t *testing.T) {
	cfg := &config.Config{RequireMFA: config.MFAAllUsers}
	h := enforceMFAEnrollment(cfg, fakeLookup("", true))(mfaPassthrough())
	for _, p := range []string{
		"/api/auth/totp/setup",
		"/api/auth/totp/verify",
		"/api/auth/totp/disable",
		"/api/auth/change-password",
		"/api/auth/logout",
		"/api/auth/sign-out-everywhere",
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, mfaReq("POST", p, "user-a", "admin"))
		if w.Code != http.StatusOK {
			t.Errorf("%s should be whitelisted; got %d", p, w.Code)
		}
	}
}

func TestEnforceMFAEnrollment_WhitespaceTotpEnrolledAtBlocks(t *testing.T) {
	// A whitespace-only totp_enrolled_at must not be treated as
	// "enrolled". Defense against a corrupted DB write or upstream
	// bug that wrote " " into the column.
	cfg := &config.Config{RequireMFA: config.MFAAdminsOnly}
	h := enforceMFAEnrollment(cfg, fakeLookup("   ", true))(mfaPassthrough())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, mfaReq("POST", "/api/sites/x", "user-a", "admin"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("whitespace-only enrolled_at must NOT count as enrolled; got %d", w.Code)
	}
}

func TestEnforceMFAEnrollment_DBLookupFailFailsClosed(t *testing.T) {
	cfg := &config.Config{RequireMFA: config.MFAAdminsOnly}
	h := enforceMFAEnrollment(cfg, fakeLookup("", false))(mfaPassthrough())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, mfaReq("POST", "/api/sites/x", "user-a", "admin"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("DB lookup failure under MFA policy must fail closed; got %d", w.Code)
	}
}

func TestEnforceMFAEnrollment_AnonymousFallsThrough(t *testing.T) {
	cfg := &config.Config{RequireMFA: config.MFAAllUsers}
	h := enforceMFAEnrollment(cfg, fakeLookup("", true))(mfaPassthrough())
	// No user in context: defer to upstream auth (which will 401 in
	// reality). The MFA middleware itself must not 403.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, mfaReq("POST", "/api/sites/x", "", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("anonymous request should pass MFA layer (401 happens upstream); got %d", w.Code)
	}
}

func TestMustEnrollMFA_PolicyMatrix(t *testing.T) {
	cases := []struct {
		policy string
		role   string
		want   bool
	}{
		{config.MFAOptional, "admin", false},
		{config.MFAOptional, "editor", false},
		{config.MFAAdminsOnly, "admin", true},
		{config.MFAAdminsOnly, "editor", false},
		{config.MFAAdminsOnly, "support", false},
		{config.MFAAllUsers, "admin", true},
		{config.MFAAllUsers, "editor", true},
		{config.MFAAllUsers, "support", true},
	}
	for _, tc := range cases {
		cfg := &config.Config{RequireMFA: tc.policy}
		got := cfg.MustEnrollMFA(tc.role)
		if got != tc.want {
			t.Errorf("policy=%q role=%q -> %v, want %v", tc.policy, tc.role, got, tc.want)
		}
	}
}

func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
