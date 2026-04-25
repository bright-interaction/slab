package handlers

import (
	"net/http"
	"strconv"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

// AnalyticsHandler exposes admin-side reads over the visit_events table populated
// by the per-site nginx log tailer (internal/analytics).
type AnalyticsHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewAnalyticsHandler(cfg *config.Config, queries *store.Queries) *AnalyticsHandler {
	return &AnalyticsHandler{cfg: cfg, queries: queries}
}

// VisitEvents returns the most recent visit_events for a site. Newest first,
// capped at 1000 rows. ?limit=N (default 100) trims the response.
func (h *AnalyticsHandler) VisitEvents(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	limit := int64(100)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = n
		}
	}
	rows, err := h.queries.ListVisitsBySite(r.Context(), store.ListVisitsBySiteParams{
		SiteID: siteID,
		Limit:  limit,
		Offset: 0,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list visit events")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}
