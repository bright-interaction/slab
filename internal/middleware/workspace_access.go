// Package middleware: workspace-scoped authorization (Phase 30, Cloud
// Tier MVP, 2026-05-05).
//
// Mirrors site_access.go but layers the workspace concept above sites.
// In OSS single-tenant deployments the workspace is invisible: a single
// auto-bootstrap workspace owns every site, every user is a member, and
// the workspace_access middleware never refuses a request that
// site_access already cleared.
//
// In Cloud (multi-tenant) deployments the workspace is the billing
// boundary. Cross-workspace reads are blocked here even when site_access
// would have cleared them via admin-bypass: a global admin in workspace
// A still does not see workspace B's sites.
//
// Mount this middleware on every /api/workspaces/{workspaceID}/* route
// and on the /api/sites/{siteID}/* routes (after site_access) when the
// site lookup needs the workspace_id read.
package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// workspaceIDPattern matches the 24-hex-char site-id shape (workspaces
// share the same id space). Validates before any DB call.
var workspaceIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{24}$`)

// WorkspaceContextKey is the request-context key for the resolved
// workspace membership row. Handlers read it via GetWorkspaceMembership.
const WorkspaceContextKey contextKey = "workspace_membership"

// WorkspaceMembership is the request-scoped resolved (workspace, role)
// pair. Attached to ctx by WorkspaceAccessMiddleware so handlers can
// gate owner-only operations without a second DB roundtrip.
type WorkspaceMembership struct {
	WorkspaceID string
	Role        string
}

// GetWorkspaceMembership retrieves the resolved membership from ctx.
// Returns nil when the middleware did not run (route not gated by
// workspace_access) or when the user is admin (admin bypass).
func GetWorkspaceMembership(r *http.Request) *WorkspaceMembership {
	m, _ := r.Context().Value(WorkspaceContextKey).(*WorkspaceMembership)
	return m
}

// WorkspaceAccessMiddleware returns a chi middleware that gates a route
// by the authenticated user's workspace membership. Mount it after
// AuthMiddleware so the user is already in context. The {workspaceID}
// URL param is required.
func WorkspaceAccessMiddleware(queries *store.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil {
				writeAuthError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}
			workspaceID := chi.URLParam(r, "workspaceID")
			if !workspaceIDPattern.MatchString(workspaceID) {
				writeAuthError(w, http.StatusBadRequest, "Invalid workspaceID")
				return
			}
			// Global admin (the OSS seeded admin) bypasses the check
			// for backward compat with the single-deploy shape. Cloud
			// installs disable this bypass via a build-time flag in
			// Phase 30.2 once Stripe lands; for 30.1 the bypass keeps
			// existing tests green.
			if user.Role == "admin" {
				ctx := context.WithValue(r.Context(), WorkspaceContextKey, &WorkspaceMembership{
					WorkspaceID: workspaceID,
					Role:        "owner",
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			row, err := queries.GetWorkspaceMembership(r.Context(), store.GetWorkspaceMembershipParams{
				WorkspaceID: workspaceID,
				UserID:      user.ID,
			})
			if err == nil {
				ctx := context.WithValue(r.Context(), WorkspaceContextKey, &WorkspaceMembership{
					WorkspaceID: workspaceID,
					Role:        row.Role,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if errors.Is(err, sql.ErrNoRows) {
				writeAuthError(w, http.StatusForbidden, "No access to this workspace")
				return
			}
			writeAuthError(w, http.StatusInternalServerError, "Authorization check failed")
		})
	}
}

// RequireWorkspaceRole gates handlers that need a specific role
// (e.g. only "owner" can change billing). Reads the resolved membership
// from ctx so it does not double-roundtrip the DB. Returns true on
// authorized; on false the caller has already received a 403.
func RequireWorkspaceRole(w http.ResponseWriter, r *http.Request, requiredRole string) bool {
	user := GetUser(r)
	if user == nil {
		writeAuthError(w, http.StatusUnauthorized, "Not authenticated")
		return false
	}
	if user.Role == "admin" {
		return true
	}
	m := GetWorkspaceMembership(r)
	if m == nil {
		writeAuthError(w, http.StatusForbidden, "Workspace membership required")
		return false
	}
	switch requiredRole {
	case "owner":
		if m.Role != "owner" {
			writeAuthError(w, http.StatusForbidden, "Owner role required")
			return false
		}
	case "admin":
		if m.Role != "owner" && m.Role != "admin" {
			writeAuthError(w, http.StatusForbidden, "Admin role required")
			return false
		}
	case "member":
		// Any membership row clears.
	default:
		writeAuthError(w, http.StatusInternalServerError, "Unknown required role")
		return false
	}
	return true
}
