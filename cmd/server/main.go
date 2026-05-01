// Command server is the AtomicSite HTTP server.
package main

import (
	"bufio"
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"

	"github.com/brightinteraction/atomicsite/internal/analytics"
	"github.com/brightinteraction/atomicsite/internal/analyticsdb"
	"github.com/brightinteraction/atomicsite/internal/builder"
	"github.com/brightinteraction/atomicsite/internal/config"
	dbpkg "github.com/brightinteraction/atomicsite/internal/db"
	"github.com/brightinteraction/atomicsite/internal/retention"
	"github.com/brightinteraction/atomicsite/internal/server"
	"github.com/brightinteraction/atomicsite/internal/storage"
	"github.com/brightinteraction/atomicsite/internal/store"
)

//go:embed all:frontend/build
var frontendFiles embed.FS

func main() {
	// Subcommand dispatch. Anything other than the (default) HTTP server lands
	// here. Subcommands open their own DB connection and exit.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "reset-password":
			resetPasswordCLI(os.Args[2:])
			return
		}
	}

	cfg := config.Load()
	// Refuse to boot in production-like deployments (BaseURL != localhost)
	// when any documented-default secret is still in place. The seeded
	// admin password is checked separately at seed time below. Keeps local
	// dev permissive while making "deployed without env" impossible to miss.
	if err := cfg.Validate(); err != nil {
		slog.Error("config validation failed; refusing to start", "error", err.Error())
		os.Exit(1)
	}
	// Default admin base URL for the personalization hydration script
	// (Phase 18.2). Per-site override via the analytics.admin_base_url
	// setting; falls back to cfg.BaseURL otherwise.
	builder.DefaultAdminBaseURL = cfg.BaseURL

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		slog.Error("create data dir", "error", err)
		os.Exit(1)
	}

	// Open SQLite DB
	sqlDB, err := sql.Open("sqlite", cfg.DBPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		slog.Error("open db", "error", err)
		os.Exit(1)
	}
	sqlDB.SetMaxOpenConns(1)
	defer sqlDB.Close()

	// Apply schema
	if err := applySchema(sqlDB); err != nil {
		slog.Error("apply schema", "error", err)
		os.Exit(1)
	}

	queries := store.New(sqlDB)

	// Seed admin user if none exists
	seedAdminUser(cfg, queries, sqlDB)

	// Audit C1 bootstrap: grant the seed admin ownership of every existing
	// site that has no member rows yet. Runs every boot but is idempotent
	// (the SQL guards with NOT EXISTS), so it's free after the first
	// successful migration. Without this, an existing deployment upgrading
	// across the C1 fix would lose admin access to all its sites because
	// non-admin role users (added later via /api/admin/invites) would 403
	// on every site-scoped route.
	if users, _ := queries.ListUsers(context.Background()); len(users) > 0 {
		for _, u := range users {
			if u.Role == "admin" {
				if err := queries.BackfillSiteMembersForAdmin(context.Background(), u.ID); err != nil {
					slog.Warn("backfill site_members for admin failed", "user_id", u.ID, "err", err)
				}
				break // Only the first admin needs the backfill row.
			}
		}
	}

	// Set embedded frontend FS
	sub, err := fs.Sub(frontendFiles, "frontend/build")
	if err != nil {
		slog.Warn("no embedded frontend", "error", err)
	} else {
		server.FrontendFS = sub
	}

	// Media storage
	st, err := storage.NewLocalStore(cfg.MediaDir)
	if err != nil {
		slog.Error("create storage", "error", err)
		os.Exit(1)
	}

	// Analytics manager: tails per-site Nginx JSON logs and writes visit_events.
	// Reload-on-settings-change is delegated to handlers (they call Reload after
	// toggling analytics.atomicsite_tracking_enabled).
	analyticsMgr := analytics.NewManager(queries, cfg.AnalyticsSalt)
	mgrCtx, mgrCancel := context.WithCancel(context.Background())
	defer mgrCancel()
	if err := analyticsMgr.Start(mgrCtx); err != nil {
		slog.Warn("analytics: initial start failed", "error", err)
	}

	// Retention manager: daily sweep that purges old visit_events, visit_engagement,
	// visit_sessions, consent_records, and consent_salts. Per-site retention windows
	// come from the general.{analytics,consent,engagement}_retention_days settings;
	// defaults are 180 / 730 / 90 days. Without this the DB grows unbounded; at
	// moderate traffic (10 sites, 1k visitors/day, 5 pageviews each) visit_events
	// alone gains ~7 GB/year.
	retentionMgr := retention.NewManager(queries, sqlDB)
	// Audit H8: tell the retention manager where workspaces live so its
	// daily sweep can reap orphans (workspaces dirs whose siteID is no
	// longer in the DB).
	retentionMgr.SetDataDir(cfg.DataDir)
	retentionMgr.Start(mgrCtx)

	// Stitched cookie-analytics layer: DuckDB ATTACHes the live SQLite file
	// read-only and serves the /api/sites/{id}/cookies/analytics/* endpoints.
	// Single source of truth (one .db file), columnar query engine on top.
	// If DuckDB fails to open (missing extension, CGO disabled, etc.) we
	// log + continue without analytics rather than abort startup; the
	// CookieAnalyticsHandler returns 503 from each endpoint in that case.
	analyticsMgrCtx, analyticsMgrCancel := context.WithTimeout(context.Background(), 30*time.Second)
	analyticsMgrConn, err := analyticsdb.Open(analyticsMgrCtx, analyticsdb.Config{SQLitePath: cfg.DBPath})
	analyticsMgrCancel()
	if err != nil {
		slog.Warn("analyticsdb: failed to open; stitched analytics endpoints will return 503", "error", err)
		analyticsMgrConn = nil
	}

	srv := server.New(cfg, sqlDB, queries, st)
	srv.AnalyticsDB = analyticsMgrConn
	srv.OnAnalyticsSettingsChange = func(_ context.Context) {
		// Use a fresh background context: the request that triggered this may
		// finish before Reload completes, and we don't want a cancelled context
		// to abort the rescan.
		if err := analyticsMgr.Reload(context.Background()); err != nil {
			slog.Warn("analytics: reload after settings change failed", "error", err)
		}
	}
	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      srv.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("atomicsite: listening", "port", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("atomicsite: shutting down")

	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutCtx)
	analyticsMgr.Stop(shutCtx)
	retentionMgr.Stop(shutCtx)
	if analyticsMgrConn != nil {
		_ = analyticsMgrConn.Close()
	}

	slog.Info("atomicsite: stopped")
}

func applySchema(sqlDB *sql.DB) error {
	// Run column migrations FIRST. addColumnIfMissing is now table-aware: if
	// the table doesn't exist (fresh DB), it no-ops and the schema below
	// will create the table with all columns. If the table exists from an
	// earlier deploy with fewer columns, the missing columns are ALTER-ADDed
	// here so that subsequent CREATE INDEX statements in the schema (which
	// reference these columns) don't fail with "no such column".
	//
	// This ordering matters: a 502 incident on 2026-04-26 was caused by the
	// analytics enrichment columns (browser, os, device, country, lang,
	// utm_*) being added to schema.sql + indexed in the same statement-set
	// without ALTER migrations, so existing prod DBs failed to apply the
	// schema (CREATE INDEX referenced country which didn't exist).
	migrations := []struct{ table, column, spec string }{
		{"pages", "no_index", "INTEGER NOT NULL DEFAULT 0"},
		{"pages", "canonical_url", "TEXT NOT NULL DEFAULT ''"},
		{"deployments", "target_id", "TEXT NOT NULL DEFAULT ''"},
		{"deployments", "deploy_url", "TEXT NOT NULL DEFAULT ''"},
		{"deployments", "deployed_at", "TEXT NOT NULL DEFAULT ''"},
		{"site_silos", "silo_type", "TEXT NOT NULL DEFAULT 'inherit'"},
		// Analytics enrichment (Phase 12.5) added 2026-04-26.
		{"visit_events", "browser", "TEXT NOT NULL DEFAULT ''"},
		{"visit_events", "os", "TEXT NOT NULL DEFAULT ''"},
		{"visit_events", "device", "TEXT NOT NULL DEFAULT ''"},
		{"visit_events", "country", "TEXT NOT NULL DEFAULT ''"},
		{"visit_events", "lang", "TEXT NOT NULL DEFAULT ''"},
		{"visit_events", "utm_source", "TEXT NOT NULL DEFAULT ''"},
		{"visit_events", "utm_medium", "TEXT NOT NULL DEFAULT ''"},
		{"visit_events", "utm_campaign", "TEXT NOT NULL DEFAULT ''"},
		// Branding extra colour slots (Phase 12.9) added 2026-04-27.
		{"sites", "surface_color", "TEXT NOT NULL DEFAULT '#FFFFFF'"},
		{"sites", "border_color", "TEXT NOT NULL DEFAULT '#E5E7EB'"},
		{"sites", "muted_color", "TEXT NOT NULL DEFAULT '#6B7280'"},
		{"sites", "accent_color", "TEXT NOT NULL DEFAULT ''"},
		{"sites", "on_primary_color", "TEXT NOT NULL DEFAULT '#FFFFFF'"},
		// Media folders (Phase 13) added 2026-04-27.
		{"media", "folder", "TEXT NOT NULL DEFAULT ''"},
		// Bidirectional CRM personalization (Phase 18) added 2026-04-28.
		// Required before applySchema executes the matching CREATE INDEX
		// on existing prod DBs (mirrors the 2026-04-26 502 fix pattern).
		{"visit_sessions", "metadata_json", "TEXT NOT NULL DEFAULT '{}'"},
		{"visit_sessions", "metadata_expires_at", "TEXT NOT NULL DEFAULT ''"},
		{"visit_sessions", "identity_confirmed_at", "TEXT NOT NULL DEFAULT ''"},
		// Per-page suppression of global blocks (Phase 14) added 2026-04-29.
		// Lets a landing page hide site-wide nav and footer for higher
		// conversion without breaking the global blocks for the rest of
		// the site.
		{"pages", "hide_global_blocks", "INTEGER NOT NULL DEFAULT 0"},
		// Trusted-domain kind (Phase 15) added 2026-04-29. Existing rows
		// default to 'script' so the upgrade is no-op for current sites.
		// Note: schema.sql also widens UNIQUE to (site_id, domain, kind)
		// but addColumnIfMissing can't redefine constraints; SQLite ignores
		// the older UNIQUE on existing tables and allows the new (id, kind)
		// combination at the application layer (handler dedupes per kind).
		{"allowed_scripts", "kind", "TEXT NOT NULL DEFAULT 'script'"},
		// Cookie analytics stitch key (Phase 11.2) added 2026-05-01.
		// consent_records.fingerprint joins to visit_events / visit_sessions
		// / visit_engagement on the existing fingerprint column so the
		// DuckDB analytics layer can do (site_id, fingerprint) joins
		// without going through visit_sessions.session_id (which doesn't
		// match the client's atomicsite_sid for new visitors).
		{"consent_records", "fingerprint", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, m := range migrations {
		if err := addColumnIfMissing(sqlDB, m.table, m.column, m.spec); err != nil {
			return fmt.Errorf("migrate %s.%s: %w", m.table, m.column, err)
		}
	}
	// Now apply the full schema. CREATE TABLE IF NOT EXISTS is a no-op for
	// existing tables; CREATE INDEX IF NOT EXISTS now succeeds because the
	// referenced columns exist (newly added by the migrations above on
	// existing DBs, or about to be created on fresh DBs).
	if _, err := sqlDB.Exec(string(dbpkg.Schema)); err != nil {
		return err
	}
	// Seed system 'brand' folder for every existing site that doesn't have
	// one. Idempotent thanks to INSERT OR IGNORE on the composite PK. New
	// sites get the row from their own creation handlers.
	if _, err := sqlDB.Exec(`INSERT OR IGNORE INTO media_folders (site_id, name, is_system)
		SELECT id, 'brand', 1 FROM sites`); err != nil {
		return fmt.Errorf("seed brand folder: %w", err)
	}
	return nil
}

// addColumnIfMissing checks PRAGMA table_info and adds the column only if it's
// not present. No-ops when the table itself does not exist yet (fresh DB),
// in which case the caller's schema apply will create the table with the
// column already declared. Works on all SQLite versions (doesn't require
// ALTER TABLE IF NOT EXISTS which is SQLite 3.35+).
func addColumnIfMissing(sqlDB *sql.DB, table, column, spec string) error {
	// Probe table existence first. Without this, PRAGMA returns no rows on
	// a non-existent table and the subsequent ALTER TABLE fails with
	// "no such table". When applySchema runs migrations BEFORE the schema
	// exec (the safe ordering), fresh DBs have no tables yet, so we must
	// no-op here and let CREATE TABLE handle column declarations.
	var name string
	if err := sqlDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	rows, err := sqlDB.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var colName, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &colName, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if colName == column {
			return nil // already present
		}
	}
	_, err = sqlDB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, spec))
	return err
}

func seedAdminUser(cfg *config.Config, queries *store.Queries, sqlDB *sql.DB) {
	ctx := context.Background()
	users, err := queries.ListUsers(ctx)
	if err != nil || len(users) > 0 {
		return
	}

	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")
	if email == "" {
		email = "admin@atomicsite.dev"
	}

	// Production-like deployments must set ADMIN_PASSWORD explicitly.
	// The audit's C3 finding: defaulting to "changeme123" lets a deploy
	// without env vars come up with a known-by-everyone admin login.
	// Local dev keeps the convenience default so contributors don't have
	// to set five env vars to run `go run`.
	if err := cfg.ValidateAdminPassword(password); err != nil {
		slog.Error("seed admin: refusing to seed with weak/default password", "error", err.Error())
		// Fail hard rather than silently using changeme123. The first
		// boot's logs are the moment to surface this.
		os.Exit(1)
	}
	if password == "" {
		// Reached only in local dev (Validate would have refused otherwise).
		password = config.DefaultAdminPassword
		slog.Warn("seeded admin with documented default password — local dev only", "email", email)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("seed admin: hash password", "error", err)
		return
	}

	err = queries.CreateUser(ctx, store.CreateUserParams{
		ID:           "admin",
		Email:        email,
		PasswordHash: string(hash),
		Name:         "Admin",
		Role:         "admin",
	})
	if err != nil {
		slog.Error("seed admin: create user", "error", err)
		return
	}

	slog.Info("seeded admin user", "email", email)
}

// resetPasswordCLI is the `atomicsite reset-password <email>` subcommand.
// Reads the new password from stdin (one line, no echo-off - run with
// `docker exec -i` from a terminal you trust). Bumps token_version so any
// existing JWTs for that user are invalidated.
//
// Usage:
//
//	docker exec -i atomicsite /app/server reset-password user@example.com
//	(then type the new password and press enter)
func resetPasswordCLI(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: atomicsite reset-password <email>")
		os.Exit(2)
	}
	email := strings.TrimSpace(strings.ToLower(args[0]))
	if email == "" {
		fmt.Fprintln(os.Stderr, "email is required")
		os.Exit(2)
	}

	cfg := config.Load()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "create data dir:", err)
		os.Exit(1)
	}
	sqlDB, err := sql.Open("sqlite", cfg.DBPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer sqlDB.Close()
	queries := store.New(sqlDB)

	ctx := context.Background()
	user, err := queries.GetUserByEmail(ctx, email)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no user with email %q\n", email)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Enter new password for %s: ", email)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "no input")
		os.Exit(1)
	}
	newPwd := strings.TrimRight(scanner.Text(), "\r\n")
	if len(newPwd) < 8 {
		fmt.Fprintln(os.Stderr, "password must be at least 8 characters")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash:", err)
		os.Exit(1)
	}
	if err := queries.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{
		PasswordHash: string(hash),
		ID:           user.ID,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "update:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "password reset for %s (active sessions invalidated)\n", email)
}
