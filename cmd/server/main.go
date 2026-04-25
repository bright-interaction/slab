// Command server is the AtomicSite HTTP server.
package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"

	"github.com/bright-interaction/slab/internal/config"
	dbpkg "github.com/bright-interaction/slab/internal/db"
	"github.com/bright-interaction/slab/internal/server"
	"github.com/bright-interaction/slab/internal/storage"
	"github.com/bright-interaction/slab/internal/store"
)

//go:embed all:frontend/build
var frontendFiles embed.FS

func main() {
	cfg := config.Load()

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
	seedAdminUser(queries, sqlDB)

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

	srv := server.New(cfg, sqlDB, queries, st)
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

	slog.Info("atomicsite: stopped")
}

func applySchema(sqlDB *sql.DB) error {
	if _, err := sqlDB.Exec(string(dbpkg.Schema)); err != nil {
		return err
	}
	// Idempotent column additions for pre-existing tables (CREATE TABLE IF NOT
	// EXISTS above doesn't alter existing tables). addColumnIfMissing silently
	// no-ops when the column already exists.
	migrations := []struct{ table, column, spec string }{
		{"pages", "no_index", "INTEGER NOT NULL DEFAULT 0"},
		{"pages", "canonical_url", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, m := range migrations {
		if err := addColumnIfMissing(sqlDB, m.table, m.column, m.spec); err != nil {
			return fmt.Errorf("migrate %s.%s: %w", m.table, m.column, err)
		}
	}
	return nil
}

// addColumnIfMissing checks PRAGMA table_info and adds the column only if it's
// not present. Works on all SQLite versions (doesn't require ALTER TABLE IF NOT
// EXISTS which is SQLite 3.35+).
func addColumnIfMissing(sqlDB *sql.DB, table, column, spec string) error {
	rows, err := sqlDB.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil // already present
		}
	}
	_, err = sqlDB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, spec))
	return err
}

func seedAdminUser(queries *store.Queries, sqlDB *sql.DB) {
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
	if password == "" {
		password = "changeme123"
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
