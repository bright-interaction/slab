package builder

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	dbpkg "github.com/brightinteraction/atomicsite/internal/db"
	"github.com/brightinteraction/atomicsite/internal/store"
)

func newBuilderTestDB(t *testing.T) (*sql.DB, *store.Queries, string, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "build.db")
	sqlDB, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(string(dbpkg.Schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	q := store.New(sqlDB)
	siteID := "abcdef0123456789abcdef04"
	if err := q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: siteID, Name: "T", Slug: "t-build", Domain: "example.com", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en",
	}); err != nil {
		t.Fatal(err)
	}
	wsDir := filepath.Join(dir, "ws")
	if err := os.MkdirAll(filepath.Join(wsDir, "src", "pages"), 0o755); err != nil {
		t.Fatal(err)
	}
	return sqlDB, q, siteID, wsDir
}

func seedCollectionWithItem(t *testing.T, q *store.Queries, siteID, slug, schemaOrgType, locale string) (string, string) {
	t.Helper()
	colID := "c_" + slug
	settings, _ := json.Marshal(map[string]any{"render_as_pages": true, "schema_org_type": schemaOrgType})
	schema, _ := json.Marshal([]map[string]any{
		{"name": "headline", "type": "text", "required": true},
		{"name": "description", "type": "textarea"},
	})
	if err := q.CreateCollection(context.Background(), store.CreateCollectionParams{
		ID: colID, SiteID: siteID, Name: slug, Slug: slug,
		SchemaJson: string(schema), SettingsJson: string(settings),
	}); err != nil {
		t.Fatal(err)
	}
	itemID := "i_" + slug + "_" + locale
	itemData, _ := json.Marshal(map[string]any{
		"headline":    "Test headline",
		"description": "Test description body.",
	})
	if err := q.CreateItem(context.Background(), store.CreateItemParams{
		ID: itemID, CollectionID: colID, SiteID: siteID,
		Slug: "first-item", Title: "First Item",
		DataJson: string(itemData), Locale: locale, Status: "published",
		PublishedAt: "2026-05-04 12:00:00",
	}); err != nil {
		t.Fatal(err)
	}
	return colID, itemID
}

func TestRenderCollectionPages_EmitsIndexAndDetail(t *testing.T) {
	_, q, siteID, wsDir := newBuilderTestDB(t)
	seedCollectionWithItem(t, q, siteID, "case-studies", "Article", "en")

	count, err := RenderCollectionPages(context.Background(), q, siteID, wsDir)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if count < 2 {
		t.Errorf("expected >=2 pages emitted (index + 1 item), got %d", count)
	}
	// Index page
	indexPath := filepath.Join(wsDir, "src", "pages", "case-studies", "index.astro")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	if !strings.Contains(string(indexBytes), "case-studies") {
		t.Errorf("index page missing collection slug: %s", indexBytes)
	}
	// Detail page
	detailPath := filepath.Join(wsDir, "src", "pages", "case-studies", "first-item.astro")
	detailBytes, err := os.ReadFile(detailPath)
	if err != nil {
		t.Fatalf("read detail: %v", err)
	}
	detail := string(detailBytes)
	if !strings.Contains(detail, "Test headline") {
		t.Errorf("detail page missing item title in body: %s", detail)
	}
	if !strings.Contains(detail, `application/ld+json`) {
		t.Errorf("detail page missing JSON-LD script tag")
	}
	if !strings.Contains(detail, `"@type":"Article"`) {
		t.Errorf("JSON-LD missing @type=Article: %s", detail)
	}
}

func TestRenderCollectionPages_HreflangAcrossLocales(t *testing.T) {
	_, q, siteID, wsDir := newBuilderTestDB(t)
	colID, _ := seedCollectionWithItem(t, q, siteID, "people", "Person", "en")
	// Add Swedish variant under same item slug.
	itemSv := "i_people_sv"
	itemData, _ := json.Marshal(map[string]any{
		"headline": "Persontest",
	})
	if err := q.CreateItem(context.Background(), store.CreateItemParams{
		ID: itemSv, CollectionID: colID, SiteID: siteID,
		Slug: "first-item", Title: "Första",
		DataJson: string(itemData), Locale: "sv", Status: "published",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := RenderCollectionPages(context.Background(), q, siteID, wsDir); err != nil {
		t.Fatalf("render: %v", err)
	}
	enBytes, err := os.ReadFile(filepath.Join(wsDir, "src", "pages", "people", "first-item.astro"))
	if err != nil {
		t.Fatal(err)
	}
	en := string(enBytes)
	if !strings.Contains(en, "alternates={") {
		t.Errorf("English variant missing alternates prop: %s", en)
	}
	svPath := filepath.Join(wsDir, "src", "pages", "sv", "people", "first-item.astro")
	if _, err := os.ReadFile(svPath); err != nil {
		t.Errorf("expected Swedish variant at %s: %v", svPath, err)
	}
}

func TestRenderCollectionPages_DraftSkipped(t *testing.T) {
	_, q, siteID, wsDir := newBuilderTestDB(t)
	colID, _ := seedCollectionWithItem(t, q, siteID, "news", "Article", "en")
	// Add a draft item.
	if err := q.CreateItem(context.Background(), store.CreateItemParams{
		ID: "i_draft", CollectionID: colID, SiteID: siteID,
		Slug: "draft-piece", Title: "Draft",
		DataJson: `{"headline":"draft"}`, Locale: "en", Status: "draft",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := RenderCollectionPages(context.Background(), q, siteID, wsDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(wsDir, "src", "pages", "news", "draft-piece.astro")); !os.IsNotExist(err) {
		t.Errorf("draft item should not have rendered, but file exists")
	}
	// Published item still rendered.
	if _, err := os.Stat(filepath.Join(wsDir, "src", "pages", "news", "first-item.astro")); err != nil {
		t.Errorf("published item missing: %v", err)
	}
}

func TestRenderCollectionPages_RenderAsPagesFalseSkipsAll(t *testing.T) {
	_, q, siteID, wsDir := newBuilderTestDB(t)
	// settings.render_as_pages = false -> nothing emitted
	colID := "c_data"
	settings, _ := json.Marshal(map[string]any{"render_as_pages": false})
	schema, _ := json.Marshal([]map[string]any{{"name": "headline", "type": "text"}})
	if err := q.CreateCollection(context.Background(), store.CreateCollectionParams{
		ID: colID, SiteID: siteID, Name: "data", Slug: "data",
		SchemaJson: string(schema), SettingsJson: string(settings),
	}); err != nil {
		t.Fatal(err)
	}
	if err := q.CreateItem(context.Background(), store.CreateItemParams{
		ID: "i_data", CollectionID: colID, SiteID: siteID,
		Slug: "x", Title: "X", DataJson: `{"headline":"x"}`, Status: "published",
	}); err != nil {
		t.Fatal(err)
	}
	count, err := RenderCollectionPages(context.Background(), q, siteID, wsDir)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 pages emitted when render_as_pages=false, got %d", count)
	}
}

func TestRenderCollectionListBlock_RendersResolvedItems(t *testing.T) {
	data := map[string]any{
		"_collection_slug": "case-studies",
		"_resolved_items": []any{
			map[string]any{
				"id":    "i1",
				"slug":  "acme",
				"title": "Acme",
				"href":  "/case-studies/acme/",
				"data":  map[string]any{},
			},
			map[string]any{
				"id":    "i2",
				"slug":  "bcorp",
				"title": "BCorp",
				"href":  "/case-studies/bcorp/",
				"data":  map[string]any{},
			},
		},
	}
	out := renderCollectionListBlock(data, nil)
	if !strings.Contains(out, "Acme") || !strings.Contains(out, "BCorp") {
		t.Errorf("missing item titles: %s", out)
	}
	if !strings.Contains(out, `data-collection="case-studies"`) {
		t.Errorf("missing data-collection attribute: %s", out)
	}
}

func TestSortCollectionItems_ByTitleDesc(t *testing.T) {
	items := []store.CollectionItem{
		{ID: "1", Title: "B"},
		{ID: "2", Title: "A"},
		{ID: "3", Title: "C"},
	}
	sorted := sortCollectionItems(items, "title", "desc")
	if sorted[0].Title != "C" || sorted[2].Title != "A" {
		t.Errorf("desc sort wrong: %+v", sorted)
	}
}

func TestMatchesStaticDSL(t *testing.T) {
	data := map[string]any{"industry": "finance", "featured": true}
	if !matchesStaticDSL(`industry == "finance"`, data) {
		t.Error("expected match on industry == finance")
	}
	if matchesStaticDSL(`industry == "tech"`, data) {
		t.Error("expected no match on industry == tech")
	}
	if !matchesStaticDSL(`featured present`, data) {
		t.Error("expected featured present")
	}
	if !matchesStaticDSL(``, data) {
		t.Error("empty filter should pass-all")
	}
}
