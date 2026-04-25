package handlers

import (
	"context"
	"net/http"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

// SettingsHandler handles per-site settings CRUD.
type SettingsHandler struct {
	cfg     *config.Config
	queries *store.Queries
	// onAnalyticsChange is invoked after any write that touches the analytics
	// category, so the analytics Manager can rescan and (re)spawn parsers for
	// sites that just had tracking flipped on.
	onAnalyticsChange func(context.Context)
}

func NewSettingsHandler(cfg *config.Config, queries *store.Queries) *SettingsHandler {
	return &SettingsHandler{cfg: cfg, queries: queries}
}

// OnAnalyticsChange registers a callback invoked after analytics-category writes.
func (h *SettingsHandler) OnAnalyticsChange(fn func(context.Context)) {
	h.onAnalyticsChange = fn
}

func (h *SettingsHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	settings, err := h.queries.ListSettingsBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list settings")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (h *SettingsHandler) ListByCategory(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	category := urlParam(r, "category")
	settings, err := h.queries.ListSettingsByCategory(r.Context(), store.ListSettingsByCategoryParams{
		SiteID:   siteID,
		Category: category,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list settings")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (h *SettingsHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	var req struct {
		Category string `json:"category"`
		Key      string `json:"key"`
		Value    string `json:"value"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Category == "" || req.Key == "" {
		writeError(w, http.StatusBadRequest, "category and key are required")
		return
	}

	id := newID()
	err := h.queries.UpsertSetting(r.Context(), store.UpsertSettingParams{
		ID:       id,
		SiteID:   siteID,
		Category: req.Category,
		Key:      req.Key,
		Value:    req.Value,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save setting")
		return
	}

	if req.Category == "analytics" && h.onAnalyticsChange != nil {
		h.onAnalyticsChange(r.Context())
	}

	setting, _ := h.queries.GetSetting(r.Context(), store.GetSettingParams{
		SiteID:   siteID,
		Category: req.Category,
		Key:      req.Key,
	})
	writeJSON(w, http.StatusOK, setting)
}

func (h *SettingsHandler) BulkUpsert(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	var req []struct {
		Category string `json:"category"`
		Key      string `json:"key"`
		Value    string `json:"value"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	touchedAnalytics := false
	for _, s := range req {
		if s.Category == "" || s.Key == "" {
			continue
		}
		if s.Category == "analytics" {
			touchedAnalytics = true
		}
		_ = h.queries.UpsertSetting(r.Context(), store.UpsertSettingParams{
			ID:       newID(),
			SiteID:   siteID,
			Category: s.Category,
			Key:      s.Key,
			Value:    s.Value,
		})
	}

	if touchedAnalytics && h.onAnalyticsChange != nil {
		h.onAnalyticsChange(r.Context())
	}

	settings, _ := h.queries.ListSettingsBySite(r.Context(), siteID)
	writeJSON(w, http.StatusOK, settings)
}

func (h *SettingsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "settingID")
	if err := h.queries.DeleteSetting(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete setting")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
