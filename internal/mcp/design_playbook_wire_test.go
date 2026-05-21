package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestWireResourceReadDesignPlaybook drives the resources/read JSON-RPC
// path and asserts the playbook payload carries the rubric sections the
// agent needs before authoring blocks. Closes the gap where the Inspector
// graded against rules the agent could not read.
func TestWireResourceReadDesignPlaybook(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "resources/read", map[string]any{
		"uri": "atomicsite://meta/design-playbook",
	})

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("resources/read response shape: %#v", resp.Result)
	}
	contents, ok := result["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("contents missing or empty: %#v", result)
	}
	first := contents[0].(map[string]any)
	body, _ := first["text"].(string)
	if body == "" {
		t.Fatalf("playbook body empty")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("playbook body is not JSON: %v\nbody: %s", err, body)
	}

	mustHaveSection := []string{
		"stack",
		"principles",
		"page_archetypes",
		"anti_patterns",
		"vibe_archetypes",
		"materiality",
		"content_authenticity",
		"motion",
		"copy_voice",
		"fonts",
		"hero_graphics",
		"audit_checklist",
		"one_shot_recipe",
	}
	for _, key := range mustHaveSection {
		if _, ok := payload[key]; !ok {
			t.Errorf("playbook missing section %q (keys present: %v)", key, mapKeys(payload))
		}
	}

	// Sanity-check the principles section has actual content, not an empty
	// shell. Agents pattern-match on these rules verbatim.
	principles, _ := payload["principles"].([]any)
	if len(principles) < 5 {
		t.Errorf("expected at least 5 design principles in playbook; got %d", len(principles))
	}
}

// TestWireToolCallGetDesignPlaybook drives the tools/call JSON-RPC path
// and asserts the get_design_playbook tool returns the same payload as
// the resource. Same shape so clients can pick either surface.
func TestWireToolCallGetDesignPlaybook(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/call", map[string]any{
		"name":      "get_design_playbook",
		"arguments": map[string]any{},
	})
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("tools/call response shape: %#v", resp.Result)
	}
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("get_design_playbook returned isError=true: %v", result["content"])
	}
	contentArr, _ := result["content"].([]any)
	if len(contentArr) == 0 {
		t.Fatalf("get_design_playbook returned empty content")
	}
	first := contentArr[0].(map[string]any)
	body, _ := first["text"].(string)
	if !strings.Contains(body, `"principles"`) {
		t.Errorf("playbook body missing principles section\nbody: %s", body[:min(400, len(body))])
	}
	if !strings.Contains(body, `"anti_patterns"`) {
		t.Errorf("playbook body missing anti_patterns section")
	}
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
