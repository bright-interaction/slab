package mcp

import (
	"testing"
)

// TestWireToolsListIncludesOrderTools confirms the Sprint 2 slice B
// order tools register in tools/list. 4 tools: list_orders, get_order,
// update_order_status, refund_order.
func TestWireToolsListIncludesOrderTools(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/list", map[string]any{})
	result := resp.Result.(map[string]any)
	tools := result["tools"].([]any)
	names := map[string]bool{}
	for _, ti := range tools {
		tool := ti.(map[string]any)
		names[tool["name"].(string)] = true
	}
	for _, name := range []string{
		"list_orders", "get_order", "update_order_status", "refund_order",
	} {
		if !names[name] {
			t.Errorf("tools/list missing order tool %q", name)
		}
	}
}
