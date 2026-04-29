package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/brightinteraction/atomicsite/internal/builder"
	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/cookieproof"
	"github.com/brightinteraction/atomicsite/internal/deploy"
	"github.com/brightinteraction/atomicsite/internal/eval"
	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// safeSiteIDPattern matches the format newID() produces (24-char hex). Blocks
// any path-traversal attempt via the URL param before we use it as a directory
// name in {DataDir}/workspaces/{siteID}/.
var safeSiteIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{24}$`)

func isSafeSiteID(s string) bool { return safeSiteIDPattern.MatchString(s) }

// BuildHandler handles build trigger and status endpoints.
type BuildHandler struct {
	cfg     *config.Config
	queries *store.Queries

	mu     sync.Mutex
	builds map[string]*buildState // deploymentID -> state
}

type buildState struct {
	Status     string `json:"status"`
	BuildLog   string `json:"build_log"`
	PagesBuilt int    `json:"pages_built"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error"`
	DistDir    string `json:"dist_dir"`
}

func NewBuildHandler(cfg *config.Config, queries *store.Queries) *BuildHandler {
	return &BuildHandler{
		cfg:     cfg,
		queries: queries,
		builds:  make(map[string]*buildState),
	}
}

// provisionCookieProof runs the best-effort CookieProof EnsureDomain step
// before a build. Failure is logged + ignored: the build still ships even
// if CookieProof is unreachable. Skips entirely if analytics is off, the
// site has no domain yet, or the admin token isn't configured.
func (h *BuildHandler) provisionCookieProof(ctx context.Context, siteID string) {
	if h.cfg == nil || h.cfg.CookieProofAdminToken == "" || h.cfg.CookieProofAPIBase == "" {
		return
	}
	site, err := h.queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return
	}
	domain := strings.TrimSpace(site.CookieproofDomain)
	if domain == "" {
		domain = strings.TrimSpace(site.Domain)
	}
	if domain == "" {
		return
	}

	// Gated on the same setting that drives the layout snippet, so we don't
	// register every site with CookieProof unless analytics is actually on.
	settings, _ := h.queries.ListSettingsBySite(ctx, siteID)
	enabled := false
	for _, s := range settings {
		if s.Category == "analytics" && s.Key == "cookieproof_enabled" {
			v := strings.ToLower(strings.TrimSpace(s.Value))
			enabled = v == "1" || v == "true" || v == "yes" || v == "on"
			break
		}
	}
	if !enabled {
		return
	}

	cp := cookieproof.New(h.cfg.CookieProofAPIBase, h.cfg.CookieProofAdminToken)
	if err := cp.EnsureDomain(ctx, domain); err != nil {
		slog.Warn("cookieproof ensure domain failed; build will continue",
			"site_id", siteID,
			"domain", domain,
			"err", err,
		)
	}
}

// TriggerBuild starts an async build for the agent's site.
func (h *BuildHandler) TriggerBuild(w http.ResponseWriter, r *http.Request) {
	a := authmw.GetAgent(r)
	if a == nil {
		writeError(w, http.StatusUnauthorized, "Agent not authenticated")
		return
	}

	// Create a deployment record
	deployID := newID()
	err := h.queries.CreateDeployment(r.Context(), store.CreateDeploymentParams{
		ID:           deployID,
		SiteID:       a.SiteID,
		DeployTarget: "local",
		DeployConfig: "{}",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create deployment record")
		return
	}

	// Track build state
	h.mu.Lock()
	h.builds[deployID] = &buildState{Status: "building"}
	h.mu.Unlock()

	// Run build async (use background context -- request context cancels when client disconnects)
	siteID := a.SiteID
	go func() {
		bgCtx := context.Background()
		h.provisionCookieProof(bgCtx, siteID)
		result := builder.Build(bgCtx, h.queries, siteID, h.cfg.DataDir)

		// Run evaluation against the built dist/ if compile succeeded.
		if result.Success && result.DistDir != "" {
			if _, err := eval.Run(bgCtx, h.queries, siteID, deployID, result.DistDir); err != nil {
				// Eval failures are non-fatal; build succeeded.
				result.BuildLog += "\n=== eval ===\nfailed: " + err.Error() + "\n"
			}
		}

		h.mu.Lock()
		state := h.builds[deployID]
		state.BuildLog = result.BuildLog
		state.PagesBuilt = result.PagesBuilt
		state.DurationMs = result.DurationMs
		state.DistDir = result.DistDir
		if result.Success {
			state.Status = "success"
		} else {
			state.Status = "failed"
			state.Error = result.Error
		}
		h.mu.Unlock()

		// Update DB record
		status := "success"
		if !result.Success {
			status = "failed"
		}
		_ = h.queries.UpdateDeploymentStatus(bgCtx, store.UpdateDeploymentStatusParams{
			ID:         deployID,
			Status:     status,
			BuildLog:   result.BuildLog,
			PagesBuilt: int64(result.PagesBuilt),
			DurationMs: result.DurationMs,
			Error:      result.Error,
		})

		_ = h.queries.UpdateSiteBuildStatus(bgCtx, store.UpdateSiteBuildStatusParams{
			ID:              siteID,
			LastBuildStatus: status,
			LastBuildError:  result.Error,
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"build_id": deployID,
		"status":   "building",
		"message":  "Build started. Poll GET /api/agent/build/" + deployID + "/status for progress.",
	})
}

// StartBuild creates a deployment record + kicks off an async build for
// the given site. Returns the build_id (= deployment id). Non-handler
// entry point used by the MCP server's trigger_build tool so it doesn't
// have to duplicate the deployment + goroutine + cookieproof provisioning
// logic. Mirrors TriggerBuild's body without the http.ResponseWriter.
func (h *BuildHandler) StartBuild(ctx context.Context, siteID string) (string, error) {
	deployID := newID()
	if err := h.queries.CreateDeployment(ctx, store.CreateDeploymentParams{
		ID:           deployID,
		SiteID:       siteID,
		DeployTarget: "local",
		DeployConfig: "{}",
	}); err != nil {
		return "", err
	}

	h.mu.Lock()
	h.builds[deployID] = &buildState{Status: "building"}
	h.mu.Unlock()

	go func() {
		bgCtx := context.Background()
		h.provisionCookieProof(bgCtx, siteID)
		result := builder.Build(bgCtx, h.queries, siteID, h.cfg.DataDir)
		if result.Success && result.DistDir != "" {
			if _, err := eval.Run(bgCtx, h.queries, siteID, deployID, result.DistDir); err != nil {
				result.BuildLog += "\n=== eval ===\nfailed: " + err.Error() + "\n"
			}
			// Auto-deploy to the site's default deploy target so trigger_build
			// is a true publish verb, not just "build to workspace + eval".
			if targetID, deployURL, deployErr := h.autoDeployDefault(bgCtx, siteID, result.DistDir); deployErr != nil {
				result.BuildLog += "\n=== deploy ===\nfailed: " + deployErr.Error() + "\n"
			} else if targetID != "" {
				result.BuildLog += "\n=== deploy ===\npublished to " + deployURL + "\n"
				_ = h.queries.UpdateDeploymentDeployed(bgCtx, store.UpdateDeploymentDeployedParams{
					TargetID: targetID, DeployUrl: deployURL, ID: deployID,
				})
			}
		}
		h.mu.Lock()
		st := h.builds[deployID]
		st.BuildLog = result.BuildLog
		st.PagesBuilt = result.PagesBuilt
		st.DurationMs = result.DurationMs
		st.DistDir = result.DistDir
		if result.Success {
			st.Status = "success"
		} else {
			st.Status = "failed"
			st.Error = result.Error
		}
		h.mu.Unlock()

		status := "success"
		if !result.Success {
			status = "failed"
		}
		_ = h.queries.UpdateDeploymentStatus(bgCtx, store.UpdateDeploymentStatusParams{
			ID: deployID, Status: status, BuildLog: result.BuildLog,
			PagesBuilt: int64(result.PagesBuilt), DurationMs: result.DurationMs, Error: result.Error,
		})
		_ = h.queries.UpdateSiteBuildStatus(bgCtx, store.UpdateSiteBuildStatusParams{
			ID: siteID, LastBuildStatus: status, LastBuildError: result.Error,
		})
	}()

	return deployID, nil
}

// autoDeployDefault publishes distDir to the site's default deploy target.
// Returns ("", "", nil) when no targets are configured (silent no-op so
// brand-new sites that haven't picked a target yet still get a clean
// build response). Returns (targetID, deployURL, nil) on success.
func (h *BuildHandler) autoDeployDefault(ctx context.Context, siteID, distDir string) (string, string, error) {
	targets, err := h.queries.ListDeployTargetsBySite(ctx, siteID)
	if err != nil {
		return "", "", fmt.Errorf("list deploy targets: %w", err)
	}
	if len(targets) == 0 {
		return "", "", nil
	}
	t := targets[0] // ordered by is_default DESC, name ASC
	cfg := map[string]any{}
	if strings.TrimSpace(t.ConfigJson) != "" {
		_ = json.Unmarshal([]byte(t.ConfigJson), &cfg)
	}
	target := deploy.Target{
		ID:     t.ID,
		SiteID: t.SiteID,
		Name:   t.Name,
		Kind:   t.Kind,
		Config: cfg,
	}
	deployer, err := deploy.New(target.Kind)
	if err != nil {
		return "", "", fmt.Errorf("deployer: %w", err)
	}
	res, err := deployer.Deploy(ctx, distDir, target)
	if err != nil {
		return "", "", fmt.Errorf("deploy: %w", err)
	}
	return t.ID, res.URL, nil
}

// GetBuildState returns the cross-source build state (in-memory if active,
// DB row if persisted). siteID is checked so MCP can't read across sites.
// Used by the MCP get_build_status + get_evaluation tools to reuse the
// same logic the BuildStatus admin endpoint exercises.
func (h *BuildHandler) GetBuildState(ctx context.Context, buildID, siteID string) (any, error) {
	deploy, err := h.queries.GetDeploymentByID(ctx, buildID)
	if err != nil {
		return nil, fmt.Errorf("build not found: %s", buildID)
	}
	if deploy.SiteID != siteID {
		return nil, fmt.Errorf("build does not belong to this site")
	}
	h.mu.Lock()
	st, ok := h.builds[buildID]
	h.mu.Unlock()
	if ok {
		return map[string]any{
			"build_id":    buildID,
			"status":      st.Status,
			"build_log":   st.BuildLog,
			"pages_built": st.PagesBuilt,
			"duration_ms": st.DurationMs,
			"error":       st.Error,
			"dist_dir":    st.DistDir,
			"created_at":  deploy.CreatedAt,
		}, nil
	}
	return map[string]any{
		"build_id":    buildID,
		"status":      deploy.Status,
		"build_log":   deploy.BuildLog,
		"pages_built": deploy.PagesBuilt,
		"duration_ms": deploy.DurationMs,
		"error":       deploy.Error,
		"created_at":  deploy.CreatedAt,
	}, nil
}

// BuildStatus returns the current status of a build.
func (h *BuildHandler) BuildStatus(w http.ResponseWriter, r *http.Request) {
	a := authmw.GetAgent(r)
	if a == nil {
		writeError(w, http.StatusUnauthorized, "Agent not authenticated")
		return
	}

	buildID := urlParam(r, "buildID")

	// Check in-memory state first (for active builds)
	h.mu.Lock()
	state, ok := h.builds[buildID]
	h.mu.Unlock()

	if ok {
		writeJSON(w, http.StatusOK, state)
		return
	}

	// Fall back to DB
	deploy, err := h.queries.GetDeploymentByID(r.Context(), buildID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Build not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      deploy.Status,
		"build_log":   deploy.BuildLog,
		"pages_built": deploy.PagesBuilt,
		"duration_ms": deploy.DurationMs,
		"error":       deploy.Error,
		"dist_dir":    "",
	})
}

// BuildStatusAdmin returns the status of a build for a site via admin auth.
// Mirrors BuildStatus but scoped to siteID to prevent cross-site ID guessing.
func (h *BuildHandler) BuildStatusAdmin(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if !isSafeSiteID(siteID) {
		writeError(w, http.StatusBadRequest, "Invalid site ID")
		return
	}
	buildID := urlParam(r, "buildID")
	if buildID == "" {
		writeError(w, http.StatusBadRequest, "Invalid build ID")
		return
	}

	// Look up the deployment so we can verify it belongs to this site.
	deploy, err := h.queries.GetDeploymentByID(r.Context(), buildID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Build not found")
		return
	}
	if deploy.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Build not found")
		return
	}

	// Prefer in-memory state for active builds (has dist_dir + live build_log).
	h.mu.Lock()
	state, ok := h.builds[buildID]
	h.mu.Unlock()

	if ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      state.Status,
			"build_log":   state.BuildLog,
			"pages_built": state.PagesBuilt,
			"duration_ms": state.DurationMs,
			"error":       state.Error,
			"dist_dir":    state.DistDir,
			"created_at":  deploy.CreatedAt,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      deploy.Status,
		"build_log":   deploy.BuildLog,
		"pages_built": deploy.PagesBuilt,
		"duration_ms": deploy.DurationMs,
		"error":       deploy.Error,
		"dist_dir":    "",
		"created_at":  deploy.CreatedAt,
	})
}

// deployRequest is the body for POST /api/sites/{siteID}/deploy.
type deployRequest struct {
	BuildID  string `json:"build_id"`
	TargetID string `json:"target_id"`
}

// Deploy ships a previously-built deployment to a configured deploy_target.
// Looks up the deployment row (must belong to siteID), looks up the target
// (must belong to siteID), then runs the kind-specific Deployer against the
// build's dist_dir. On success the deployment row gets target_id, deploy_url,
// deployed_at filled in.
func (h *BuildHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if !isSafeSiteID(siteID) {
		writeError(w, http.StatusBadRequest, "Invalid site ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var req deployRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.BuildID = strings.TrimSpace(req.BuildID)
	req.TargetID = strings.TrimSpace(req.TargetID)
	if req.BuildID == "" {
		writeError(w, http.StatusBadRequest, "build_id is required")
		return
	}
	if req.TargetID == "" {
		writeError(w, http.StatusBadRequest, "target_id is required")
		return
	}

	deploymentRow, err := h.queries.GetDeploymentByID(r.Context(), req.BuildID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Build not found")
		return
	}
	if deploymentRow.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Build not found")
		return
	}

	targetRow, err := h.queries.GetDeployTarget(r.Context(), req.TargetID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Deploy target not found")
		return
	}
	if targetRow.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Deploy target not found")
		return
	}

	// Pull the build's dist_dir from the in-memory build state. We deliberately
	// do not persist dist paths to DB (they're ephemeral workspace artifacts),
	// so a deploy after server restart will fail loudly here.
	h.mu.Lock()
	state, ok := h.builds[req.BuildID]
	h.mu.Unlock()
	if !ok || state == nil || state.DistDir == "" {
		writeError(w, http.StatusBadRequest, "Build dist_dir not available; trigger a fresh build before deploying")
		return
	}
	if state.Status != "success" {
		writeError(w, http.StatusBadRequest, "Build did not succeed; cannot deploy")
		return
	}

	cfg := map[string]any{}
	if strings.TrimSpace(targetRow.ConfigJson) != "" {
		_ = json.Unmarshal([]byte(targetRow.ConfigJson), &cfg)
	}
	target := deploy.Target{
		ID:     targetRow.ID,
		SiteID: targetRow.SiteID,
		Name:   targetRow.Name,
		Kind:   targetRow.Kind,
		Config: cfg,
	}

	deployer, err := deploy.New(target.Kind)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := deployer.Deploy(r.Context(), state.DistDir, target)
	if err != nil {
		slog.Error("deploy failed",
			"site_id", siteID,
			"build_id", req.BuildID,
			"target_id", req.TargetID,
			"kind", target.Kind,
			"err", err,
		)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	if err := h.queries.UpdateDeploymentDeployed(r.Context(), store.UpdateDeploymentDeployedParams{
		TargetID:  req.TargetID,
		DeployUrl: result.URL,
		ID:        req.BuildID,
	}); err != nil {
		slog.Error("update deployment deployed",
			"build_id", req.BuildID,
			"err", err,
		)
		writeError(w, http.StatusInternalServerError, "Deploy succeeded but DB update failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"deployment_id": req.BuildID,
		"deploy_url":    result.URL,
		"deployed_at":   result.DeployedAt,
		"size_bytes":    result.SizeBytes,
		"file_count":    result.FileCount,
	})
}

// TriggerBuildAdmin starts a build for a site via admin auth.
func (h *BuildHandler) TriggerBuildAdmin(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if !isSafeSiteID(siteID) {
		writeError(w, http.StatusBadRequest, "Invalid site ID")
		return
	}

	deployID := newID()
	err := h.queries.CreateDeployment(r.Context(), store.CreateDeploymentParams{
		ID:           deployID,
		SiteID:       siteID,
		DeployTarget: "local",
		DeployConfig: "{}",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create deployment record")
		return
	}

	h.mu.Lock()
	h.builds[deployID] = &buildState{Status: "building"}
	h.mu.Unlock()

	adminSiteID := siteID
	go func() {
		bgCtx := context.Background()
		h.provisionCookieProof(bgCtx, adminSiteID)
		result := builder.Build(bgCtx, h.queries, adminSiteID, h.cfg.DataDir)
		if result.Success && result.DistDir != "" {
			if _, err := eval.Run(bgCtx, h.queries, adminSiteID, deployID, result.DistDir); err != nil {
				result.BuildLog += "\n=== eval ===\nfailed: " + err.Error() + "\n"
			}
		}

		h.mu.Lock()
		state := h.builds[deployID]
		state.BuildLog = result.BuildLog
		state.PagesBuilt = result.PagesBuilt
		state.DurationMs = result.DurationMs
		state.DistDir = result.DistDir
		if result.Success {
			state.Status = "success"
		} else {
			state.Status = "failed"
			state.Error = result.Error
		}
		h.mu.Unlock()

		status := "success"
		if !result.Success {
			status = "failed"
		}
		_ = h.queries.UpdateDeploymentStatus(bgCtx, store.UpdateDeploymentStatusParams{
			ID:         deployID,
			Status:     status,
			BuildLog:   result.BuildLog,
			PagesBuilt: int64(result.PagesBuilt),
			DurationMs: result.DurationMs,
			Error:      result.Error,
		})
		_ = h.queries.UpdateSiteBuildStatus(bgCtx, store.UpdateSiteBuildStatusParams{
			ID:              adminSiteID,
			LastBuildStatus: status,
			LastBuildError:  result.Error,
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"build_id": deployID,
		"status":   "building",
	})
}
