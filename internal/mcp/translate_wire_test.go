package mcp

import (
	"testing"
)

// TestWireToolsListIncludesTranslateTool confirms the Sprint 3 slice B
// translate_entity tool registers in tools/list. One tool, dual-mode
// (read vs write determined by the translations parameter).
func TestWireToolsListIncludesTranslateTool(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/list", map[string]any{})
	result := resp.Result.(map[string]any)
	tools := result["tools"].([]any)
	for _, ti := range tools {
		tool := ti.(map[string]any)
		if tool["name"].(string) == "translate_entity" {
			return
		}
	}
	t.Errorf("tools/list missing translate_entity tool")
}
