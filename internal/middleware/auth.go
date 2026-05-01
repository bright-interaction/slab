// Package middleware provides HTTP middleware for authentication.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/store"
)

type contextKey string

const UserContextKey contextKey = "auth_user"

// AuthUser is the request-scoped user after successful authentication.
type AuthUser struct {
	ID           string
	Email        string
	Name         string
	Role         string
	TokenVersion int64
}

// GetUser retrieves the authenticated user from the request context.
func GetUser(r *http.Request) *AuthUser {
	u, _ := r.Context().Value(UserContextKey).(*AuthUser)
	return u
}

// SignToken creates a JWT token for the user.
func SignToken(cfg *config.Config, user *AuthUser) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"tv":    user.TokenVersion,
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// SetTokenCookie writes the JWT token as an HTTP-only, SameSite=Strict
// cookie. Strict (not Lax) is correct here: every legitimate dashboard
// navigation originates from the admin SPA on the same origin as the
// API. Lax would allow a cross-site POST from a malicious built-site
// subdomain (`*.atomicsite.example.com`) to land authenticated
// on the admin host's `/api/sites/...` routes; Strict blocks that
// entirely. The audit's H2 finding.
//
// Secure is set whenever the deployment is not local-dev. The previous
// "not localhost" check missed proxied http://prod deployments; the
// IsLocalDev helper covers 127.0.0.1 and 0.0.0.0 too.
func SetTokenCookie(w http.ResponseWriter, cfg *config.Config, tokenStr string) {
	secure := !cfg.IsLocalDev()
	http.SetCookie(w, &http.Cookie{
		Name:     "atomicsite_token",
		Value:    tokenStr,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearTokenCookie removes the auth cookie.
func ClearTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "atomicsite_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// AuthMiddleware authenticates requests via JWT cookie.
type AuthMiddleware struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewAuthMiddleware(cfg *config.Config, queries *store.Queries) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg, queries: queries}
}

func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := m.authenticate(r)
		if err != nil {
			writeUnauth(w, err.Error())
			return
		}
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin wraps a handler so it 403s for any authenticated non-admin.
// Must be mounted *inside* an authenticated route group so the user is
// already attached to the request context.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := GetUser(r)
		if u == nil {
			http.Error(w, `{"error":"Not authenticated"}`, http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			return
		}
		if u.Role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Admin role required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) authenticate(r *http.Request) (*AuthUser, error) {
	cookie, err := r.Cookie("atomicsite_token")
	if err != nil {
		return nil, errNoAuth
	}

	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errInvalidToken
		}
		return []byte(m.cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return nil, errInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errInvalidToken
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errInvalidToken
	}

	row, err := m.queries.GetUserByID(r.Context(), sub)
	if err != nil {
		return nil, errUserNotFound
	}

	// Check token version
	tv, _ := claims["tv"].(float64)
	if int64(tv) != row.TokenVersion {
		return nil, errSessionInvalidated
	}

	return &AuthUser{
		ID:           row.ID,
		Email:        row.Email,
		Name:         row.Name,
		Role:         row.Role,
		TokenVersion: row.TokenVersion,
	}, nil
}

type authError string

func (e authError) Error() string { return string(e) }

const (
	errNoAuth             authError = "Authorization required"
	errUserNotFound       authError = "User not found"
	errInvalidToken       authError = "Invalid or expired token"
	errSessionInvalidated authError = "Session invalidated. Please log in again."
)

func writeUnauth(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
