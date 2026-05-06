package migration

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	dbpkg "github.com/brightinteraction/atomicsite/internal/db"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// fakeUploader returns deterministic atomicsite URLs without hitting disk.
// Tests can flip failNext to make the next upload return an error so we can
// exercise the porter's media-warning path.
type fakeUploader struct {
	uploads  atomic.Int32
	failNext atomic.Bool
	failURL  string // when set, fail only this exact source URL
}

func (f *fakeUploader) UploadFromURL(ctx context.Context, siteID, sourceURL, alt, caption string) (string, string, error) {
	if f.failURL != "" && sourceURL == f.failURL {
		return "", "", errors.New("simulated upload failure")
	}
	if f.failNext.CompareAndSwap(true, false) {
		return "", "", errors.New("simulated upload failure")
	}
	n := f.uploads.Add(1)
	id := "media_test_" + strings.Repeat("0", 23-len(itoa(int(n)))) + itoa(int(n))
	url := "https://atomicsite.local/media/" + id + ".jpg"
	return id[:24], url, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}

func openPorterTestDB(t *testing.T) (*sql.DB, *store.Queries, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "porter.db")
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
	siteID := "abcdef0123456789abcdefp1"
	if err := q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: siteID, Name: "T", Slug: "t-port", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en",
	}); err != nil {
		t.Fatalf("create site: %v", err)
	}
	return sqlDB, q, siteID
}

// ----- Happy path: pages + collections + items + redirects + media -----

func TestPorter_AppliesPagesCollectionsRedirects(t *testing.T) {
	_, q, siteID := openPorterTestDB(t)
	manifest := &MigrationManifest{
		SourceDomain: "old-site.example",
		Pages: []MigrationPage{
			{SourcePath: "/about-us.html", Title: "About",
				MetaDescription: "About text", OGImageURL: "https://old/og.jpg",
				CanonicalURL: "https://old-site.example/about-us"},
			{SourcePath: "/contact", Title: "Contact"},
		},
		Collections: []MigrationCollection{
			{
				Slug: "posts", Name: "Posts",
				Schema: []MigrationCollectionField{
					{Name: "title", Type: "text", Required: true},
					{Name: "content", Type: "richtext"},
				},
				Items: []MigrationCollectionItem{
					{Slug: "first-post", Title: "First Post",
						PublishedAt: time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
						Data: map[string]any{
							"title":   "First Post",
							"content": `<p>Body with <img src="https://old/img.jpg"> and <a href="/about-us">link</a></p>`,
						},
					},
				},
			},
		},
		Media: []MigrationMediaRef{
			{SourceURL: "https://old/og.jpg", Alt: "OG"},
			{SourceURL: "https://old/img.jpg", Alt: "Inline"},
		},
	}

	plan := PlanURLs(manifest, nil, DefaultPlanOptions())
	if len(plan.Conflicts) != 0 {
		t.Fatalf("plan conflicts: %+v", plan.Conflicts)
	}

	uploader := &fakeUploader{}
	porter := NewPorter(q, uploader)
	res, err := porter.Apply(context.Background(), siteID, manifest, plan)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if res.PagesCreated != 2 {
		t.Errorf("PagesCreated = %d", res.PagesCreated)
	}
	if res.CollectionsCreated != 1 {
		t.Errorf("CollectionsCreated = %d", res.CollectionsCreated)
	}
	if res.ItemsCreated != 1 {
		t.Errorf("ItemsCreated = %d", res.ItemsCreated)
	}
	if res.MediaUploaded != 2 {
		t.Errorf("MediaUploaded = %d", res.MediaUploaded)
	}
	// One slug changed (/about-us.html -> /about-us). Only one redirect expected.
	if res.RedirectsCreated != 1 {
		t.Errorf("RedirectsCreated = %d (want 1)", res.RedirectsCreated)
	}

	// Verify pages landed.
	pages, _ := q.ListPagesBySite(context.Background(), siteID)
	if len(pages) != 2 {
		t.Errorf("DB pages: want 2, got %d", len(pages))
	}
	hasAbout := false
	for _, p := range pages {
		if p.Slug == "about-us" {
			hasAbout = true
			if p.MetaDescription != "About text" {
				t.Errorf("meta_description not stored: %q", p.MetaDescription)
			}
			if p.CanonicalUrl == "" {
				t.Errorf("canonical_url not stored")
			}
		}
	}
	if !hasAbout {
		t.Errorf("about page missing: %+v", pages)
	}

	// Verify collection + item.
	colls, _ := q.ListCollectionsBySite(context.Background(), siteID)
	if len(colls) != 1 {
		t.Errorf("DB collections: %d", len(colls))
	}
	items, _ := q.ListItemsByCollection(context.Background(), colls[0].ID)
	if len(items) != 1 {
		t.Fatalf("items: %d", len(items))
	}

	// Verify media URL was rewritten in item content.
	if !strings.Contains(items[0].DataJson, "atomicsite.local/media/") {
		t.Errorf("media URL not rewritten in item.data_json: %s", items[0].DataJson)
	}
	if strings.Contains(items[0].DataJson, "https://old/img.jpg") {
		t.Errorf("source media URL leaked in item.data_json")
	}
	// Verify internal link was rewritten /about-us -> /about-us (no change here)
	// But the .html stripping means /about-us.html -> /about-us
	// Actually /about-us was an HREF in content - planner mapped /about-us.html
	// to /about-us. The HREF was already /about-us so no change. Just confirm no
	// regression.

	// Verify redirect.
	redirects, _ := q.ListRedirectsBySite(context.Background(), siteID)
	if len(redirects) != 1 {
		t.Fatalf("redirects: want 1, got %d (%+v)", len(redirects), redirects)
	}
	if redirects[0].FromPath != "/about-us.html" || redirects[0].ToPath != "/about-us" {
		t.Errorf("redirect mapping wrong: %+v", redirects[0])
	}
	if redirects[0].StatusCode != 301 || redirects[0].IsAuto != 1 {
		t.Errorf("redirect metadata wrong: %+v", redirects[0])
	}
}

// ----- Conflict guard: planner conflicts must block Apply -----

func TestPorter_RefusesPlanWithConflicts(t *testing.T) {
	_, q, siteID := openPorterTestDB(t)
	porter := NewPorter(q, &fakeUploader{})
	plan := URLPlan{
		Conflicts: []PlanConflict{{SourcePath: "/x", ProposedSlug: "/x", ExistingKind: "page"}},
	}
	_, err := porter.Apply(context.Background(), siteID, &MigrationManifest{}, plan)
	if err == nil {
		t.Errorf("Apply should refuse a plan with conflicts")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Errorf("error should mention conflicts: %v", err)
	}
}

// ----- Page / redirect collision: page wins, redirect skipped with warning -----

func TestPorter_PageWinsOverRedirectAtSamePath(t *testing.T) {
	_, q, siteID := openPorterTestDB(t)
	// Single page at /about. We inject a redirect whose from_path also
	// resolves to /about - that redirect would be dead because the
	// page is being created there. Porter must skip it with a warning
	// instead of letting the unique(site_id, from_path) constraint blow up.
	manifest := &MigrationManifest{
		Pages: []MigrationPage{{SourcePath: "/about", Title: "About"}},
	}
	plan := PlanURLs(manifest, nil, DefaultPlanOptions())
	plan.Redirects = append(plan.Redirects, RedirectPlan{
		FromPath: "/about", ToPath: "/somewhere-else", StatusCode: 301,
	})
	porter := NewPorter(q, &fakeUploader{})
	res, err := porter.Apply(context.Background(), siteID, manifest, plan)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	skipped := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "/about -> /somewhere-else") && strings.Contains(w, "skipped") {
			skipped = true
		}
	}
	if !skipped {
		t.Errorf("page-vs-redirect collision warning missing: %+v", res.Warnings)
	}
	// Confirm the dead redirect did NOT make it to the DB.
	rows, _ := q.ListRedirectsBySite(context.Background(), siteID)
	for _, r := range rows {
		if r.FromPath == "/about" {
			t.Errorf("dead redirect should not be inserted: %+v", r)
		}
	}
}

// ----- Media failure becomes warning, not crash -----

func TestPorter_MediaFailureIsNonFatal(t *testing.T) {
	_, q, siteID := openPorterTestDB(t)
	manifest := &MigrationManifest{
		Pages: []MigrationPage{{SourcePath: "/x", Title: "X", OGImageURL: "https://old/broken.jpg"}},
		Media: []MigrationMediaRef{{SourceURL: "https://old/broken.jpg"}},
	}
	plan := PlanURLs(manifest, nil, DefaultPlanOptions())
	uploader := &fakeUploader{failURL: "https://old/broken.jpg"}
	porter := NewPorter(q, uploader)
	res, err := porter.Apply(context.Background(), siteID, manifest, plan)
	if err != nil {
		t.Fatalf("Apply must not fail on a single media error: %v", err)
	}
	if res.MediaFailed != 1 {
		t.Errorf("MediaFailed: %d", res.MediaFailed)
	}
	if len(res.Warnings) == 0 {
		t.Errorf("media failure should produce a warning")
	}
	if res.PagesCreated != 1 {
		t.Errorf("page should still be created when media fails: %d", res.PagesCreated)
	}
}

// ----- Rollback: any item-create failure unwinds the entire batch -----

type failingQueriesShim struct {
	*store.Queries
}

func TestPorter_RollbackOnItemFailure(t *testing.T) {
	_, q, siteID := openPorterTestDB(t)
	// Create a duplicate-slug scenario inside a single collection: two
	// items with the same (collection_id, locale, slug). The second
	// CreateItem fails and the porter must roll back the collection
	// + the first item.
	manifest := &MigrationManifest{
		Collections: []MigrationCollection{{
			Slug: "posts", Name: "Posts",
			Items: []MigrationCollectionItem{
				{Slug: "dup", Title: "First", Data: map[string]any{}},
				{Slug: "dup", Title: "Second", Data: map[string]any{}}, // collides
			},
		}},
	}
	plan := URLPlan{}
	porter := NewPorter(q, &fakeUploader{})
	_, err := porter.Apply(context.Background(), siteID, manifest, plan)
	if err == nil {
		t.Fatalf("duplicate-slug item should fail Apply")
	}
	// Confirm rollback: zero collections and zero items in the DB.
	colls, _ := q.ListCollectionsBySite(context.Background(), siteID)
	if len(colls) != 0 {
		t.Errorf("collections not rolled back: %d remain", len(colls))
	}
}

// ----- HTML media URL rewriting -----

func TestRewriteMediaURL_RewritesSrcAndHref(t *testing.T) {
	in := `<img src="https://old/img.jpg"><a href="https://old/img.jpg">link</a>`
	mediaMap := map[string]string{"https://old/img.jpg": "https://new/img.jpg"}
	got := rewriteMediaURL(in, mediaMap)
	if strings.Count(got, "https://new/img.jpg") != 2 {
		t.Errorf("did not rewrite both src and href: %q", got)
	}
	if strings.Contains(got, "https://old/img.jpg") {
		t.Errorf("source URL leaked: %q", got)
	}
}

func TestRewriteMediaURL_HandlesSingleQuotes(t *testing.T) {
	in := `<img src='https://old/img.jpg'>`
	got := rewriteMediaURL(in, map[string]string{"https://old/img.jpg": "https://new/img.jpg"})
	if !strings.Contains(got, "https://new/img.jpg") {
		t.Errorf("single-quote rewrite missing: %q", got)
	}
}

func TestRewriteLinksInItemData_RewritesHrefOnly(t *testing.T) {
	plan := URLPlan{Pages: []PagePlan{
		{SourcePath: "/about-us.html", NewSlug: "/about-us"},
	}}
	data := map[string]any{
		"content": `<a href="/about-us.html">read</a>`,
	}
	got := rewriteLinksInItemData(data, plan)
	c, _ := got["content"].(string)
	if !strings.Contains(c, `href="/about-us"`) {
		t.Errorf("link not rewritten: %q", c)
	}
}

// ----- truncateMeta -----

func TestTruncateMeta(t *testing.T) {
	if got := truncateMeta("short", 100); got != "short" {
		t.Errorf("short string altered: %q", got)
	}
	long := strings.Repeat("x", 250)
	if got := truncateMeta(long, 200); len(got) != 200 {
		t.Errorf("truncate length wrong: %d", len(got))
	}
}
