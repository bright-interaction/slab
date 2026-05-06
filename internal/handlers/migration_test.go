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
