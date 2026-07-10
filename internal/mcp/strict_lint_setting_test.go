package mcp

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	dbpkg "github.com/bright-interaction/atomicsite/internal/db"
	"github.com/bright-interaction/atomicsite/internal/store"
)

// openStrictLintTestDB matches the openTestDB helper in
// internal/retention/retention_test.go so the MCP tests stay aligned.
func openStrictLintTestDB(t *testing.T) (*sql.DB, *store.Queries) {
	t.Helper()
	dir := t.TempDir()
	sqlDB, err := sql.Open("sqlite", filepath.Join(dir, "t.sqlite")+"?_journal_mode=WAL&_busy_timeout=5000&_pragma=foreign_keys(1)")
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

func seedSiteAndSetting(t *testing.T, sqlDB *sql.DB, siteID, key, val string) {
	t.Helper()
	if _, err := sqlDB.Exec(`INSERT INTO sites (id, name, slug) VALUES (?, ?, ?)`, siteID, "T", siteID); err != nil {
		t.Fatalf("seed site: %v", err)
	}
	if val == "" {
		return
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO site_settings (id, site_id, category, key, value) VALUES (?, ?, 'design', ?, ?)`,
		"setting_"+siteID+"_"+key, siteID, key, val,
	); err != nil {
		t.Fatalf("seed setting: %v", err)
	}
}

// TestStrictDesignLintEnabled_DefaultsOn confirms a site with no row gets
// strict mode by default. This is the load-bearing rule: design enforcement
// is opt-OUT, not opt-IN.
func TestStrictDesignLintEnabled_DefaultsOn(t *testing.T) {
	_, queries := openStrictLintTestDB(t)
	if !strictDesignLintEnabled(context.Background(), queries, "no-such-site") {
		t.Error("expected strict mode ON for missing setting")
	}
}

// TestStrictDesignLintEnabled_RespectsOperatorOptOut covers the five literal
// forms an operator can use to disable the gate from the settings UI.
func TestStrictDesignLintEnabled_RespectsOperatorOptOut(t *testing.T) {
	cases := []struct {
		stored string
		want   bool
	}{
		{"1", true},
		{"true", true},
		{"on", true},
		{"yes", true},
		{"0", false},
		{"false", false},
		{"off", false},
		{"no", false},
		{"FALSE", false},
		{"  off  ", false},
		{"anything-else", true},
	}
	for _, c := range cases {
		t.Run(c.stored, func(t *testing.T) {
			sqlDB, queries := openStrictLintTestDB(t)
			siteID := "site_" + c.stored
			seedSiteAndSetting(t, sqlDB, siteID, "strict_lint", c.stored)
			if got := strictDesignLintEnabled(context.Background(), queries, siteID); got != c.want {
				t.Errorf("strict mode for stored=%q: got %v, want %v", c.stored, got, c.want)
			}
		})
	}
}
