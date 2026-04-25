package starterkits

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	dbpkg "github.com/bright-interaction/slab/internal/db"
	"github.com/bright-interaction/slab/internal/store"
)

func setupKitTestDB(t *testing.T) (*sql.DB, *store.Queries) {
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

func insertSite(t *testing.T, sqlDB *sql.DB, siteID string) {
	t.Helper()
	_, err := sqlDB.Exec(`INSERT INTO sites (id, name, slug, status) VALUES (?, ?, ?, 'draft')`, siteID, "Test "+siteID, siteID)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}
}

func count(t *testing.T, sqlDB *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := sqlDB.QueryRowContext(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

func TestDefaultRegistry_HasAllKits(t *testing.T) {
	for _, id := range []string{"law-firm", "saas-landing", "portfolio"} {
		k, ok := Default.Get(id)
		if !ok {
			t.Fatalf("expected kit %q to be registered via init()", id)
			continue
		}
		if k.Name() == "" {
			t.Errorf("kit %q has empty Name()", id)
		}
		if k.Description() == "" {
			t.Errorf("kit %q has empty Description()", id)
		}
		if len(k.TargetSiteTypes()) == 0 {
			t.Errorf("kit %q has empty TargetSiteTypes()", id)
		}
	}
}

func TestLawFirmKit_ApplySeedsExpectedRows(t *testing.T) {
	sqlDB, q := setupKitTestDB(t)
	siteID := "site-lawfirm"
	insertSite(t, sqlDB, siteID)

	kit, ok := Default.Get("law-firm")
	if !ok {
		t.Fatalf("law-firm not registered")
	}
	if err := kit.Apply(context.Background(), q, siteID); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	wantComponents := []string{"hero-formal", "services-grid", "contact-callout", "team-grid"}
	for _, name := range wantComponents {
		if got := count(t, sqlDB, "SELECT COUNT(*) FROM components WHERE site_id=? AND name=?", siteID, name); got != 1 {
			t.Errorf("component %q: want 1, got %d", name, got)
		}
	}
	wantCSS := []string{".section-formal", ".hero-formal", ".callout-svar", ".team-card"}
	for _, name := range wantCSS {
		if got := count(t, sqlDB, "SELECT COUNT(*) FROM css_classes WHERE site_id=? AND name=?", siteID, name); got != 1 {
			t.Errorf("css class %q: want 1, got %d", name, got)
		}
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM knowledgebase_entries WHERE site_id=? AND category='brand'", siteID); got < 3 {
		t.Errorf("brand kb entries: want >=3, got %d", got)
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM knowledgebase_entries WHERE site_id=? AND category='technical'", siteID); got < 2 {
		t.Errorf("technical kb entries: want >=2, got %d", got)
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM guardrail_rules WHERE site_id=? AND rule_type='allow_block_type'", siteID); got != 1 {
		t.Errorf("allow_block_type guardrail: want 1, got %d", got)
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM guardrail_rules WHERE site_id=? AND rule_type='forbid_pattern'", siteID); got != 1 {
		t.Errorf("forbid_pattern guardrail: want 1, got %d", got)
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM guardrail_rules WHERE site_id=? AND rule_type='require_block'", siteID); got != 1 {
		t.Errorf("require_block guardrail: want 1, got %d", got)
	}
	wantPages := []string{"/", "/tjanster", "/tjanster/gdpr-radgivning", "/tjanster/avtal"}
	for _, slug := range wantPages {
		if got := count(t, sqlDB, "SELECT COUNT(*) FROM pages WHERE site_id=? AND slug=?", siteID, slug); got != 1 {
			t.Errorf("page %q: want 1, got %d", slug, got)
		}
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM blocks b JOIN pages p ON b.page_id = p.id WHERE p.site_id=? AND p.slug='/'", siteID); got < 4 {
		t.Errorf("home blocks: want >=4, got %d", got)
	}
}

func TestSaasLandingKit_ApplySeedsExpectedRows(t *testing.T) {
	sqlDB, q := setupKitTestDB(t)
	siteID := "site-saas"
	insertSite(t, sqlDB, siteID)

	kit, ok := Default.Get("saas-landing")
	if !ok {
		t.Fatalf("saas-landing not registered")
	}
	if err := kit.Apply(context.Background(), q, siteID); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	wantComponents := []string{"hero-saas", "feature-grid", "pricing-table", "testimonial-carousel", "faq-accordion", "metric-card"}
	for _, name := range wantComponents {
		if got := count(t, sqlDB, "SELECT COUNT(*) FROM components WHERE site_id=? AND name=?", siteID, name); got != 1 {
			t.Errorf("component %q: want 1, got %d", name, got)
		}
	}
	wantCSS := []string{".gradient-cta", ".metric-card", ".feature-tile"}
	for _, name := range wantCSS {
		if got := count(t, sqlDB, "SELECT COUNT(*) FROM css_classes WHERE site_id=? AND name=?", siteID, name); got != 1 {
			t.Errorf("css class %q: want 1, got %d", name, got)
		}
	}
	wantPages := []string{"/", "/docs", "/pricing", "/about"}
	for _, slug := range wantPages {
		if got := count(t, sqlDB, "SELECT COUNT(*) FROM pages WHERE site_id=? AND slug=?", siteID, slug); got != 1 {
			t.Errorf("page %q: want 1, got %d", slug, got)
		}
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM blocks b JOIN pages p ON b.page_id = p.id WHERE p.site_id=? AND p.slug='/'", siteID); got < 5 {
		t.Errorf("home blocks: want >=5, got %d", got)
	}
}

func TestPortfolioKit_ApplySeedsExpectedRows(t *testing.T) {
	sqlDB, q := setupKitTestDB(t)
	siteID := "site-portfolio"
	insertSite(t, sqlDB, siteID)

	kit, ok := Default.Get("portfolio")
	if !ok {
		t.Fatalf("portfolio not registered")
	}
	if err := kit.Apply(context.Background(), q, siteID); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	wantComponents := []string{"hero-portrait", "project-grid", "about-block", "contact-block"}
	for _, name := range wantComponents {
		if got := count(t, sqlDB, "SELECT COUNT(*) FROM components WHERE site_id=? AND name=?", siteID, name); got != 1 {
			t.Errorf("component %q: want 1, got %d", name, got)
		}
	}
	wantCSS := []string{".scroll-snap", ".project-card"}
	for _, name := range wantCSS {
		if got := count(t, sqlDB, "SELECT COUNT(*) FROM css_classes WHERE site_id=? AND name=?", siteID, name); got != 1 {
			t.Errorf("css class %q: want 1, got %d", name, got)
		}
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM pages WHERE site_id=?", siteID); got != 1 {
		t.Errorf("portfolio page count: want 1 (one-pager), got %d", got)
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM pages WHERE site_id=? AND slug='/tjanster'", siteID); got != 0 {
		t.Errorf("portfolio must not seed /tjanster silo, got %d", got)
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM pages WHERE site_id=? AND slug='/docs'", siteID); got != 0 {
		t.Errorf("portfolio must not seed /docs silo, got %d", got)
	}
}

func TestKits_ApplyIdempotent_ComponentsAndCSS(t *testing.T) {
	sqlDB, q := setupKitTestDB(t)
	siteID := "site-idem"
	insertSite(t, sqlDB, siteID)

	kit, ok := Default.Get("law-firm")
	if !ok {
		t.Fatalf("law-firm not registered")
	}
	// First Apply should succeed.
	if err := kit.Apply(context.Background(), q, siteID); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	componentsBefore := count(t, sqlDB, "SELECT COUNT(*) FROM components WHERE site_id=?", siteID)
	cssBefore := count(t, sqlDB, "SELECT COUNT(*) FROM css_classes WHERE site_id=?", siteID)

	// Second Apply on the same site should not duplicate components or css classes.
	// Re-applying may add additional pages/kb entries, which is acceptable for v1.
	// We only assert idempotency for components and css_classes since those are
	// keyed by name and would otherwise blow up future builds.
	_ = kit.Apply(context.Background(), q, siteID)

	if got := count(t, sqlDB, "SELECT COUNT(*) FROM components WHERE site_id=?", siteID); got != componentsBefore {
		t.Errorf("components not idempotent: before=%d after=%d", componentsBefore, got)
	}
	if got := count(t, sqlDB, "SELECT COUNT(*) FROM css_classes WHERE site_id=?", siteID); got != cssBefore {
		t.Errorf("css_classes not idempotent: before=%d after=%d", cssBefore, got)
	}
}
