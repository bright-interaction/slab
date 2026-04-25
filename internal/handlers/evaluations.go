package handlers

import (
	"net/http"

	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/store"
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
