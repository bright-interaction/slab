package retention

import (
	"context"
	"testing"
	"time"
)

func TestAuditLog_OldRowsPurged(t *testing.T) {
	sqlDB, queries := openTestDB(t)
	ctx := context.Background()

	// 365-day default → old: 400d ago, fresh: 30d ago.
	old := time.Now().UTC().AddDate(0, 0, -400).Format("2006-01-02 15:04:05")
	fresh := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`INSERT INTO audit_log (id, action, resource_type, resource_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		"audit-old", "delete", "site", "s1", old)
	mustExec(t, sqlDB,
		`INSERT INTO audit_log (id, action, resource_type, resource_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		"audit-fresh", "create", "site", "s1", fresh)

	mgr := NewManager(queries, sqlDB)
	res, err := mgr.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.AuditLog != 1 {
		t.Errorf("AuditLog deleted = %d, want 1", res.AuditLog)
	}
	if res.AuditDays != DefaultAuditLogDays {
		t.Errorf("AuditDays = %d, want %d", res.AuditDays, DefaultAuditLogDays)
	}
	remaining := mustCount(t, sqlDB, `SELECT COUNT(*) FROM audit_log`)
	if remaining != 1 {
		t.Errorf("rows remaining = %d, want 1 (fresh row)", remaining)
	}
	id := ""
	if err := sqlDB.QueryRow(`SELECT id FROM audit_log`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if id != "audit-fresh" {
		t.Errorf("survivor id = %q, want audit-fresh", id)
	}
}

func TestAuditLog_CustomRetentionWindow(t *testing.T) {
	sqlDB, queries := openTestDB(t)
	ctx := context.Background()

	// Set 30-day retention. Insert a 60-day-old row + a 7-day-old row.
	mgr := NewManager(queries, sqlDB)
	mgr.SetAuditLogRetentionDays(30)

	old := time.Now().UTC().AddDate(0, 0, -60).Format("2006-01-02 15:04:05")
	fresh := time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`INSERT INTO audit_log (id, action, resource_type, created_at) VALUES (?, ?, ?, ?)`,
		"a1", "delete", "site", old)
	mustExec(t, sqlDB,
		`INSERT INTO audit_log (id, action, resource_type, created_at) VALUES (?, ?, ?, ?)`,
		"a2", "create", "site", fresh)

	res, err := mgr.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.AuditDays != 30 {
		t.Errorf("AuditDays = %d, want 30", res.AuditDays)
	}
	if res.AuditLog != 1 {
		t.Errorf("deleted = %d, want 1", res.AuditLog)
	}
}

func TestAuditLog_OutOfRangeRetentionClamps(t *testing.T) {
	_, queries := openTestDB(t)
	mgr := NewManager(queries, nil)

	mgr.SetAuditLogRetentionDays(1) // below MinRetentionDays
	if got := mgr.getAuditLogRetentionDays(); got != MinRetentionDays {
		t.Errorf("retention=%d, want clamped to %d", got, MinRetentionDays)
	}
	mgr.SetAuditLogRetentionDays(99999) // above MaxRetentionDays
	if got := mgr.getAuditLogRetentionDays(); got != MaxRetentionDays {
		t.Errorf("retention=%d, want clamped to %d", got, MaxRetentionDays)
	}
	mgr.SetAuditLogRetentionDays(0) // unset / zero
	if got := mgr.getAuditLogRetentionDays(); got != DefaultAuditLogDays {
		t.Errorf("retention=%d, want default %d", got, DefaultAuditLogDays)
	}
	mgr.SetAuditLogRetentionDays(180)
	if got := mgr.getAuditLogRetentionDays(); got != 180 {
		t.Errorf("retention=%d, want 180", got)
	}
}

func TestAuditLog_EmptyTableNoOps(t *testing.T) {
	sqlDB, queries := openTestDB(t)
	mgr := NewManager(queries, sqlDB)
	res, err := mgr.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.AuditLog != 0 {
		t.Errorf("empty table should delete 0 rows, got %d", res.AuditLog)
	}
	if len(res.Errors) > 0 {
		t.Errorf("expected no errors, got %v", res.Errors)
	}
}
