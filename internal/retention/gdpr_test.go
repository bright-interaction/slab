package retention

import (
	"context"
	"testing"
	"time"
)

func TestGDPRSweep_DeletesUsersAfterCoolingOff(t *testing.T) {
	sqlDB, queries := openTestDB(t)
	ctx := context.Background()

	// Two users: one requested deletion 10 days ago (overdue under
	// the 7-day default), one requested 2 days ago (still in
	// cooling-off).
	for _, u := range []struct {
		id, email string
		ageDays   int
	}{
		{"u-old", "old@example.com", 10},
		{"u-fresh", "fresh@example.com", 2},
		{"u-untouched", "untouched@example.com", -1}, // -1 = never requested
	} {
		mustExec(t, sqlDB,
			`INSERT INTO users (id, email, password_hash, name, role) VALUES (?, ?, ?, ?, ?)`,
			u.id, u.email, "x", "n", "editor")
		if u.ageDays >= 0 {
			ts := time.Now().UTC().AddDate(0, 0, -u.ageDays).Format("2006-01-02 15:04:05")
			mustExec(t, sqlDB,
				`UPDATE users SET deletion_requested_at = ? WHERE id = ?`,
				ts, u.id)
		}
	}

	mgr := NewManager(queries, sqlDB)
	res, err := mgr.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsersHardDeleted != 1 {
		t.Errorf("UsersHardDeleted = %d, want 1 (only u-old past the 7-day default)", res.UsersHardDeleted)
	}

	for _, expect := range []struct {
		id     string
		exists bool
	}{
		{"u-old", false},
		{"u-fresh", true},
		{"u-untouched", true},
	} {
		var n int
		if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, expect.id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		got := n > 0
		if got != expect.exists {
			t.Errorf("user %s exists=%v, want %v", expect.id, got, expect.exists)
		}
	}
}

func TestGDPRSweep_CustomCoolingDays(t *testing.T) {
	sqlDB, queries := openTestDB(t)
	ctx := context.Background()
	mustExec(t, sqlDB,
		`INSERT INTO users (id, email, password_hash, name, role) VALUES (?, ?, ?, ?, ?)`,
		"u-1d", "x@y.z", "x", "n", "editor")
	twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`UPDATE users SET deletion_requested_at = ? WHERE id = ?`,
		twoHoursAgo, "u-1d")

	mgr := NewManager(queries, sqlDB)
	// 1-day cooling: 2 hours is NOT enough.
	mgr.SetGDPRDeleteCoolingDays(1)
	res, err := mgr.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsersHardDeleted != 0 {
		t.Errorf("UsersHardDeleted = %d, want 0 (2 hours < 1 day cooling)", res.UsersHardDeleted)
	}

	// 0 cooling falls back to default (7d). The same user is still
	// inside the window.
	mgr.SetGDPRDeleteCoolingDays(0)
	res, err = mgr.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsersHardDeleted != 0 {
		t.Errorf("UsersHardDeleted = %d under default 7d cooling, want 0", res.UsersHardDeleted)
	}
}

func TestGDPRSweep_AuditLogRowSurvives(t *testing.T) {
	// Compliance: when we hard-delete a user, an audit_log row must
	// be written FIRST so the proof-of-deletion survives the
	// cascade. The row carries action=delete + resource_type=
	// user_deletion_completed.
	sqlDB, queries := openTestDB(t)
	ctx := context.Background()
	mustExec(t, sqlDB,
		`INSERT INTO users (id, email, password_hash, name, role) VALUES (?, ?, ?, ?, ?)`,
		"u-evidence", "evidence@example.com", "x", "n", "editor")
	old := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02 15:04:05")
	mustExec(t, sqlDB,
		`UPDATE users SET deletion_requested_at = ? WHERE id = ?`,
		old, "u-evidence")

	mgr := NewManager(queries, sqlDB)
	if _, err := mgr.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM audit_log WHERE resource_type = 'user_deletion_completed' AND resource_id = ?`,
		"u-evidence").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected 1 audit_log row for user_deletion_completed, got %d", n)
	}
}

func TestEscapeJSONString(t *testing.T) {
	cases := map[string]string{
		"alice@example.com": "alice@example.com",
		`"quoted"`:          `\"quoted\"`,
		"a\\b":              "a\\\\b",
		"a\nb":              "a\\nb",
		"a\x00b":            "ab", // control byte stripped
	}
	for in, want := range cases {
		if got := escapeJSONString(in); got != want {
			t.Errorf("escapeJSONString(%q) = %q, want %q", in, got, want)
		}
	}
}
