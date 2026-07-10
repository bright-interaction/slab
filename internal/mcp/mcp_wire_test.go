package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authmw "github.com/bright-interaction/atomicsite/internal/middleware"
)

// withTestAgent attaches a synthetic AgentIdentity to the request
// context, mirroring what AgentAuthMiddleware does in production.
// Used to exercise the JSON-RPC handler end-to-end without bringing
// up the full server.
func withTestAgent(req *http.Request) *http.Request {
	identity := &authmw.AgentIdentity{
		KeyID:        "test-key",
		SiteID:       "site_test",
		Capabilities: []string{"read", "write"},
	}
	return req.WithContext(context.WithValue(req.Context(), authmw.AgentContextKey, identity))
}

// jsonRPCRoundtrip POSTs a JSON-RPC envelope through the MCP server
// handler and decodes the response. Returns the parsed Response struct
// plus the raw body for further assertions.
func jsonRPCRoundtrip(t *testing.T, s *Server, method string, params any) (Response, []byte) {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req = withTestAgent(req)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rr.Body.String())
	}
	if resp.Error != nil {
		t.Fatalf("expected ok result, got JSON-RPC error: code=%d msg=%s", resp.Error.Code, resp.Error.Message)
	}
	return resp, rr.Body.Bytes()
}

// freshTestServer builds an MCP server with empty handlers + the full
// registry (tools, resources, prompts) wired the same way NewServer
// does. queries is nil; tests below only exercise registry shape and
// the knowledge / capabilities resources, none of which hit the DB.
func freshTestServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{
		tools:     map[string]Tool{},
		resources: map[string]Resource{},
		prompts:   map[string]Prompt{},
	}
	s.registerTools()
	s.registerResources()
	s.registerPrompts()
	return s
}

// TestWireToolsListReturnsExpectedTools is the headline regression for
// the expansion: the live JSON-RPC tools/list response must contain
// every operator-mutable surface AND the get_capabilities tool that
// agents call to discover their environment.
func TestWireToolsListReturnsExpectedTools(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/list", map[string]any{})
	result := resp.Result.(map[string]any)
	tools := result["tools"].([]any)

	names := map[string]bool{}
	for _, ti := range tools {
		tool := ti.(map[string]any)
		names[tool["name"].(string)] = true
	}

	mustHave := []string{
		"get_site_context",
		"trigger_build",
		"get_capabilities",
		"get_design_playbook",
		"preview_screenshot",
		"lint_block",
		"upsert_css_class",
		"create_knowledgebase_entry",
		"register_allowed_script",
		"upload_font",
		"get_figma_import_url",
		"add_design_reference",
	}
	for _, name := range mustHave {
		if !names[name] {
			t.Errorf("tools/list missing %q (got %d total)", name, len(tools))
		}
	}
	if len(tools) < 30 {
		t.Errorf("expected at least 30 tools after expansion; got %d", len(tools))
	}
}

// TestWireResourcesListReturnsKnowledgeAndServiceContext asserts that
// the JSON-RPC resources/list response includes the curriculum docs,
// the meta knowledge-graph, and the service-context resources.
func TestWireResourcesListReturnsKnowledgeAndServiceContext(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "resources/list", map[string]any{})
	result := resp.Result.(map[string]any)
	resources := result["resources"].([]any)

	uris := map[string]bool{}
	for _, ri := range resources {
		r := ri.(map[string]any)
		uris[r["uri"].(string)] = true
	}

	mustHave := []string{
		"atomicsite://site/context",
		"atomicsite://knowledge/index",
		"atomicsite://knowledge/astro-conventions",
		"atomicsite://knowledge/premium-design-principles",
		"atomicsite://meta/knowledge-graph",
		"atomicsite://meta/capabilities",
		"atomicsite://meta/design-playbook",
		"atomicsite://eval/latest",
		"atomicsite://design-references",
	}
	for _, uri := range mustHave {
		if !uris[uri] {
			t.Errorf("resources/list missing %q (got %d total)", uri, len(resources))
		}
	}
}

// TestWireResourcesReadKnowledgeReturnsMarkdown verifies that fetching
// a curriculum doc returns the embedded markdown body, not just
// metadata. The contents/{text} field must contain the H1 of the doc.
func TestWireResourcesReadKnowledgeReturnsMarkdown(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "resources/read", map[string]any{
		"uri": "atomicsite://knowledge/typography-scale",
	})
	result := resp.Result.(map[string]any)
	contents := result["contents"].([]any)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content entry, got %d", len(contents))
	}
	entry := contents[0].(map[string]any)
	if entry["mimeType"] != "text/markdown" {
		t.Errorf("expected text/markdown mime, got %v", entry["mimeType"])
	}
	body := entry["text"].(string)
	if !strings.Contains(body, "# Typography scale") {
		t.Errorf("expected typography-scale doc body to contain its H1; got first 80 chars: %q", body[:min(80, len(body))])
	}
	// Build the em-dash character at runtime from its codepoint so the
	// source file does not contain the literal U+2014 byte sequence
	// (which would itself trip the pre-commit em-dash hook).
	emDash := string(rune(0x2014))
	if strings.Contains(body, emDash) {
		t.Errorf("typography-scale doc body contains an em dash; loader_test should have caught this")
	}
}

// TestWireResourcesReadKnowledgeGraphHasExpectedShape verifies the
// graph resource produces the nodes + edges + stats payload at the
// shape the agent expects.
func TestWireResourcesReadKnowledgeGraphHasExpectedShape(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "resources/read", map[string]any{
		"uri": "atomicsite://meta/knowledge-graph",
	})
	result := resp.Result.(map[string]any)
	contents := result["contents"].([]any)
	entry := contents[0].(map[string]any)
	if entry["mimeType"] != "application/json" {
		t.Errorf("expected application/json mime, got %v", entry["mimeType"])
	}
	var graph map[string]any
	if err := json.Unmarshal([]byte(entry["text"].(string)), &graph); err != nil {
		t.Fatalf("decode graph: %v", err)
	}
	for _, key := range []string{"nodes", "edges", "stats"} {
		if _, ok := graph[key]; !ok {
			t.Errorf("graph payload missing %q", key)
		}
	}
	stats := graph["stats"].(map[string]any)
	byKind := stats["nodes_by_kind"].(map[string]any)
	if int(byKind["doc"].(float64)) < 18 {
		t.Errorf("expected at least 18 doc nodes, got %v", byKind["doc"])
	}
	if int(byKind["resource"].(float64)) < 30 {
		t.Errorf("expected at least 30 resource nodes, got %v", byKind["resource"])
	}
	if int(byKind["tool"].(float64)) < 30 {
		t.Errorf("expected at least 30 tool nodes, got %v", byKind["tool"])
	}
}

// TestWirePromptsListReturnsExpansionPrompts asserts the four new
// prompts (master_the_stack, audit_for_premium_feel, build_from_figma,
// learn_from_reference) are reachable via JSON-RPC.
func TestWirePromptsListReturnsExpansionPrompts(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "prompts/list", map[string]any{})
	result := resp.Result.(map[string]any)
	prompts := result["prompts"].([]any)
	names := map[string]bool{}
	for _, pi := range prompts {
		p := pi.(map[string]any)
		names[p["name"].(string)] = true
	}
	for _, name := range []string{"master_the_stack", "audit_for_premium_feel", "build_from_figma", "learn_from_reference"} {
		if !names[name] {
			t.Errorf("prompts/list missing %q", name)
		}
	}
}

// TestWireGetCapabilitiesToolReturnsSelfDescribingPayload calls the
// get_capabilities tool through tools/call and asserts the payload
// shape mirrors atomicsite://meta/capabilities.
func TestWireGetCapabilitiesToolReturnsSelfDescribingPayload(t *testing.T) {
	s := freshTestServer(t)
	resp, _ := jsonRPCRoundtrip(t, s, "tools/call", map[string]any{
		"name":      "get_capabilities",
		"arguments": map[string]any{},
	})
	result := resp.Result.(map[string]any)
	if isErr, ok := result["isError"].(bool); ok && isErr {
		t.Fatalf("get_capabilities returned isError=true: %v", result["content"])
	}
	contentArr := result["content"].([]any)
	first := contentArr[0].(map[string]any)
	var snapshot map[string]any
	if err := json.Unmarshal([]byte(first["text"].(string)), &snapshot); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	for _, key := range []string{"identity", "server", "surface", "eval_thresholds", "privacy_invariants"} {
		if _, ok := snapshot[key]; !ok {
			t.Errorf("capabilities payload missing %q", key)
		}
	}
	identity := snapshot["identity"].(map[string]any)
	if identity["site_id"] != "site_test" {
		t.Errorf("expected site_test identity; got %v", identity["site_id"])
	}
	if identity["can_write"] != true {
		t.Errorf("expected can_write=true (test agent has write cap); got %v", identity["can_write"])
	}
}

// TestWireWriteToolGatedByCapability calls a write tool with a
// read-only AgentIdentity and asserts the tool returns isError=true
// instead of a successful execution.
func TestWireWriteToolGatedByCapability(t *testing.T) {
	s := freshTestServer(t)
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "upsert_css_class",
			"arguments": map[string]any{"name": "test-class", "css": ".x{color:red}"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), authmw.AgentContextKey, &authmw.AgentIdentity{
		KeyID:        "ro-key",
		SiteID:       "site_test",
		Capabilities: []string{"read"},
	}))
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	var resp Response
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	result := resp.Result.(map[string]any)
	if isErr, ok := result["isError"].(bool); !ok || !isErr {
		t.Errorf("expected isError=true on write call from read-only key; got result=%v", result)
	}
	contentArr := result["content"].([]any)
	first := contentArr[0].(map[string]any)
	if !strings.Contains(first["text"].(string), "write") {
		t.Errorf("expected write-cap error message to mention 'write'; got %v", first["text"])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
