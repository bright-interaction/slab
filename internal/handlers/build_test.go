package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// 24-char hex ID matching safeSiteIDPattern.
const testSiteID = "aaaaaaaaaaaaaaaaaaaaaaaa"

// seedSiteWithSafeID creates a site row whose ID matches isSafeSiteID so
// the admin handler doesn't reject the request before doing anything.
func seedSiteWithSafeID(t *testing.T, q *store.Queries, id string) {
	t.Helper()
	if err := q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: id, Name: "Test", Slug: id, Domain: "",
		PrimaryColor: "#000000", SecondaryColor: "#000000",
		BgColor: "#FFFFFF", TextColor: "#000000",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en",
	}); err != nil {
		t.Fatalf("create site: %v", err)
	}
}

func buildRouter(h *BuildHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/api/sites/{siteID}/deploy", h.Deploy)
	return r
}

func postJSON(t *testing.T, r chi.Router, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if len(raw) == 0 {
		return resp.StatusCode, nil
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

// TestBuildHandler_Deploy_LocalTarget_Success drives the full Deploy handler
// path with a Local target. We pre-create a "completed" build state in
// memory + matching deployment row, then POST /deploy and verify the dist
// landed at the local path and the DB row was updated.
func TestBuildHandler_Deploy_LocalTarget_Success(t *testing.T) {
	sqlDB, q := setupDeployTestDB(t)
	seedSiteWithSafeID(t, q, testSiteID)

	// Synthesise a built dist.
	distDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatalf("write dist: %v", err)
	}

	// Create deployment row (simulates a successful build).
	buildID := "b1deadbeef"
	if err := q.CreateDeployment(context.Background(), store.CreateDeploymentParams{
		ID:           buildID,
		SiteID:       testSiteID,
		DeployTarget: "local",
		DeployConfig: "{}",
	}); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	// Create local deploy target.
	dest := filepath.Join(t.TempDir(), "site")
	dh := NewDeployHandler(&config.Config{}, q, sqlDB)
	dr := deployRouter(dh)
	createBody := map[string]any{
		"name": "primary",
		"kind": "local",
		"config": map[string]any{
			"path":       dest,
			"public_url": "https://example.com",
		},
	}
	code, resp := doJSON(t, dr, http.MethodPost, "/api/sites/"+testSiteID+"/deploy-targets", createBody)
	if code != http.StatusCreated {
		t.Fatalf("create target: %d %+v", code, resp)
	}
	targetID, _ := resp["id"].(string)

	// Build handler with primed in-memory state so Deploy can find the dist.
	bh := NewBuildHandler(&config.Config{}, q)
	bh.builds[buildID] = &buildState{
		Status:     "success",
		DistDir:    distDir,
		PagesBuilt: 1,
	}
	br := buildRouter(bh)

	// POST /deploy
	code, resp = postJSON(t, br, "/api/sites/"+testSiteID+"/deploy", map[string]any{
		"build_id":  buildID,
		"target_id": targetID,
	})
	if code != http.StatusOK {
		t.Fatalf("deploy: %d %+v", code, resp)
	}
	if got, _ := resp["deployment_id"].(string); got != buildID {
		t.Fatalf("deployment_id=%q", got)
	}
	if got, _ := resp["deploy_url"].(string); got != "https://example.com" {
		t.Fatalf("deploy_url=%q", got)
	}

	// Dist should be at dest now.
	indexPath := filepath.Join(dest, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("expected file at %q: %v", indexPath, err)
	}
	data, _ := os.ReadFile(indexPath)
	if !strings.Contains(string(data), "ok") {
		t.Fatalf("dist content not copied")
	}

	// DB row should be updated.
	dep, err := q.GetDeploymentByID(context.Background(), buildID)
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if dep.TargetID != targetID {
		t.Fatalf("target_id not stored: got %q", dep.TargetID)
	}
	if dep.DeployUrl != "https://example.com" {
		t.Fatalf("deploy_url not stored: got %q", dep.DeployUrl)
	}
	if dep.DeployedAt == "" {
		t.Fatal("deployed_at not set")
	}
}

func TestBuildHandler_Deploy_RejectsCrossSiteBuild(t *testing.T) {
	sqlDB, q := setupDeployTestDB(t)
	otherSiteID := "bbbbbbbbbbbbbbbbbbbbbbbb"
	seedSiteWithSafeID(t, q, testSiteID)
	seedSiteWithSafeID(t, q, otherSiteID)

	// Build belongs to otherSiteID.
	buildID := "b2deadbeef"
	if err := q.CreateDeployment(context.Background(), store.CreateDeploymentParams{
		ID: buildID, SiteID: otherSiteID, DeployTarget: "local", DeployConfig: "{}",
	}); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	// Target belongs to testSiteID.
	dh := NewDeployHandler(&config.Config{}, q, sqlDB)
	dr := deployRouter(dh)
	dest := filepath.Join(t.TempDir(), "site")
	code, resp := doJSON(t, dr, http.MethodPost, "/api/sites/"+testSiteID+"/deploy-targets", map[string]any{
		"name":   "primary",
		"kind":   "local",
		"config": map[string]any{"path": dest},
	})
	if code != http.StatusCreated {
		t.Fatalf("create target: %d %+v", code, resp)
	}
	targetID, _ := resp["id"].(string)

	bh := NewBuildHandler(&config.Config{}, q)
	br := buildRouter(bh)
	// POST under testSiteID with otherSiteID's build ID.
	code, _ = postJSON(t, br, "/api/sites/"+testSiteID+"/deploy", map[string]any{
		"build_id":  buildID,
		"target_id": targetID,
	})
	if code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-site build, got %d", code)
	}
}

func TestBuildHandler_Deploy_RejectsMissingDistDir(t *testing.T) {
	sqlDB, q := setupDeployTestDB(t)
	seedSiteWithSafeID(t, q, testSiteID)

	buildID := "b3deadbeef"
	if err := q.CreateDeployment(context.Background(), store.CreateDeploymentParams{
		ID: buildID, SiteID: testSiteID, DeployTarget: "local", DeployConfig: "{}",
	}); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	dh := NewDeployHandler(&config.Config{}, q, sqlDB)
	dr := deployRouter(dh)
	dest := filepath.Join(t.TempDir(), "site")
	code, resp := doJSON(t, dr, http.MethodPost, "/api/sites/"+testSiteID+"/deploy-targets", map[string]any{
		"name":   "primary",
		"kind":   "local",
		"config": map[string]any{"path": dest},
	})
	if code != http.StatusCreated {
		t.Fatalf("create target: %d %+v", code, resp)
	}
	targetID, _ := resp["id"].(string)

	// No build state in memory -> handler should refuse.
	bh := NewBuildHandler(&config.Config{}, q)
	br := buildRouter(bh)
	code, body := postJSON(t, br, "/api/sites/"+testSiteID+"/deploy", map[string]any{
		"build_id":  buildID,
		"target_id": targetID,
	})
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %+v", code, body)
	}
}

func TestBuildHandler_Deploy_RejectsBadSiteID(t *testing.T) {
	_, q := setupDeployTestDB(t)
	bh := NewBuildHandler(&config.Config{}, q)
	r := buildRouter(bh)
	code, _ := postJSON(t, r, "/api/sites/not-hex/deploy", map[string]any{
		"build_id":  "x",
		"target_id": "y",
	})
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", code)
	}
}
