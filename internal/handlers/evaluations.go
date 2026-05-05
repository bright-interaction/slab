package handlers

import (
	"net/http"

	"github.com/bright-interaction/slab/internal/config"
	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/store"
)

// EvaluationHandler exposes evaluation results to admin + agent endpoints.
type EvaluationHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewEvaluationHandler(cfg *config.Config, queries *store.Queries) *EvaluationHandler {
	return &EvaluationHandler{cfg: cfg, queries: queries}
}

// ListByBuild returns all category evaluations for a single build.
func (h *EvaluationHandler) ListByBuild(w http.ResponseWriter, r *http.Request) {
	buildID := urlParam(r, "buildID")
	rows, err := h.queries.ListEvaluationsByBuild(r.Context(), buildID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list evaluations")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// ListBySite returns the most recent evaluations across builds for a site.
func (h *EvaluationHandler) ListBySite(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	limit := int64(parseIntDefault(r.URL.Query().Get("limit"), 50))
	rows, err := h.queries.ListEvaluationsBySite(r.Context(), store.ListEvaluationsBySiteParams{
		SiteID: siteID,
		Limit:  limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list evaluations")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// AgentCritique returns the persisted design critique for a build,
// scoped to the calling agent's site. Mirrors RunDesignCritique used
// by the MCP tool so REST + MCP return the same shape. URL: GET
// /api/agent/critique/{buildID} (buildID may be the literal "latest"
// to fall back to the site's most-recent successful build).
func (h *EvaluationHandler) AgentCritique(w http.ResponseWriter, r *http.Request) {
	id := authmw.GetAgent(r)
	if id == nil {
		writeError(w, http.StatusUnauthorized, "agent identity required")
		return
	}
	buildID := urlParam(r, "buildID")
	if buildID == "latest" {
		buildID = ""
	}
	res, err := RunDesignCritique(r.Context(), h.queries, id.SiteID, DesignCritiqueRequest{BuildID: buildID})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
