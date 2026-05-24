package mcp

import (
	"testing"
)

// TestWireToolsListIncludesAppsTools confirms the Sprint 4 slice A
// apps tools register in tools/list. Two tools: list_apps_marketplace
// (read the cross-tenant catalogue) + list_installed_apps (read the
// per-site install ledger). Slice B adds a third tool (use_app) that
// proxies upstream MCP server tool calls.
func TestWireToolsListIncludesAppsTools(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/list", map[string]any{})
	result := resp.Result.(map[string]any)
	tools := result["tools"].([]any)
	names := map[string]bool{}
	for _, ti := range tools {
		tool := ti.(map[string]any)
		names[tool["name"].(string)] = true
	}
	for _, name := range []string{"list_apps_marketplace", "list_installed_apps"} {
		if !names[name] {
			t.Errorf("tools/list missing apps tool %q", name)
		}
	}
}
