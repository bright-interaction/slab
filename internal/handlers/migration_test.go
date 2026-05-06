package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"

	"github.com/brightinteraction/atomicsite/internal/config"
	dbpkg "github.com/brightinteraction/atomicsite/internal/db"
	"github.com/brightinteraction/atomicsite/internal/migration"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// fakeUploaderForHandlers mirrors fakeUploader in the migration package
// (kept local so we don't reach across packages for a test type).
type fakeUploaderForHandlers struct {
	uploads atomic.Int32
}

func (f *fakeUploaderForHandlers) UploadFromURL(ctx context.Context, siteID, sourceURL, alt, caption string) (string, string, error) {
	if sourceURL == "" {
		return "", "", errors.New("empty source URL")
	}
	n := f.uploads.Add(1)
	return fmt.Sprintf("media_%020d", n), fmt.Sprintf("https://atomicsite.local/m/%d.jpg", n), nil
}

func newMigrationHandlerForTest(t *testing.T) (*MigrationHandler, *store.Queries, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "migr.db")
	sqlDB, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(string(dbpkg.Schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	q := store.New(sqlDB)
	siteID := "abcdef0123456789abcdefm1"
	if err := q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: siteID, Name: "T", Slug: "t-migr", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en",
	}); err != nil {
		t.Fatalf("create site: %v", err)
	}
	// Sprint 3 (2026-05-06): Apply is plan-gated. Seed an OSS workspace
	// so the gate returns -1 (unlimited) and tests pass without hitting
	// caps. Cap-blocking is exercised in dedicated quota tests below.
	if err := q.CreateWorkspace(context.Background(), store.CreateWorkspaceParams{
		ID: "ws-migr-test", Name: "Test", Slug: "test-migr-ws",
		Plan: "oss", Region: "eu", BillingEmail: "ops@test.local",
		Status: "active", TrialEndsAt: "",
	}); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	return NewMigrationHandler(cfg, q, &fakeUploaderForHandlers{}), q, siteID
}

func withMigrationParams(siteID, migrationID string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("siteID", siteID)
		if migrationID != "" {
			rctx.URLParams.Add("migrationID", migrationID)
		}
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		// Sprint 3 (2026-05-06): Apply requires auth context for the
		// plan-quota gate. Inject admin so resolveUserWorkspace finds
		// the seeded workspace. Non-gated handlers ignore the extra
		// context, so this is a safe no-op for them.
		r = seedAuthCtx(r)
		h(w, r)
	}
}

func postJSONMigr(handler http.HandlerFunc, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// ----- Start: real sitemap crawl against a httptest server -----

func TestMigration_StartSitemapStoresManifest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		base := "http://" + r.Host
		_, _ = fmt.Fprintf(w, `<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<url><loc>%s/about</loc></url><url><loc>%s/contact</loc></url>
</urlset>`, base, base)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, `<html><head><title>%s</title></head><body><h1>%s</h1></body></html>`,
			r.URL.Path, r.URL.Path)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	h, q, siteID := newMigrationHandlerForTest(t)
	// We need allow_private fetches in the importer when targeting httptest.
	// The handler goes through migration.CrawlSitemap with default options
	// which has SafeFetch.AllowPrivate=false. To work around for this test
	// we POST a test-only flag... but the public Start API doesn't expose
	// AllowPrivate (rightly so). So bypass Start and seed the DB directly
	// with a manifest produced by an in-test crawl.
	manifest, err := migration.CrawlSitemap(context.Background(), srv.URL+"/sitemap.xml",
		migration.CrawlOptions{FetchOpts: migration.FetchOptions{AllowPrivate: true}})
	if err != nil {
		t.Fatalf("crawl: %v", err)
	}
	manifestJSON, _ := json.Marshal(manifest)
	migID := "test-mig-1"
	if err := q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: srv.URL + "/sitemap.xml",
		SourceType: "sitemap", ManifestJson: string(manifestJSON), Status: "ready",
	}); err != nil {
		t.Fatalf("seed migration: %v", err)
	}

	// Get returns the manifest.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	withMigrationParams(siteID, migID, h.Get)(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Get: want 200 got %d body=%s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["status"] != "ready" {
		t.Errorf("status: %v", got["status"])
	}
}

// ----- Plan: returns conflicts when collisions exist -----

func TestMigration_PlanReturnsConflictsAgainstExistingPages(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	// Seed a live page at slug "about".
	_ = q.CreatePage(context.Background(), store.CreatePageParams{
		ID: "p1", SiteID: siteID, Title: "About", Slug: "about", Layout: "default",
	})
	// Seed a manifest whose page collides at /about.
	manifest := migration.MigrationManifest{
		Pages: []migration.MigrationPage{{SourcePath: "/about", Title: "About"}},
	}
	mj, _ := json.Marshal(manifest)
	migID := "test-mig-conflict"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})

	rr := postJSONMigr(withMigrationParams(siteID, migID, h.Plan), map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("Plan: %d body=%s", rr.Code, rr.Body.String())
	}
	var plan migration.URLPlan
	_ = json.Unmarshal(rr.Body.Bytes(), &plan)
	if len(plan.Conflicts) != 1 {
		t.Fatalf("want 1 conflict, got %d (%+v)", len(plan.Conflicts), plan.Conflicts)
	}
}

// ----- Apply end-to-end: manifest -> pages + redirects in DB -----

func TestMigration_ApplyCommitsManifest(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	manifest := migration.MigrationManifest{
		Pages: []migration.MigrationPage{
			{SourcePath: "/about-us.html", Title: "About",
				MetaDescription: "About text"},
			{SourcePath: "/contact", Title: "Contact"},
		},
	}
	mj, _ := json.Marshal(manifest)
	migID := "test-mig-apply"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})

	rr := postJSONMigr(withMigrationParams(siteID, migID, h.Apply), map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("Apply: %d body=%s", rr.Code, rr.Body.String())
	}

	// 2 pages should land.
	pages, _ := q.ListPagesBySite(context.Background(), siteID)
	if len(pages) != 2 {
		t.Errorf("DB pages: %d", len(pages))
	}
	// 1 redirect (.html stripped).
	redirects, _ := q.ListRedirectsBySite(context.Background(), siteID)
	if len(redirects) != 1 {
		t.Errorf("DB redirects: %d", len(redirects))
	}
	if redirects[0].FromPath != "/about-us.html" || redirects[0].ToPath != "/about-us" {
		t.Errorf("redirect mapping: %+v", redirects[0])
	}
	// Migration row status should flip to applied.
	row, _ := q.GetMigrationByID(context.Background(), migID)
	if row.Status != "applied" {
		t.Errorf("status not flipped: %q", row.Status)
	}
}

// ----- Apply: refuses second apply -----

func TestMigration_ApplyTwiceConflicts(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	manifest := migration.MigrationManifest{
		Pages: []migration.MigrationPage{{SourcePath: "/x", Title: "X"}},
	}
	mj, _ := json.Marshal(manifest)
	migID := "test-mig-twice"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})

	first := postJSONMigr(withMigrationParams(siteID, migID, h.Apply), map[string]any{})
	if first.Code != http.StatusOK {
		t.Fatalf("first Apply: %d body=%s", first.Code, first.Body.String())
	}
	second := postJSONMigr(withMigrationParams(siteID, migID, h.Apply), map[string]any{})
	if second.Code != http.StatusConflict {
		t.Errorf("second Apply must 409, got %d", second.Code)
	}
}

// ----- Cross-tenant: site B can't see site A's migration -----

func TestMigration_CrossTenantBlocked(t *testing.T) {
	h, q, siteA := newMigrationHandlerForTest(t)
	siteB := "abcdef0123456789abcdefm2"
	_ = q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: siteB, Name: "B", Slug: "b-migr", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en",
	})
	migID := "test-mig-tenantA"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteA, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: "{}", Status: "ready",
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	withMigrationParams(siteB, migID, h.Get)(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("cross-tenant Get must 404, got %d", rr.Code)
	}
}

// ----- Delete -----

func TestMigration_DeleteRemovesRow(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	migID := "test-mig-del"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: "{}", Status: "ready",
	})
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rr := httptest.NewRecorder()
	withMigrationParams(siteID, migID, h.Delete)(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Delete: %d", rr.Code)
	}
	rows, _ := q.ListMigrationsBySite(context.Background(), siteID)
	if len(rows) != 0 {
		t.Errorf("delete left rows: %+v", rows)
	}
}

// ----- List shape -----

func TestMigration_ListReturnsSummaryNotFullManifest(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	manifest := migration.MigrationManifest{
		Pages: []migration.MigrationPage{
			{SourcePath: "/x", Title: "X", HTML: strings.Repeat("a", 10000)},
		},
		Stats: migration.MigrationStats{PagesFound: 1},
	}
	mj, _ := json.Marshal(manifest)
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: "id-list-1", SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	withMigrationParams(siteID, "", h.List)(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("List: %d", rr.Code)
	}
	// Body should NOT contain the 10kB HTML blob.
	if strings.Count(rr.Body.String(), "a") > 1000 {
		t.Errorf("List leaked full manifest body; expected only summary stats")
	}
}

// ----- Sprint 2: VerifyCoverage (pre-launch dry-run) -----

func TestMigration_VerifyCoverageBucketsCorrectly(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	// Seed: a live page at /existing and a manual redirect /old -> /new.
	_ = q.CreatePage(context.Background(), store.CreatePageParams{
		ID: "p1", SiteID: siteID, Title: "Existing", Slug: "existing", Layout: "default",
	})
	_ = q.CreateRedirect(context.Background(), store.CreateRedirectParams{
		ID: "r1", SiteID: siteID, FromPath: "/old", ToPath: "/new",
		StatusCode: 301, IsAuto: 0,
	})
	// Manifest contributes a NEW page /from-manifest plus a redirect
	// /legacy -> /modern (auto from a slug-strip).
	manifest := migration.MigrationManifest{
		Pages: []migration.MigrationPage{
			{SourcePath: "/from-manifest", Title: "From manifest"},
			{SourcePath: "/legacy.html", Title: "Legacy"},
		},
	}
	mj, _ := json.Marshal(manifest)
	migID := "test-mig-vc"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})

	rr := postJSONMigr(withMigrationParams(siteID, migID, h.VerifyCoverage), map[string]any{
		"urls": []string{
			"https://old/existing",      // live page -> 200
			"https://old/from-manifest", // planned page -> 200 (post-apply)
			"https://old/old",           // existing redirect -> 301
			"https://old/legacy.html",   // planned redirect -> 301 (post-apply)
			"https://old/orphan",        // unmapped -> 404
			"",                          // malformed
		},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("VerifyCoverage: %d body=%s", rr.Code, rr.Body.String())
	}
	var resp VerifyResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Will200) != 2 {
		t.Errorf("Will200: want 2, got %d (%v)", len(resp.Will200), resp.Will200)
	}
	if len(resp.Will301) != 2 {
		t.Errorf("Will301: want 2, got %d (%v)", len(resp.Will301), resp.Will301)
	}
	if len(resp.Will404) != 1 {
		t.Errorf("Will404: want 1, got %d (%v)", len(resp.Will404), resp.Will404)
	}
	if len(resp.Malformed) != 1 {
		t.Errorf("Malformed: want 1, got %d", len(resp.Malformed))
	}
}

func TestMigration_VerifyCoverageRejectsEmpty(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	migID := "test-mig-vc-empty"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: "{}", Status: "ready",
	})
	rr := postJSONMigr(withMigrationParams(siteID, migID, h.VerifyCoverage), map[string]any{
		"urls": []string{},
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("empty urls must 400, got %d", rr.Code)
	}
}

func TestMigration_VerifyCoverageCrossTenantBlocked(t *testing.T) {
	h, q, siteA := newMigrationHandlerForTest(t)
	siteB := "abcdef0123456789abcdefvc"
	_ = q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: siteB, Name: "B", Slug: "b-vc", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en",
	})
	migID := "test-mig-vc-tenA"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteA, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: "{}", Status: "ready",
	})
	rr := postJSONMigr(withMigrationParams(siteB, migID, h.VerifyCoverage), map[string]any{
		"urls": []string{"/x"},
	})
	if rr.Code != http.StatusNotFound {
		t.Errorf("cross-tenant must 404, got %d", rr.Code)
	}
}

// ----- Sprint 2: VerifyLive (post-launch crawl) -----

func TestMigration_VerifyLiveStoresResults(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	// Synthetic deployed atomicsite: /old 301 -> /new (200), /broken 404.
	mux := http.NewServeMux()
	mux.HandleFunc("/old", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/new", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/broken", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	migID := "test-mig-vl"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: "{}", Status: "applied",
	})

	// Test seam: allow loopback so httptest.Server is reachable. The
	// production code path (server.go) never sets AllowPrivate=true.
	h.SetVerifyOptionsForTest(migration.VerifyOptions{
		AllowPrivate: true, PolitenessDelay: 1 * time.Millisecond,
	})

	rr := postJSONMigr(withMigrationParams(siteID, migID, h.VerifyLive), map[string]any{
		"urls":            []string{"/old", "/new", "/broken"},
		"deployed_domain": srv.URL, // includes scheme
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("VerifyLive: %d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Checked int                       `json:"checked"`
		OK      int                       `json:"ok"`
		Failed  int                       `json:"failed"`
		Results []migration.VerifyResult  `json:"results"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Checked != 3 || resp.OK != 2 || resp.Failed != 1 {
		t.Errorf("counts: checked=%d ok=%d failed=%d", resp.Checked, resp.OK, resp.Failed)
	}

	// Persisted rows: 3 verifications stored on the migration.
	stored, _ := q.ListMigrationVerifications(context.Background(), migID)
	if len(stored) != 3 {
		t.Errorf("stored verifications: want 3, got %d", len(stored))
	}
	counts, _ := q.CountMigrationVerificationsByOK(context.Background(), migID)
	if counts.OkCount != 2 || counts.FailCount != 1 {
		t.Errorf("count query: ok=%d fail=%d", counts.OkCount, counts.FailCount)
	}
}

func TestMigration_VerifyLiveRequiresDeployedDomain(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	migID := "test-mig-vl-nodomain"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: "{}", Status: "applied",
	})
	rr := postJSONMigr(withMigrationParams(siteID, migID, h.VerifyLive), map[string]any{
		"urls": []string{"/x"},
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("missing deployed_domain must 400, got %d", rr.Code)
	}
}

func TestMigration_VerifyLiveCapsTo1000URLs(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	migID := "test-mig-vl-cap"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: "{}", Status: "applied",
	})
	urls := make([]string, 1001)
	for i := range urls {
		urls[i] = "/p" + strings.Repeat("0", 6)
	}
	rr := postJSONMigr(withMigrationParams(siteID, migID, h.VerifyLive), map[string]any{
		"urls":            urls,
		"deployed_domain": "deploy.example",
	})
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("over-cap must 413, got %d", rr.Code)
	}
}

// ----- Sprint 2: ListVerifications -----

func TestMigration_ListVerificationsReturnsStoredRows(t *testing.T) {
	h, q, siteID := newMigrationHandlerForTest(t)
	migID := "test-mig-listv"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: "{}", Status: "applied",
	})
	// Seed two stored verifications directly.
	_ = q.CreateMigrationVerification(context.Background(), store.CreateMigrationVerificationParams{
		ID: "v1", SiteID: siteID, MigrationID: migID,
		SourceUrl: "https://old/a", StatusCode: 200, FinalUrl: "https://new/a",
		Hops: 0, Ok: 1, Error: "",
	})
	_ = q.CreateMigrationVerification(context.Background(), store.CreateMigrationVerificationParams{
		ID: "v2", SiteID: siteID, MigrationID: migID,
		SourceUrl: "https://old/b", StatusCode: 404, FinalUrl: "https://new/b",
		Hops: 1, Ok: 0, Error: "terminal status 404",
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	withMigrationParams(siteID, migID, h.ListVerifications)(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("ListVerifications: %d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items  []map[string]any `json:"items"`
		OK     int64            `json:"ok"`
		Failed int64            `json:"failed"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Items) != 2 {
		t.Errorf("items: %d", len(resp.Items))
	}
	if resp.OK != 1 || resp.Failed != 1 {
		t.Errorf("counts: ok=%d fail=%d", resp.OK, resp.Failed)
	}
}

// ----- Sprint 3: workspace plan-quota gate on Apply -----

// newPaidPlanMigrationStack returns a migration handler whose seeded
// workspace is on a specific plan. Used by the quota tests.
func newPaidPlanMigrationStack(t *testing.T, plan string) (*MigrationHandler, *store.Queries, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "migr-quota.db")
	sqlDB, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(string(dbpkg.Schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	q := store.New(sqlDB)
	siteID := "abcdef0123456789abcdefqq"
	_ = q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: siteID, Name: "T", Slug: "t-migr-q", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en",
	})
	_ = q.CreateWorkspace(context.Background(), store.CreateWorkspaceParams{
		ID: "ws-quota", Name: "Quota Test", Slug: "quota-test-ws",
		Plan: plan, Region: "eu", BillingEmail: "ops@test.local",
		Status: "active", TrialEndsAt: "",
	})
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	return NewMigrationHandler(cfg, q, &fakeUploaderForHandlers{}), q, siteID
}

// makeManifestWithNPages helper - generates a manifest the URL planner
// turns into N new pages (no skips, no collisions).
func makeManifestWithNPages(n int) migration.MigrationManifest {
	pages := make([]migration.MigrationPage, n)
	for i := 0; i < n; i++ {
		pages[i] = migration.MigrationPage{
			SourcePath: fmt.Sprintf("/quota-page-%d", i),
			Title:      fmt.Sprintf("Page %d", i),
		}
	}
	return migration.MigrationManifest{Pages: pages}
}

// TestMigration_ApplyBlockedBySoloPlanCap verifies that a 51-page
// manifest applied against a Solo workspace (50-page cap) returns 402
// without inserting any pages and leaves the migration row at status=ready.
func TestMigration_ApplyBlockedBySoloPlanCap(t *testing.T) {
	h, q, siteID := newPaidPlanMigrationStack(t, "solo")
	manifest := makeManifestWithNPages(51) // exceeds solo cap of 50
	mj, _ := json.Marshal(manifest)
	migID := "test-mig-quota-block"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})

	rr := postJSONMigr(withMigrationParams(siteID, migID, h.Apply), map[string]any{})
	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("want 402, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "max_pages_per_site") &&
		!strings.Contains(rr.Body.String(), "plan cap") {
		t.Errorf("error should mention plan cap or dimension: %s", rr.Body.String())
	}

	// Migration row must stay at "ready" so the operator can adjust skips
	// and retry without re-crawling.
	row, _ := q.GetMigrationByID(context.Background(), migID)
	if row.Status != "ready" {
		t.Errorf("status flipped on quota block (must stay ready), got %q", row.Status)
	}
	// No pages should have been inserted.
	pages, _ := q.ListPagesBySite(context.Background(), siteID)
	if len(pages) != 0 {
		t.Errorf("quota-blocked apply must insert zero pages, got %d", len(pages))
	}
}

// TestMigration_ApplyAllowedAtSoloCap verifies the boundary condition:
// exactly 50 pages on a Solo plan succeeds.
func TestMigration_ApplyAllowedAtSoloCap(t *testing.T) {
	h, q, siteID := newPaidPlanMigrationStack(t, "solo")
	manifest := makeManifestWithNPages(50) // exactly at cap
	mj, _ := json.Marshal(manifest)
	migID := "test-mig-quota-edge"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})
	rr := postJSONMigr(withMigrationParams(siteID, migID, h.Apply), map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("at-cap apply must succeed, got %d body=%s", rr.Code, rr.Body.String())
	}
	pages, _ := q.ListPagesBySite(context.Background(), siteID)
	if len(pages) != 50 {
		t.Errorf("want 50 pages, got %d", len(pages))
	}
}

// TestMigration_ApplyAllowedOnOSSPlan verifies that the OSS plan (-1
// unlimited) lets a large import through without complaint.
func TestMigration_ApplyAllowedOnOSSPlan(t *testing.T) {
	h, q, siteID := newPaidPlanMigrationStack(t, "oss")
	manifest := makeManifestWithNPages(120) // far above any paid cap
	mj, _ := json.Marshal(manifest)
	migID := "test-mig-oss-unlimited"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})
	rr := postJSONMigr(withMigrationParams(siteID, migID, h.Apply), map[string]any{})
	if rr.Code != http.StatusOK {
		t.Errorf("OSS plan must allow unlimited; got %d body=%s", rr.Code, rr.Body.String())
	}
}

// TestMigration_ApplyAccountsForExistingPages confirms that the gate adds
// current site pages to the projected count - if the site already has
// 49 pages on a Solo plan and the manifest adds 2 more, the second one
// pushes past 50 and the apply must be refused.
func TestMigration_ApplyAccountsForExistingPages(t *testing.T) {
	h, q, siteID := newPaidPlanMigrationStack(t, "solo")
	// Pre-seed 49 pages directly.
	for i := 0; i < 49; i++ {
		_ = q.CreatePage(context.Background(), store.CreatePageParams{
			ID: fmt.Sprintf("pre-%03d", i), SiteID: siteID,
			Title: fmt.Sprintf("Pre %d", i), Slug: fmt.Sprintf("pre-%d", i),
			Layout: "default",
		})
	}
	manifest := makeManifestWithNPages(2) // 49 + 2 = 51 > 50
	mj, _ := json.Marshal(manifest)
	migID := "test-mig-quota-existing"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})
	rr := postJSONMigr(withMigrationParams(siteID, migID, h.Apply), map[string]any{})
	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("want 402 (existing+projected over cap), got %d body=%s", rr.Code, rr.Body.String())
	}
}

// TestMigration_ApplyRequiresAuth proves the gate refuses anonymous
// callers (matches the existing SiteHandler.Seed contract).
func TestMigration_ApplyRequiresAuth(t *testing.T) {
	h, q, siteID := newPaidPlanMigrationStack(t, "solo")
	manifest := makeManifestWithNPages(1)
	mj, _ := json.Marshal(manifest)
	migID := "test-mig-anon"
	_ = q.CreateMigration(context.Background(), store.CreateMigrationParams{
		ID: migID, SiteID: siteID, SourceUrl: "x", SourceType: "sitemap",
		ManifestJson: string(mj), Status: "ready",
	})
	// Bypass the seedAuthCtx-injecting helper to send a truly unauth'd
	// request straight at h.Apply.
	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("siteID", siteID)
	rctx.URLParams.Add("migrationID", migID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.Apply(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("anonymous Apply must 401, got %d", rr.Code)
	}
}

// ----- NoopMediaUploader returns error so porter records warning -----

func TestNoopMediaUploader_ErrorsCleanly(t *testing.T) {
	u := NoopMediaUploader()
	_, _, err := u.UploadFromURL(context.Background(), "site", "https://x/y.jpg", "", "")
	if err == nil {
		t.Errorf("noop uploader must error so porter records warning")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error message should be clear: %v", err)
	}
}
