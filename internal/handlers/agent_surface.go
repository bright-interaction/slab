// Admin-only inventory of the MCP agent surface. The admin SPA's
// Settings -> Agent page reads this to render what an agent connecting
// via MCP can see and do (tools, resources, prompts, curriculum). No
// agent key required: the route lives behind the regular admin session
// + site-access middleware.

package handlers

import (
	"net/http"
)

// AgentSurface returns the live MCP surface for the admin Settings ->
// Agent page. Empty payload when SetSurfaceFn was not wired (tests or
// MCP-disabled builds); the SPA renders a clean "MCP not configured"
// state in that case.
//
// GET /api/sites/{siteID}/agent-surface (RequireAdmin + RequireSiteAccess)
func (h *AgentHandler) AgentSurface(w http.ResponseWriter, r *http.Request) {
	if h.surfaceFn == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"available": false,
			"reason":    "MCP server not configured on this build (handler surfaceFn unset).",
		})
		return
	}
	payload := h.surfaceFn()
	writeJSON(w, http.StatusOK, payload)
}
