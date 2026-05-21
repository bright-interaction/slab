package mcp

import (
	"strings"
	"testing"
)

// TestWireToolsListIncludesRevisionTools confirms the three Sprint 1
// surfaces are registered: list_revisions, get_revision,
// restore_revision. Wiring regression for the WP/Webflow roadmap
// version-history feature.
func TestWireToolsListIncludesRevisionTools(t *testing.T) {
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
		"list_revisions",
		"get_revision",
		"restore_revision",
	} {
		if !names[name] {
			t.Errorf("tools/list missing revision tool %q", name)
		}
	}
}

// TestWireRestoreRevision_SafeFailWhenUnconfigured locks the
// boundary that prevents restore_revision from panicking on a nil
// handler. Mirrors the same fail-soft pattern as
// request_clarification.
func TestWireRestoreRevision_SafeFailWhenUnconfigured(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/call", map[string]any{
		"name": "restore_revision",
		"arguments": map[string]any{
			"entity_type": "page",
			"entity_id":   "any-page-id",
			"version":     1,
		},
	})
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("tools/call shape: %#v", resp.Result)
	}
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected isError=true when revisions handler unset; got %#v", result)
	}
	content := result["content"].([]any)
	first := content[0].(map[string]any)
	text := first["text"].(string)
	if !strings.Contains(text, "not configured") {
		t.Errorf("error text = %q; want hint about 'not configured'", text)
	}
}
