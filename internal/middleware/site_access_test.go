package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"

	dbpkg "github.com/brightinteraction/atomicsite/internal/db"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// openTestDB sets up a fresh SQLite + applies the embedded schema. Same
// pattern as analyticsdb_test / retention_test so this stays portable.
func openTestDB(t *testing.T) (*sql.DB, *store.Queries) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.sqlite")
	sqlDB, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(string(dbpkg.Schema)); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return sqlDB, store.New(sqlDB)
}

// seedUserAndSite returns (userID, siteID). The user is created with the
// given role; site_members is NOT auto-populated so each test can decide.
func seedUserAndSite(t *testing.T, sqlDB *sql.DB, role, userID, siteID string) {
	t.Helper()
	if _, err := sqlDB.Exec(
		`INSERT INTO users (id, email, password_hash, name, role) VALUES (?, ?, '', '', ?)`,
		userID, userID+"@example.com", role,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO sites (id, name, slug) VALUES (?, ?, ?)`,
		siteID, "Site "+siteID, siteID,
	); err != nil {
		t.Fatalf("seed site: %v", err)
	}
}

func grantMembership(t *testing.T, queries *store.Queries, siteID, userID, role string) {
	t.Helper()
	if err := queries.AddSiteMember(context.Background(), store.AddSiteMemberParams{
		SiteID: siteID,
		UserID: userID,
		Role:   role,
	}); err != nil {
		t.Fatalf("grant membership: %v", err)
	}
}

// runRouted exercises the middleware via a chi router so URLParam
// resolves correctly. Returns the recorded response.
func runRouted(t *testing.T, mw func(http.Handler) http.Handler, user *AuthUser, siteID string) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	router.With(mw).Get("/api/sites/{siteID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("granted"))
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID, nil)
	if user != nil {
		ctx := context.WithValue(req.Context(), UserContextKey, user)
		req = req.WithContext(ctx)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestSiteAccess_UnauthenticatedRejects(t *testing.T) {
	_, q := openTestDB(t)
	rr := runRouted(t, SiteAccessMiddleware(q), nil, "abcdef0123456789abcdef01")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestSiteAccess_AdminBypassesMembershipCheck(t *testing.T) {
	sqlDB, q := openTestDB(t)
	siteID := "abcdef0123456789abcdef01"
	seedUserAndSite(t, sqlDB, "admin", "admin1", siteID)
	// No membership row, but admin role bypasses.
	user := &AuthUser{ID: "admin1", Role: "admin"}
	rr := runRouted(t, SiteAccessMiddleware(q), user, siteID)
	if rr.Code != http.StatusOK {
		t.Errorf("admin should pass through; status = %d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestSiteAccess_NonMemberRejected(t *testing.T) {
	sqlDB, q := openTestDB(t)
	siteID := "abcdef0123456789abcdef01"
	seedUserAndSite(t, sqlDB, "editor", "alice", siteID)
	// Alice has no site_members row → 403.
	user := &AuthUser{ID: "alice", Role: "editor"}
	rr := runRouted(t, SiteAccessMiddleware(q), user, siteID)
	if rr.Code != http.StatusForbidden {
		t.Errorf("non-member should 403; got %d", rr.Code)
	}
}

func TestSiteAccess_MemberAccepted(t *testing.T) {
	sqlDB, q := openTestDB(t)
	siteID := "abcdef0123456789abcdef01"
	seedUserAndSite(t, sqlDB, "editor", "alice", siteID)
	grantMembership(t, q, siteID, "alice", "editor")
	user := &AuthUser{ID: "alice", Role: "editor"}
	rr := runRouted(t, SiteAccessMiddleware(q), user, siteID)
	if rr.Code != http.StatusOK {
		t.Errorf("member should pass; got %d body=%s", rr.Code, rr.Body.String())
	}
}

// The audit C1 regression guard: a site_members row for site A must NOT
// grant Alice access to site B.
func TestSiteAccess_CrossSiteIsolation(t *testing.T) {
	sqlDB, q := openTestDB(t)
	siteA := "aaaaaaaaaaaaaaaaaaaaaaaa"
	siteB := "bbbbbbbbbbbbbbbbbbbbbbbb"
	seedUserAndSite(t, sqlDB, "editor", "alice", siteA)
	if _, err := sqlDB.Exec(`INSERT INTO sites (id, name, slug) VALUES (?, 'B', 'siteb')`, siteB); err != nil {
		t.Fatalf("seed site B: %v", err)
	}
	grantMembership(t, q, siteA, "alice", "editor")
	user := &AuthUser{ID: "alice", Role: "editor"}

	// Alice → site A: 200
	if rr := runRouted(t, SiteAccessMiddleware(q), user, siteA); rr.Code != http.StatusOK {
		t.Errorf("alice on site A: got %d, want 200", rr.Code)
	}
	// Alice → site B: 403 (no membership row)
	if rr := runRouted(t, SiteAccessMiddleware(q), user, siteB); rr.Code != http.StatusForbidden {
		t.Errorf("alice on site B: got %d, want 403 (cross-site isolation)", rr.Code)
	}
}

func TestSiteAccess_MalformedSiteIDRejects(t *testing.T) {
	_, q := openTestDB(t)
	user := &AuthUser{ID: "alice", Role: "editor"}
	// chi will 404 on empty / slash-bearing strings before middleware
	// runs (no route match), which is fine. We only test cases that
	// chi DOES route through to the middleware: bare strings that
	// match the {siteID} placeholder shape but fail isSafeSiteID.
	for _, bad := range []string{"abc", "abcdef0123456789abcdef01zz", "ZZZZZZZZZZZZZZZZZZZZZZZZ"} {
		rr := runRouted(t, SiteAccessMiddleware(q), user, bad)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("siteID=%q: got %d, want 400", bad, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Invalid siteID") {
			t.Errorf("siteID=%q: body should mention Invalid siteID; got %s", bad, rr.Body.String())
		}
	}
}

func TestBackfillSiteMembersForAdmin_Idempotent(t *testing.T) {
	sqlDB, q := openTestDB(t)
	seedUserAndSite(t, sqlDB, "admin", "admin1", "abcdef0123456789abcdef01")
	if _, err := sqlDB.Exec(`INSERT INTO sites (id, name, slug) VALUES ('bbbb000000000000000000bb', 'B', 'b')`); err != nil {
		t.Fatalf("seed site B: %v", err)
	}
	ctx := context.Background()

	// First pass: should grant 2 memberships.
	if err := q.BackfillSiteMembersForAdmin(ctx, "admin1"); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	ids, _ := q.ListSiteIDsForUser(ctx, "admin1")
	if len(ids) != 2 {
		t.Errorf("after backfill: ListSiteIDsForUser = %v, want 2 sites", ids)
	}

	// Second pass: idempotent — count must not double.
	if err := q.BackfillSiteMembersForAdmin(ctx, "admin1"); err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	ids2, _ := q.ListSiteIDsForUser(ctx, "admin1")
	if len(ids2) != 2 {
		t.Errorf("second backfill must be idempotent; got %d rows, want 2", len(ids2))
	}
}
