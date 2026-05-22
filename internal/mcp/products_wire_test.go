package mcp

import (
	"testing"
)

// TestWireToolsListIncludesCatalogTools confirms the Sprint 2 slice A
// catalog + discount-code surfaces are registered. 9 product/variant
// tools + 4 discount-code tools = 13 total.
func TestWireToolsListIncludesCatalogTools(t *testing.T) {
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
		"list_products",
		"get_product",
		"create_product",
		"update_product",
		"delete_product",
		"create_variant",
		"update_variant",
		"delete_variant",
		"set_inventory",
		"list_discount_codes",
		"create_discount_code",
		"update_discount_code",
		"delete_discount_code",
	} {
		if !names[name] {
			t.Errorf("tools/list missing catalog tool %q", name)
		}
	}
}
