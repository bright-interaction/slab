package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/bright-interaction/slab/internal/store"
)

// Phase 30.6 (2026-05-06): unit coverage for the cross-workspace
// bypass + audit emission paths in WorkspaceAccessMiddleware. The
// e2e harness can't naturally exercise these branches because it
// seeds a single admin user who is auto-added as owner on workspace
// creation; here we stand up isolated DB state with no
// workspace_members row and assert the bypass + audit_log writes
// behave as documented in workspace_access.go:60-138.

func seedUserAndWorkspace(t *testing.T, sqlDB *sql.DB, role, userID, wsID string) {
	t.Helper()
	if _, err := sqlDB.Exec(
		`INSERT INTO users (id, email, password_hash, name, role) VALUES (?, ?, '', '', ?)`,
		userID, userID+"@example.com", role,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO workspaces (id, name, slug) VALUES (?, ?, ?)`,
		wsID, "Workspace "+wsID, wsID,
	); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
}

func runWorkspaceRouted(t *testing.T, mw func(http.Handler) http.Handler, user *AuthUser, method, wsID string) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	router.With(mw).MethodFunc(method, "/api/workspaces/{workspaceID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("granted"))
	})
	req := httptest.NewRequest(method, "/api/workspaces/"+wsID, nil)
	if user != nil {
		ctx := context.WithValue(req.Context(), UserContextKey, user)
		req = req.WithContext(ctx)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestWorkspaceAccess_UnauthenticatedRejects(t *testing.T) {
	t.Parallel()
	_, q := openTestDB(t)
	rr := runWorkspaceRouted(t, WorkspaceAccessMiddleware(q), nil, http.MethodGet, "abcdef0123456789abcdef01")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", rr.Code)
	}
}

func TestWorkspaceAccess_InvalidWorkspaceIDRejects(t *testing.T) {
	t.Parallel()
	_, q := openTestDB(t)
	user := &AuthUser{ID: "u1", Role: "admin"}
	rr := runWorkspaceRouted(t, WorkspaceAccessMiddleware(q), user, http.MethodGet, "not-hex")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", rr.Code)
	}
}

func TestWorkspaceAccess_MemberHappyPath(t *testing.T) {
	t.Parallel()
	sqlDB, q := openTestDB(t)
	wsID := "abcdef0123456789abcdef01"
	seedUserAndWorkspace(t, sqlDB, "user", "u1", wsID)
	if err := q.AddWorkspaceMember(context.Background(), store.AddWorkspaceMemberParams{
		WorkspaceID: wsID,
		UserID:      "u1",
		Role:        "owner",
	}); err != nil {
		t.Fatalf("add member: %v", err)
	}
	user := &AuthUser{ID: "u1", Role: "user"}
	rr := runWorkspaceRouted(t, WorkspaceAccessMiddleware(q), user, http.MethodGet, wsID)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
	rows, err := q.ListAuditLogGlobal(context.Background(), 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("happy-path member access wrote %d audit rows; want 0", len(rows))
	}
}

func TestWorkspaceAccess_AdminBypassWritesAuditAndAllowsWrite(t *testing.T) {
	t.Parallel()
	sqlDB, q := openTestDB(t)
	wsID := "abcdef0123456789abcdef02"
	seedUserAndWorkspace(t, sqlDB, "admin", "admin1", wsID)
	user := &AuthUser{ID: "admin1", Role: "admin"}

	// Admin without a workspace_members row issues a PATCH (write).
	rr := runWorkspaceRouted(t, WorkspaceAccessMiddleware(q), user, http.MethodPatch, wsID)
	if rr.Code != http.StatusOK {
		t.Fatalf("admin bypass write: status=%d, body=%q; want 200", rr.Code, rr.Body.String())
	}
	rows, err := q.ListAuditLogGlobal(context.Background(), 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("audit rows=%d, want 1", len(rows))
	}
	row := rows[0]
	if row.Action != "cross_workspace_admin" {
		t.Errorf("action=%q, want cross_workspace_admin", row.Action)
	}
	if row.ResourceType != "workspace" || row.ResourceID != wsID {
		t.Errorf("resource=(%q,%q), want (workspace,%s)", row.ResourceType, row.ResourceID, wsID)
	}
	if row.ActorUserID != "admin1" || row.ActorRole != "admin" {
		t.Errorf("actor=(%q,%q), want (admin1,admin)", row.ActorUserID, row.ActorRole)
	}
	if !strings.Contains(row.ChangesJson, `"method":"PATCH"`) {
		t.Errorf("changes_json missing method: %q", row.ChangesJson)
	}
}

func TestWorkspaceAccess_SupportBypassReadOnly(t *testing.T) {
	t.Parallel()
	sqlDB, q := openTestDB(t)
	wsID := "abcdef0123456789abcdef03"
	seedUserAndWorkspace(t, sqlDB, "support", "sup1", wsID)
	user := &AuthUser{ID: "sup1", Role: "support"}

	// GET passes through with audit row.
	rr := runWorkspaceRouted(t, WorkspaceAccessMiddleware(q), user, http.MethodGet, wsID)
	if rr.Code != http.StatusOK {
		t.Fatalf("support GET: status=%d, want 200", rr.Code)
	}
	rows, err := q.ListAuditLogGlobal(context.Background(), 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(rows) != 1 || rows[0].Action != "cross_workspace_support" {
		t.Fatalf("after GET: rows=%d, action0=%q; want 1 / cross_workspace_support",
			len(rows), func() string {
				if len(rows) > 0 {
					return rows[0].Action
				}
				return ""
			}())
	}

	// Non-GET is refused with 403 and emits no further audit row.
	rrPatch := runWorkspaceRouted(t, WorkspaceAccessMiddleware(q), user, http.MethodPatch, wsID)
	if rrPatch.Code != http.StatusForbidden {
		t.Errorf("support PATCH: status=%d, want 403", rrPatch.Code)
	}
	rows2, _ := q.ListAuditLogGlobal(context.Background(), 10)
	if len(rows2) != 1 {
		t.Errorf("forbidden write should not write audit; rows=%d, want 1", len(rows2))
	}
}

func TestWorkspaceAccess_OtherRoleRefused(t *testing.T) {
	t.Parallel()
	sqlDB, q := openTestDB(t)
	wsID := "abcdef0123456789abcdef04"
	seedUserAndWorkspace(t, sqlDB, "user", "u2", wsID)
	user := &AuthUser{ID: "u2", Role: "user"}
	rr := runWorkspaceRouted(t, WorkspaceAccessMiddleware(q), user, http.MethodGet, wsID)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status=%d, want 403", rr.Code)
	}
	rows, _ := q.ListAuditLogGlobal(context.Background(), 10)
	if len(rows) != 0 {
		t.Errorf("403 path should not emit audit; rows=%d", len(rows))
	}
}

func TestRequireWorkspaceRole_AdminGlobalRoleClearsWithoutMembership(t *testing.T) {
	t.Parallel()
	user := &AuthUser{ID: "admin1", Role: "admin"}
	r := httptest.NewRequest(http.MethodPatch, "/x", nil)
	ctx := context.WithValue(r.Context(), UserContextKey, user)
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()
	if !RequireWorkspaceRole(rr, r, "owner") {
		t.Errorf("global admin should clear owner check; got %d", rr.Code)
	}
}

func TestRequireWorkspaceRole_MemberRoleEnforcesOwner(t *testing.T) {
	t.Parallel()
	user := &AuthUser{ID: "u1", Role: "user"}
	r := httptest.NewRequest(http.MethodPatch, "/x", nil)
	ctx := context.WithValue(r.Context(), UserContextKey, user)
	ctx = context.WithValue(ctx, WorkspaceContextKey, &WorkspaceMembership{
		WorkspaceID: "abcdef0123456789abcdef01",
		Role:        "member",
	})
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()
	if RequireWorkspaceRole(rr, r, "owner") {
		t.Errorf("member should be rejected for owner role")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("status=%d, want 403", rr.Code)
	}
}
