package handlers

import (
	"context"
	"net/http"
	"regexp"
	"sync"

	"github.com/bright-interaction/slab/internal/builder"
	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/eval"
	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/store"
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
