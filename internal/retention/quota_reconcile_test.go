package retention

import (
	"context"
	"strings"
	"testing"

	"github.com/bright-interaction/slab/internal/store"
)

// seedWorkspaceWithStorage creates a workspace + a site under it +
// a media row of totalBytes. Lets the reconciler tests plant concrete
// usage numbers without going through the upload handler. The siteID
// is a deterministic 24-hex slot keyed by `seq` so multiple seed
// calls in one test don't collide.
func seedWorkspaceWithStorage(t *testing.T, sqlDB any, q *store.Queries, wsID, plan string, totalBytes int64) string {
	t.Helper()
	ctx := context.Background()
	if err := q.CreateWorkspace(ctx, store.CreateWorkspaceParams{
		ID: wsID, Name: wsID, Slug: wsID, Plan: plan, Region: "eu", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	// Build a 24-hex siteID derived from the workspace slug. We hash
	// the wsID with a tiny FNV-1a so nothing has to be 24 chars long
	// at the call site.
	siteID := siteIDForSeed(wsID)
	if err := q.CreateSite(ctx, store.CreateSiteParams{
		ID: siteID, Name: "S", Slug: wsID + "-s", Domain: "", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en", WorkspaceID: wsID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := q.CreateMedia(ctx, store.CreateMediaParams{
		ID: "m_" + wsID, SiteID: siteID, Filename: "f.bin", AltText: "",
		MimeType: "application/octet-stream", FileSize: totalBytes, Width: 1, Height: 1,
		OriginalPath: "/dev/null", Folder: "",
	}); err != nil {
		t.Fatal(err)
	}
	return wsID
}

// siteIDForSeed returns a deterministic 24-hex string derived from
// the input. Avoids the brittle `[:24-len(s)]` slicing the prior
// helper used.
func siteIDForSeed(s string) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 24)
	h := uint64(1469598103934665603)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	for i := 0; i < 24; i++ {
		out[i] = hex[(h>>(i%16*4))&0xf]
		h = h*1099511628211 + uint64(i)
	}
	return string(out)
}

func TestQuotaReconcile_SoloOverStorageWritesAuditRow(t *testing.T) {
	sqlDB, q := openTestDB(t)
	// Solo plan = 1 GiB cap. Seed 2 GiB total - well over.
	seedWorkspaceWithStorage(t, sqlDB, q, "ws-over", "solo", 2*1024*1024*1024)

	mgr := NewManager(q, sqlDB)
	res, err := mgr.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.QuotaOverages != 1 {
		t.Errorf("QuotaOverages = %d, want 1", res.QuotaOverages)
	}
	var n int
	if err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM audit_log WHERE resource_type = 'plan_quota_overage_detected' AND resource_id = ?`,
		"ws-over").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected 1 audit_log row for the overage, got %d", n)
	}
}

func TestQuotaReconcile_OssPlanSkipsAllChecks(t *testing.T) {
	sqlDB, q := openTestDB(t)
	// 100 TB on OSS plan: cap is -1 so the row should NOT appear in
	// the audit_log. The sweep must short-circuit before issuing the
	// SUM query for unlimited plans.
	seedWorkspaceWithStorage(t, sqlDB, q, "ws-oss", "oss", 100*1024*1024*1024*1024)

	mgr := NewManager(q, sqlDB)
	res, err := mgr.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.QuotaOverages != 0 {
		t.Errorf("OSS plan should never report overages; got %d", res.QuotaOverages)
	}
}

func TestQuotaReconcile_UnderCapDoesNotWriteRow(t *testing.T) {
	sqlDB, q := openTestDB(t)
	// 500 MiB on solo (1 GiB cap): under, no overage row.
	seedWorkspaceWithStorage(t, sqlDB, q, "ws-fine", "solo", 500*1024*1024)

	mgr := NewManager(q, sqlDB)
	res, err := mgr.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.QuotaOverages != 0 {
		t.Errorf("under-cap workspace flagged: %d overages", res.QuotaOverages)
	}
}

func TestQuotaReconcile_ChangesJsonContainsCapDelta(t *testing.T) {
	sqlDB, q := openTestDB(t)
	seedWorkspaceWithStorage(t, sqlDB, q, "ws-evidence", "solo", int64(1.5*float64(1024*1024*1024)))

	mgr := NewManager(q, sqlDB)
	if _, err := mgr.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	var changes string
	if err := sqlDB.QueryRow(
		`SELECT changes_json FROM audit_log WHERE resource_id = ? AND resource_type = 'plan_quota_overage_detected'`,
		"ws-evidence").Scan(&changes); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"plan":"solo"`, `"storage_used"`, `"storage_cap"`, `"storage_over":true`, `"build_over":false`} {
		if !strings.Contains(changes, want) {
			t.Errorf("audit changes_json missing %q; got %s", want, changes)
		}
	}
}
