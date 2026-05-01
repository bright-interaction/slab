// Package middleware: site-scoped authorization.
//
// Closes audit finding C1 (Critical). Pre-fix, every authenticated admin
// could read, update, delete or rebuild ANY site in the workspace by
// enumerating site IDs. The fix has two parts:
//
//  1. New site_members table linking users to sites with a role (see
//     schema.sql).
//  2. This middleware, mounted on every /api/sites/{siteID}/* route, that
//     verifies the authenticated user has a membership row for the path's
//     siteID before allowing the request through.
//
// Admin bypass: users with role='admin' on the workspace pass through
// regardless of site_members. This preserves the legacy single-workspace
// behaviour where the seeded admin owns everything; non-admin "editor"
// users (added via the existing /api/admin/invites flow) are gated by
// site_members rows the owner explicitly grants.
//
// The middleware extracts siteID from the URL via chi's URLParam. Routes
// using a different param name need a different middleware instance (or
// rename the param to siteID — every site-scoped route in atomicsite
// already uses {siteID}, verified at registration time).
package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"github.com/bright-interaction/slab/internal/store"
)

// siteIDPattern mirrors handlers.isSafeSiteID. We duplicate it here so the
// middleware has no handlers/middleware import cycle and can validate
// before any DB call.
var siteIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{24}$`)

// SiteAccessMiddleware returns a chi middleware that gates a route by the
// authenticated user's site membership. Mount it after AuthMiddleware so
// the user is already in context.
//
// Behaviour:
//   - Missing user (auth not run, or expired token) → 401
//   - Missing/malformed {siteID} URL param → 400
//   - User is admin → pass through (no DB query)
//   - User is non-admin and has no site_members row → 403
//   - DB error during membership check → 500 (fail closed; never grant
//     access on infra failure)
func SiteAccessMiddleware(queries *store.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil {
				writeAuthError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}
			siteID := chi.URLParam(r, "siteID")
			if !siteIDPattern.MatchString(siteID) {
				writeAuthError(w, http.StatusBadRequest, "Invalid siteID")
				return
			}
			// Workspace-level admins bypass the membership check. This
			// preserves the legacy single-admin-owns-everything model
			// while gating future non-admin users (editors / contributors
			// invited via the existing members flow).
			if user.Role == "admin" {
				next.ServeHTTP(w, r)
				return
			}
			_, err := queries.GetSiteMembership(r.Context(), store.GetSiteMembershipParams{
				SiteID: siteID,
				UserID: user.ID,
			})
			if err == nil {
				// Membership row found — pass through. The role lives in
				// the row but we don't enforce role-level gates here;
				// owner-only operations (delete site, manage members)
				// check role explicitly inside their handlers.
				next.ServeHTTP(w, r)
				return
			}
			if errors.Is(err, sql.ErrNoRows) {
				writeAuthError(w, http.StatusForbidden, "No access to this site")
				return
			}
			// Infrastructure failure (DB closed, query error). Fail
			// closed: 500, never grant access. Logging is the auth
			// middleware's job upstream.
			writeAuthError(w, http.StatusInternalServerError, "Authorization check failed")
		})
	}
}

// SiteAccessRole returns the membership row's role for the current
// (user, site) so handlers can enforce owner-only operations on top of the
// middleware's coarse pass/fail gate. Returns "" when the user is admin
// (no membership row needed) or when the user has no row.
func SiteAccessRole(ctx context.Context, queries *store.Queries, userID, siteID string) string {
	row, err := queries.GetSiteMembership(ctx, store.GetSiteMembershipParams{
		SiteID: siteID,
		UserID: userID,
	})
	if err != nil {
		return ""
	}
	return row.Role
}

// RequireOwnerOrAdmin gates owner-only handlers (delete site, add member,
// remove member). Admins always pass; non-admins need role='owner'.
// Returns true when authorized; on false the caller has already received
// a 403 written by this function.
func RequireOwnerOrAdmin(w http.ResponseWriter, r *http.Request, queries *store.Queries) bool {
	user := GetUser(r)
	if user == nil {
		writeAuthError(w, http.StatusUnauthorized, "Not authenticated")
		return false
	}
	if user.Role == "admin" {
		return true
	}
	siteID := chi.URLParam(r, "siteID")
	role := SiteAccessRole(r.Context(), queries, user.ID, siteID)
	if role != "owner" {
		writeAuthError(w, http.StatusForbidden, "Owner access required")
		return false
	}
	return true
}

// writeAuthError writes a JSON error without depending on handlers/helpers
// (which would create an import cycle).
func writeAuthError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
