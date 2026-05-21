package mcp

import (
	"strings"
	"testing"
)

// TestWireToolsListIncludesArchetypeTools confirms the gap-6 surface is
// registered: set_page_archetype + get_page_archetype.
func TestWireToolsListIncludesArchetypeTools(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/list", map[string]any{})
	result := resp.Result.(map[string]any)
	tools := result["tools"].([]any)
	names := map[string]bool{}
	for _, ti := range tools {
		names[ti.(map[string]any)["name"].(string)] = true
	}
	for _, name := range []string{"set_page_archetype", "get_page_archetype"} {
		if !names[name] {
			t.Errorf("tools/list missing archetype tool %q", name)
		}
	}
}

// TestWireSetPageArchetype_RejectsInvalidName belts the validation:
// only the 4 curated archetypes are accepted.
func TestWireSetPageArchetype_RejectsInvalidName(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/call", map[string]any{
		"name": "set_page_archetype",
		"arguments": map[string]any{
			"page_slug": "/index",
			"archetype": "make-up-a-vibe",
		},
	})
	result := resp.Result.(map[string]any)
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected isError for invalid archetype; got %v", result)
	}
	first := result["content"].([]any)[0].(map[string]any)
	body, _ := first["text"].(string)
	if !strings.Contains(body, "archetype must be one of") {
		t.Errorf("expected validation message; got %q", body)
	}
}

// TestWireLintBlock_ArchetypeAware confirms passing archetype to
// lint_block fires the archetype_drift check.
func TestWireLintBlock_ArchetypeAware(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/call", map[string]any{
		"name": "lint_block",
		"arguments": map[string]any{
			"block_type": "hero",
			"archetype":  "audit-receipt",
			"data": map[string]any{
				"hero_graphic": "mesh",
				"headline":     "Own it",
			},
		},
	})
	result := resp.Result.(map[string]any)
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("lint_block returned isError=true: %v", result["content"])
	}
	first := result["content"].([]any)[0].(map[string]any)
	body, _ := first["text"].(string)
	if !strings.Contains(body, "archetype_drift") {
		t.Errorf("expected archetype_drift warning; got %s", body[:min(500, len(body))])
	}
}
