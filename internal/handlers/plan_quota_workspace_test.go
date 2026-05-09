package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bright-interaction/slab/internal/store"
)

// seedWorkspaceForQuota creates a workspace + assigns the seeded site
// to it, then sets the workspace plan to the given key. Returns the
// workspace ID so the caller can plant additional sites if needed.
func seedWorkspaceForQuota(t *testing.T, q *store.Queries, siteID, plan string) string {
	t.Helper()
	wsID := "ws_quota_" + plan
	if err := q.CreateWorkspace(context.Background(), store.CreateWorkspaceParams{
		ID: wsID, Name: "QA", Slug: "qa", Plan: plan, Region: "eu",
		BillingEmail: "", Status: "active", TrialEndsAt: "",
	}); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	// Move the seeded site into this workspace so the plan-quota
	// helpers find it. BackfillSitesWorkspace is idempotent and only
	// touches sites with workspace_id=''.
	if err := q.BackfillSitesWorkspace(context.Background(), wsID); err != nil {
		t.Fatalf("backfill sites workspace: %v", err)
	}
	return wsID
}

func TestCheckPlanStorageQuota_OssIsUnlimited(t *testing.T) {
	h, q, siteID := newQuotaHandlerForTest(t)
	seedWorkspaceForQuota(t, q, siteID, "oss")
	seedMediaRow(t, q, siteID, "m1", 5*gigabyte)

	check, err := h.CheckPlanStorageQuota(context.Background(), siteID, 100*gigabyte)
	if err != nil {
		t.Fatal(err)
	}
	if !check.Allowed || check.Limit != -1 {
		t.Errorf("oss plan should be unlimited; got %+v", check)
	}
}

func TestCheckPlanStorageQuota_SoloBlocksOver1GB(t *testing.T) {
	h, q, siteID := newQuotaHandlerForTest(t)
	seedWorkspaceForQuota(t, q, siteID, "solo")
	// Already 900 MB used; pending upload of 200 MB pushes over 1 GiB.
	seedMediaRow(t, q, siteID, "m1", 900*1024*1024)

	check, err := h.CheckPlanStorageQuota(context.Background(), siteID, 200*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if check.Allowed {
		t.Error("solo plan should block at >1 GiB pending")
	}
	if check.Limit != gigabyte {
		t.Errorf("solo limit = %d, want %d", check.Limit, gigabyte)
	}
}

func TestCheckPlanStorageQuota_SoloAllowsAtCap(t *testing.T) {
	h, q, siteID := newQuotaHandlerForTest(t)
	seedWorkspaceForQuota(t, q, siteID, "solo")
	// Exactly 1 GiB - 100 bytes used; 100 bytes pending lands at cap
	// (allowed because Allowed = used <= limit).
	seedMediaRow(t, q, siteID, "m1", gigabyte-100)

	check, err := h.CheckPlanStorageQuota(context.Background(), siteID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !check.Allowed {
		t.Errorf("expected allowed at exact cap; got %+v", check)
	}
}

func TestCheckPlanStorageQuota_AggregatesAcrossSites(t *testing.T) {
	// Solo plan: 1 GB cap workspace-wide. Two sites each at 600 MB =
	// 1.2 GB total. Even though each site individually is under 1 GB,
	// the workspace rollup blocks.
	h, q, siteID := newQuotaHandlerForTest(t)
	wsID := seedWorkspaceForQuota(t, q, siteID, "solo")
	site2 := "abcdef0123456789abcdef02"
	if err := q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: site2, Name: "T2", Slug: "t2", Domain: "", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en", WorkspaceID: wsID,
	}); err != nil {
		t.Fatal(err)
	}
	seedMediaRow(t, q, siteID, "m1", 600*1024*1024)
	seedMediaRow(t, q, site2, "m2", 600*1024*1024)

	check, err := h.CheckPlanStorageQuota(context.Background(), siteID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if check.Allowed {
		t.Error("workspace at 1.2 GB on solo (1 GB cap) must be blocked")
	}
	wantMin := gigabyte + (gigabyte / 10) // 1.1 GiB
	if check.Used < wantMin {
		t.Errorf("expected workspace-aggregated Used >= 1.1 GiB, got %d", check.Used)
	}
}

func TestCheckPlanBuildMinutesQuota_StudioCapEnforced(t *testing.T) {
	h, q, siteID := newQuotaHandlerForTest(t)
	seedWorkspaceForQuota(t, q, siteID, "studio") // 3000 build minutes
	// Plant a deployment with 3001 minutes (in ms).
	seedCompletedDeployment(t, q, siteID, "dep1", 3001*60*1000)

	check, err := h.CheckPlanBuildMinutesQuota(context.Background(), siteID)
	if err != nil {
		t.Fatal(err)
	}
	if check.Allowed {
		t.Errorf("3001 minutes on studio (3000 cap) must block; got %+v", check)
	}
	if check.Limit != 3000 {
		t.Errorf("studio limit = %d, want 3000", check.Limit)
	}
}

func TestEnforceStorageQuota_PlanCap402BeforePerSiteLayer(t *testing.T) {
	h, q, siteID := newQuotaHandlerForTest(t)
	seedWorkspaceForQuota(t, q, siteID, "solo")
	seedMediaRow(t, q, siteID, "m1", gigabyte+1)

	r := httptest.NewRequest("POST", "/api/sites/"+siteID+"/media", nil)
	rr := httptest.NewRecorder()
	short := h.EnforceStorageQuota(rr, r, siteID, 1)
	if !short {
		t.Fatal("expected EnforceStorageQuota to short-circuit at the plan layer")
	}
	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d, want 402 (plan_quota_exceeded)", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] != "plan_quota_exceeded" {
		t.Errorf("error code = %v, want plan_quota_exceeded", body["error"])
	}
	if body["kind"] != "plan_storage" {
		t.Errorf("kind = %v, want plan_storage", body["kind"])
	}
}

func TestEnforceBuildMinutesQuota_PlanCap402BeforePerSiteLayer(t *testing.T) {
	h, q, siteID := newQuotaHandlerForTest(t)
	seedWorkspaceForQuota(t, q, siteID, "solo") // 600 build minutes plan cap
	seedCompletedDeployment(t, q, siteID, "dep1", 601*60*1000)

	r := httptest.NewRequest("POST", "/api/sites/"+siteID+"/build", nil)
	rr := httptest.NewRecorder()
	short := h.EnforceBuildMinutesQuota(rr, r, siteID)
	if !short {
		t.Fatal("expected EnforceBuildMinutesQuota to short-circuit at the plan layer")
	}
	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d, want 402", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["kind"] != "plan_build_minutes" {
		t.Errorf("kind = %v, want plan_build_minutes", body["kind"])
	}
}

func TestEnforceStorageQuota_OssPlanFallsThroughToPerSiteLayer(t *testing.T) {
	// On the OSS plan there is no plan-level cap, so the per-site
	// layer (1 GiB default) is what fires. We seed 2 GiB of media so
	// the per-site cap blocks; if the plan layer were short-circuiting
	// to "allowed" without then deferring to the per-site check, we'd
	// see allowed=true here.
	h, q, siteID := newQuotaHandlerForTest(t)
	seedWorkspaceForQuota(t, q, siteID, "oss")
	seedMediaRow(t, q, siteID, "m1", 2*gigabyte)

	r := httptest.NewRequest("POST", "/api/sites/"+siteID+"/media", nil)
	rr := httptest.NewRecorder()
	short := h.EnforceStorageQuota(rr, r, siteID, 1)
	if !short {
		t.Fatal("per-site layer should still block on OSS plan when over 1 GiB default")
	}
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429 (per-site layer)", rr.Code)
	}
}
