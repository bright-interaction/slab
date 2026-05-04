package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/brightinteraction/atomicsite/internal/config"
	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

type AuthHandler struct {
	cfg     *config.Config
	queries *store.Queries
	// loginLimiter throttles failed logins per (email, ip). Closes audit
	// finding H3. Configured at NewAuthHandler with 5 failures / 15min
	// lockout , defeats both single-account brute force and credential
	// spray from one host.
	loginLimiter *loginRateLimiter
	// MailSender is the optional integration that delivers password-
	// reset links via email. nil means "log the link to slog so an
	// operator can deliver it manually" , appropriate for self-host
	// single-admin deployments where setting up SMTP is overkill.
	MailSender MailSender
}

func NewAuthHandler(cfg *config.Config, queries *store.Queries) *AuthHandler {
	return &AuthHandler{
		cfg:          cfg,
		queries:      queries,
		loginLimiter: newLoginRateLimiter(5, 15*time.Minute, time.Hour),
	}
}

// loginKey returns the per-email throttle key. We lowercase + trim because
// "Admin@Example.com" and "admin@example.com" should share a counter (an
// attacker won't get extra attempts by varying case).
func loginKey(prefix, value string) string {
	return prefix + ":" + strings.ToLower(strings.TrimSpace(value))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// Two-axis rate limit: per-email AND per-IP. Either hitting the
	// threshold blocks the request with 429 + Retry-After. The check
	// happens BEFORE the bcrypt compare so a locked-out attacker wastes
	// no server CPU; bcrypt is the most expensive operation in this
	// handler at ~10ms per call.
	emailKey := loginKey("e", req.Email)
	ipKey := loginKey("i", clientIP(r))
	for _, key := range []string{emailKey, ipKey} {
		if allowed, retry := h.loginLimiter.allow(key); !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(retry))
			writeError(w, http.StatusTooManyRequests, "Too many failed login attempts; try again later")
			return
		}
	}

	user, err := h.queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		// Record failure on both axes so the limiter can lock either.
		h.loginLimiter.recordFailure(emailKey)
		h.loginLimiter.recordFailure(ipKey)
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.loginLimiter.recordFailure(emailKey)
		h.loginLimiter.recordFailure(ipKey)
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Successful login: reset both counters so an honest user who
	// mistyped doesn't get punished after typing it right.
	h.loginLimiter.recordSuccess(emailKey)
	h.loginLimiter.recordSuccess(ipKey)

	authUser := &authmw.AuthUser{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
	}

	tokenStr, err := authmw.SignToken(h.cfg, authUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	authmw.SetTokenCookie(w, h.cfg, tokenStr)
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	authmw.ClearTokenCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := authmw.GetUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	row, err := h.queries.GetUserByID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":         row.ID,
			"email":      row.Email,
			"name":       row.Name,
			"role":       row.Role,
			"created_at": row.CreatedAt,
			"updated_at": row.UpdatedAt,
		},
	})
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := authmw.GetUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	// Audit L5: bump from 8 to 12 chars. NIST SP 800-63B current
	// guidance is 8 minimum but 12+ for accounts with system-wide
	// privileges (atomicsite admins fall in this bucket). bcrypt at
	// cost 10 + a 12-char passphrase puts even a multi-GPU offline
	// crack out of reach for any realistic attacker.
	if len(req.NewPassword) < 12 {
		writeError(w, http.StatusBadRequest, "Password must be at least 12 characters")
		return
	}

	row, err := h.queries.GetUserByID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load user")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		writeError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	if err := h.queries.UpdateUserPassword(r.Context(), store.UpdateUserPasswordParams{
		PasswordHash: string(hash),
		ID:           user.ID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SignOutEverywhere revokes every JWT for the current user (including this
// browser) by bumping token_version, then re-issues a fresh cookie so the
// caller stays signed in here.
func (h *AuthHandler) SignOutEverywhere(w http.ResponseWriter, r *http.Request) {
	user := authmw.GetUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	if err := h.queries.IncrementTokenVersion(r.Context(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to revoke sessions")
		return
	}

	row, err := h.queries.GetUserByID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reload user")
		return
	}

	authUser := &authmw.AuthUser{
		ID:           row.ID,
		Email:        row.Email,
		Name:         row.Name,
		Role:         row.Role,
		TokenVersion: row.TokenVersion,
	}
	tokenStr, err := authmw.SignToken(h.cfg, authUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to refresh session")
		return
	}
	authmw.SetTokenCookie(w, h.cfg, tokenStr)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
