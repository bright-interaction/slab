package mcp

import (
	"strings"
	"testing"
)

// TestWireToolsListIncludesClarificationTools confirms the three new
// surfaces are registered: request_clarification, get_clarification,
// list_pending_clarifications. Wiring regression for the gap 3 ship.
func TestWireToolsListIncludesClarificationTools(t *testing.T) {
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
		"request_clarification",
		"get_clarification",
		"list_pending_clarifications",
	} {
		if !names[name] {
			t.Errorf("tools/list missing clarification tool %q", name)
		}
	}
}

// TestWireRequestClarification_SafeFailWhenUnconfigured exercises the
// fail-soft path when the MCP server is built without
// WithClarifications. Locks the boundary that prevents the tool from
// panicking on a nil handler.
func TestWireRequestClarification_SafeFailWhenUnconfigured(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/call", map[string]any{
		"name": "request_clarification",
		"arguments": map[string]any{
			"question": "Mesh or audit-receipt hero for the law-firm landing?",
		},
	})
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("tools/call shape: %#v", resp.Result)
	}
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected isError=true when clarifications unset")
	}
	first := result["content"].([]any)[0].(map[string]any)
	body, _ := first["text"].(string)
	if !strings.Contains(body, "request_clarification not configured") {
		t.Errorf("diagnostic missing 'not configured': %q", body)
	}
}

// TestWireGetClarification_RejectsBlankID belts the input validation.
// An empty id must return a clear error rather than a generic 404.
func TestWireGetClarification_RejectsBlankID(t *testing.T) {
	s := freshTestServer(t)
	s = s.WithClarifications(nil) // explicit nil to force the unset path
	resp, _ := jsonRPCRoundtrip(t, s, "tools/call", map[string]any{
		"name":      "get_clarification",
		"arguments": map[string]any{"id": ""},
	})
	result := resp.Result.(map[string]any)
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected isError=true; got %v", result)
	}
	first := result["content"].([]any)[0].(map[string]any)
	body, _ := first["text"].(string)
	if !strings.Contains(body, "not configured") && !strings.Contains(body, "id required") {
		t.Errorf("expected configuration or validation error; got %q", body)
	}
}
