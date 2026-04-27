// Agent-side site settings API.
//
// Mirrors the admin settings endpoints. Lets an agent read settings (any
// category) and write to safe categories (seo, analytics). Categories
// that affect security posture (security, allowed-scripts, nginx, danger)
// stay admin-only.

package handlers

import (
	"net/http"

	"github.com/bright-interaction/slab/internal/store"
)

// agentWritableSettingsCategories enumerates settings categories an agent
// is allowed to write. Anything not in this set is read-only via the agent
// API (the human admin still controls them in the UI).
//
// We deliberately exclude:
//   - security: HSTS, CSP, frame options change attack surface
//   - allowed-scripts: CSP whitelist additions widen XSS exposure
//   - nginx: server config preview, internal-only
//   - danger: site deletion, destructive
var agentWritableSettingsCategories = map[string]bool{
	"seo":       true,
	"analytics": true,
	"general":   true,
}

// ListSettings returns all settings rows for the agent's site (any category).
//
// GET /api/agent/settings
func (h *AgentHandler) ListSettings(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	rows, err := h.queries.ListSettingsBySite(r.Context(), a.SiteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list settings")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// ListSettingsByCategory returns settings for one category.
//
// GET /api/agent/settings/{category}
func (h *AgentHandler) ListSettingsByCategory(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	category := urlParam(r, "category")
	if category == "" {
		writeError(w, http.StatusBadRequest, "category is required")
		return
	}
	rows, err := h.queries.ListSettingsByCategory(r.Context(), store.ListSettingsByCategoryParams{
		SiteID:   a.SiteID,
		Category: category,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list settings")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// BulkUpsertSettings applies a list of {category, key, value} writes in one
// call. Categories not in agentWritableSettingsCategories are silently
// skipped (the response carries the full settings list afterwards so the
// caller can see which writes landed).
//
// PATCH /api/agent/settings
func (h *AgentHandler) BulkUpsertSettings(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	var req []struct {
		Category string `json:"category"`
		Key      string `json:"key"`
		Value    string `json:"value"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	rejected := []string{}
	for _, s := range req {
		if s.Category == "" || s.Key == "" {
			continue
		}
		if !agentWritableSettingsCategories[s.Category] {
			rejected = append(rejected, s.Category+"."+s.Key)
			continue
		}
		_ = h.queries.UpsertSetting(r.Context(), store.UpsertSettingParams{
			ID:       newID(),
			SiteID:   a.SiteID,
			Category: s.Category,
			Key:      s.Key,
			Value:    s.Value,
		})
	}

	rows, _ := h.queries.ListSettingsBySite(r.Context(), a.SiteID)
	writeJSON(w, http.StatusOK, map[string]any{
		"settings":               rows,
		"rejected_admin_only":    rejected,
		"writable_categories":    keysOfStringBoolMap(agentWritableSettingsCategories),
	})
}

func keysOfStringBoolMap(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
