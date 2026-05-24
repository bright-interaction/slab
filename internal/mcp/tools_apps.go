package mcp

import (
	"context"
	"encoding/json"
	"errors"

	authmw "github.com/bright-interaction/slab/internal/middleware"
)

// registerAppsTools wires the Sprint 4 slice A read-only MCP tools.
// Two tools: list_apps_marketplace (cross-tenant catalogue) +
// list_installed_apps (per-site installed list with credentials_set
// metadata, never the values themselves). Slice B adds a third
// tool (use_app) that proxies upstream MCP server tool calls
// through atomicsite using the stored credentials.
//
// Both tools are RequiresWrite=false: they read state, never mutate.
// Install + revoke flow is operator-only via the REST + admin UI in
// slice A; agents can read what's installed but not change it.
// Slice B will add an opt-in agent-write capability behind an
// explicit per-site flag.
func (s *Server) registerAppsTools() {
	register := func(t Tool) { s.tools[t.Name] = t }

	register(Tool{
		Name: "list_apps_marketplace",
		Description: "Lists every active app in the cross-tenant Atomicsite Apps marketplace. Returns app_id, slug, name, description, category, publisher, version, icon_url, docs_url, plus the credential field schema each app needs at install. Use this to discover what third-party integrations are available before recommending an install to the human. Read-only: install happens via the admin UI in slice A; slice B will gate agent-driven installs behind a per-site flag.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.apps == nil {
				return "", errors.New("list_apps_marketplace not configured on this server (apps handler unset)")
			}
			out, err := s.apps.ListAgentMarketplace(ctx)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"apps": out, "count": len(out)}), nil
		},
	})

	register(Tool{
		Name: "list_installed_apps",
		Description: "Lists the apps installed on this site. Returns app_id, slug, name, category, status (active|revoked), credentials_set (the FIELD KEYS that have a value stored; values themselves are never exposed via MCP), and last_used_at. Use this to plan multi-step flows: e.g. 'I see Stripe is installed with secret_key set, so I can create a checkout link via the use_app proxy in slice B'.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.apps == nil {
				return "", errors.New("list_installed_apps not configured on this server (apps handler unset)")
			}
			out, err := s.apps.ListAgentInstalls(ctx, agent.SiteID)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"installs": out, "count": len(out)}), nil
		},
	})
}
