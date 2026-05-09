package retention

import (
	"context"
	"testing"
	"time"

	"github.com/bright-interaction/slab/internal/store"
)

func TestLifecycle_PauseAndDeleteThresholds(t *testing.T) {
	sqlDB, queries := openTestDB(t)
	ctx := context.Background()

	// Active workspace idle for 35 days. With pauseDays=30, the
	// sweep should flip its status to 'paused' but NOT 'deleted'
	// because deleteDays=365.
	if err := queries.CreateWorkspace(ctx, store.CreateWorkspaceParams{
		ID: "ws-stale", Name: "Stale", Slug: "stale", Plan: "solo",
		Region: "eu", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().UTC().AddDate(0, 0, -35).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`UPDATE workspaces SET created_at = ?, updated_at = ? WHERE id = ?`,
		stale, stale, "ws-stale")

	// Fresh workspace 5 days old, status active. Sweep should NOT touch.
	if err := queries.CreateWorkspace(ctx, store.CreateWorkspaceParams{
		ID: "ws-fresh", Name: "Fresh", Slug: "fresh", Plan: "solo",
		Region: "eu", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	fresh := time.Now().UTC().AddDate(0, 0, -5).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`UPDATE workspaces SET created_at = ?, updated_at = ? WHERE id = ?`,
		fresh, fresh, "ws-fresh")

	// Long-idle paused workspace 400 days old. Sweep should flip to
	// 'deleted' because deleteDays=365.
	if err := queries.CreateWorkspace(ctx, store.CreateWorkspaceParams{
		ID: "ws-zombie", Name: "Zombie", Slug: "zombie", Plan: "solo",
		Region: "eu", Status: "paused",
	}); err != nil {
		t.Fatal(err)
	}
	ancient := time.Now().UTC().AddDate(0, 0, -400).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`UPDATE workspaces SET created_at = ?, updated_at = ? WHERE id = ?`,
		ancient, ancient, "ws-zombie")

	mgr := NewManager(queries, sqlDB)
	mgr.SetLifecyclePolicy(30, 365)
	res, err := mgr.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused != 1 {
		t.Errorf("Paused = %d, want 1 (ws-stale)", res.Paused)
	}
	if res.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1 (ws-zombie)", res.Deleted)
	}

	// Verify the rows landed.
	gotStatus := func(id string) string {
		var s string
		if err := sqlDB.QueryRow(`SELECT status FROM workspaces WHERE id = ?`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if got := gotStatus("ws-stale"); got != "paused" {
		t.Errorf("ws-stale status = %q, want paused", got)
	}
	if got := gotStatus("ws-fresh"); got != "active" {
		t.Errorf("ws-fresh status = %q, want active (untouched)", got)
	}
	if got := gotStatus("ws-zombie"); got != "deleted" {
		t.Errorf("ws-zombie status = %q, want deleted", got)
	}
}

func TestLifecycle_DisabledByDefault(t *testing.T) {
	sqlDB, queries := openTestDB(t)
	ctx := context.Background()
	// Create a 1000-day-idle workspace. With both thresholds = 0
	// (the OSS default), the sweep MUST NOT touch it.
	if err := queries.CreateWorkspace(ctx, store.CreateWorkspaceParams{
		ID: "ws-untouched", Name: "U", Slug: "u", Plan: "oss",
		Region: "eu", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().UTC().AddDate(0, 0, -1000).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`UPDATE workspaces SET created_at = ?, updated_at = ? WHERE id = ?`,
		old, old, "ws-untouched")

	mgr := NewManager(queries, sqlDB)
	res, err := mgr.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused != 0 || res.Deleted != 0 {
		t.Errorf("disabled lifecycle should produce 0/0; got paused=%d deleted=%d", res.Paused, res.Deleted)
	}
	var s string
	if err := sqlDB.QueryRow(`SELECT status FROM workspaces WHERE id = ?`, "ws-untouched").Scan(&s); err != nil {
		t.Fatal(err)
	}
	if s != "active" {
		t.Errorf("status = %q, want active (sweep was disabled)", s)
	}
}

func TestLifecycle_DeleteTakesPrecedenceOverPause(t *testing.T) {
	// A 400-day-idle ACTIVE workspace under pause=30 + delete=365
	// should land as 'deleted', not 'paused'. The sweep does NOT
	// transition through paused first.
	sqlDB, queries := openTestDB(t)
	ctx := context.Background()
	if err := queries.CreateWorkspace(ctx, store.CreateWorkspaceParams{
		ID: "ws-deep-stale", Name: "X", Slug: "x", Plan: "solo",
		Region: "eu", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().UTC().AddDate(0, 0, -400).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`UPDATE workspaces SET created_at = ?, updated_at = ? WHERE id = ?`,
		old, old, "ws-deep-stale")

	mgr := NewManager(queries, sqlDB)
	mgr.SetLifecyclePolicy(30, 365)
	res, err := mgr.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", res.Deleted)
	}
	if res.Paused != 0 {
		t.Errorf("Paused = %d, want 0 (delete took precedence)", res.Paused)
	}
	var s string
	if err := sqlDB.QueryRow(`SELECT status FROM workspaces WHERE id = ?`, "ws-deep-stale").Scan(&s); err != nil {
		t.Fatal(err)
	}
	if s != "deleted" {
		t.Errorf("status = %q, want deleted", s)
	}
}

func TestLifecycle_BumpsResetClockViaSiteActivity(t *testing.T) {
	// A workspace whose row is ancient but whose SITE was just
	// updated should NOT pause: sites.updated_at counts as activity.
	sqlDB, queries := openTestDB(t)
	ctx := context.Background()
	if err := queries.CreateWorkspace(ctx, store.CreateWorkspaceParams{
		ID: "ws-recent-site", Name: "R", Slug: "r", Plan: "solo",
		Region: "eu", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().UTC().AddDate(0, 0, -100).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`UPDATE workspaces SET created_at = ?, updated_at = ? WHERE id = ?`,
		old, old, "ws-recent-site")
	// Site updated yesterday (within the 30-day pause window).
	mustExec(t, sqlDB,
		`INSERT INTO sites (id, name, slug, workspace_id, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"abcdef0123456789abcdef99", "S", "s-recent", "ws-recent-site",
		time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))

	mgr := NewManager(queries, sqlDB)
	mgr.SetLifecyclePolicy(30, 365)
	res, err := mgr.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused != 0 {
		t.Errorf("recent site activity must reset the clock; paused=%d", res.Paused)
	}
}
