package handlers

import (
	"context"
	"net/http"
	"sync"

	"github.com/bright-interaction/slab/internal/builder"
	"github.com/bright-interaction/slab/internal/config"
	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/store"
)

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
		result := builder.Build(context.Background(), h.queries, siteID, h.cfg.DataDir)

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
		bgCtx := context.Background()
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

// TriggerBuildAdmin starts a build for a site via admin auth.
func (h *BuildHandler) TriggerBuildAdmin(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")

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
		result := builder.Build(context.Background(), h.queries, adminSiteID, h.cfg.DataDir)

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

		bgCtx := context.Background()
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
