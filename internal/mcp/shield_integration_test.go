package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/shield"
)

// TestShieldEndToEndOnToolCall verifies the moat: every PII string
// crossing the MCP boundary is replaced with a [shield:...] marker
// before reaching the agent, AND token-bearing tool-call arguments
// resolve back to plaintext server-side before the underlying handler
// runs. This is the contract Tom learned about: agent sees context +
// metadata only, never raw values.
func TestShieldEndToEndOnToolCall(t *testing.T) {
	const (
		secretEmail = "anna@andersson-law.se"
		secretName  = "Anna Andersson"
		secretPhone = "+46701234567"
	)

	s := &Server{
		tools:     map[string]Tool{},
		resources: map[string]Resource{},
		prompts:   map[string]Prompt{},
	}

	// One read tool that returns PII-bearing content. Demonstrates the
	// canonical handler pattern: typed struct + shield tags + an
	// explicit Session.Shield call before marshaling. The dispatcher's
	// ShieldJSON safety net catches anything the handler missed (email,
	// phone via regex), but the typed pass is what guarantees name +
	// other context-only PII is caught.
	type contact struct {
		ID    string `json:"id"    shield:"id"`
		Email string `json:"email" shield:"tokenize,kind=email,hint=domain len"`
		Name  string `json:"name"  shield:"tokenize,kind=name,hint=initials"`
		Phone string `json:"phone" shield:"tokenize,kind=phone,hint=country len"`
		Stage string `json:"stage"`
	}
	var lastResolvedTo string
	s.tools["test_get_contact"] = Tool{
		Name:        "test_get_contact",
		Description: "fixture for shield integration test",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Handler: func(ctx context.Context, _ *middleware.AgentIdentity, _ json.RawMessage) (string, error) {
			c := contact{
				ID:    "c-1",
				Email: secretEmail,
				Name:  secretName,
				Phone: secretPhone,
				Stage: "qualified",
			}
			if sess := shield.SessionFromContext(ctx); sess != nil {
				if err := sess.Shield(ctx, &c); err != nil {
					return "", err
				}
			}
			b, _ := json.Marshal(c)
			return string(b), nil
		},
	}

	// One write tool that records what plaintext value it received. The
	// agent will call it referencing a token; the shield layer must
	// resolve that token to the real email before the handler runs.
	s.tools["test_send_email"] = Tool{
		Name:          "test_send_email",
		Description:   "fixture write tool",
		InputSchema:   json.RawMessage(`{"type":"object"}`),
		RequiresWrite: false, // skip the capability check for the fixture
		Handler: func(_ context.Context, _ *middleware.AgentIdentity, args json.RawMessage) (string, error) {
			var p struct {
				To string `json:"to"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", err
			}
			lastResolvedTo = p.To
			return `{"sent":true}`, nil
		},
	}

	// Wire shield with an in-memory store (no DB needed).
	store := shield.NewMemoryStore()
	key := []byte("01234567890123456789012345678901")
	s.WithShield(store, key, time.Minute, shield.HintFull)

	// Stub auth identity. The real middleware injects this; we set it
	// directly on the request context for the test.
	identity := &middleware.AgentIdentity{
		KeyID:        "test-key",
		SiteID:       "site-test",
		Capabilities: []string{"read", "write"},
	}

	// Step 1: agent calls test_get_contact via JSON-RPC.
	rpcBody, _ := json.Marshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      "1",
		Method:  "tools/call",
		Params: mustJSONRaw(map[string]any{
			"name":      "test_get_contact",
			"arguments": map[string]any{},
		}),
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(rpcBody)))
	req.Header.Set("Mcp-Session-Id", "test-session-1")
	req = withAgentForTest(req, identity)
	rec := httptest.NewRecorder()

	s.handlePost(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get: status %d body=%s", rec.Code, rec.Body.String())
	}
	getResp := decodeRPC(t, rec.Body.Bytes())
	if getResp.Error != nil {
		t.Fatalf("get: rpc error: %+v", getResp.Error)
	}
	contentText := extractContentText(t, getResp.Result)

	// The sensitive values must NOT appear in the agent-visible payload.
	for _, leak := range []string{secretEmail, secretName, secretPhone} {
		if strings.Contains(contentText, leak) {
			t.Fatalf("agent payload leaks %q:\n%s", leak, contentText)
		}
	}
	// The shield markers must.
	for _, marker := range []string{"[shield:email:tok_", "[shield:phone:tok_"} {
		if !strings.Contains(contentText, marker) {
			t.Errorf("expected marker %q in agent payload:\n%s", marker, contentText)
		}
	}

	// Step 2: agent calls test_send_email referencing the email marker.
	// Pull the marker out of the agent payload so we use the same one
	// the LLM would see.
	emailMarker := extractMarkerForKind(t, contentText, "email")
	sendBody, _ := json.Marshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      "2",
		Method:  "tools/call",
		Params: mustJSONRaw(map[string]any{
			"name": "test_send_email",
			"arguments": map[string]any{
				"to": emailMarker,
			},
		}),
	})
	req2 := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(sendBody)))
	req2.Header.Set("Mcp-Session-Id", "test-session-1")
	req2 = withAgentForTest(req2, identity)
	rec2 := httptest.NewRecorder()
	s.handlePost(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("send: status %d body=%s", rec2.Code, rec2.Body.String())
	}
	sendResp := decodeRPC(t, rec2.Body.Bytes())
	if sendResp.Error != nil {
		t.Fatalf("send: rpc error: %+v", sendResp.Error)
	}

	// Server-side, the handler must have received the resolved real email.
	if lastResolvedTo != secretEmail {
		t.Fatalf("server resolved-to mismatch: got %q want %q", lastResolvedTo, secretEmail)
	}

	// Step 3: cross-session replay protection. A different MCP session
	// using the same marker must be rejected by Unshield, the underlying
	// handler must NOT run, and the secret must not surface.
	lastResolvedTo = "" // reset so we can detect whether step 3 ran the handler
	req3 := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(sendBody)))
	req3.Header.Set("Mcp-Session-Id", "different-session-2")
	req3 = withAgentForTest(req3, identity)
	rec3 := httptest.NewRecorder()
	s.handlePost(rec3, req3)
	resp3 := decodeRPC(t, rec3.Body.Bytes())
	if resp3.Error != nil {
		t.Fatalf("cross-session call: unexpected rpc error: %+v", resp3.Error)
	}
	// Dispatcher should have produced an application-level IsError
	// because UnshieldJSON failed.
	body3 := extractContentText(t, resp3.Result)
	if !strings.Contains(body3, "shield: failed to resolve tool arguments") {
		t.Fatalf("cross-session call: expected shield error in result, got: %s", body3)
	}
	if lastResolvedTo != "" {
		t.Fatalf("cross-session replay reached the handler: lastResolvedTo=%q", lastResolvedTo)
	}
	if strings.Contains(body3, secretEmail) {
		t.Fatalf("cross-session error response leaked the secret: %s", body3)
	}
}

// TestShieldDisabledPasses covers the off path: when no SHIELD_KEY is
// configured, the MCP server returns plaintext as before. Verifies the
// no-op default.
func TestShieldDisabledPasses(t *testing.T) {
	s := &Server{
		tools:     map[string]Tool{},
		resources: map[string]Resource{},
		prompts:   map[string]Prompt{},
	}
	s.tools["test_passthrough"] = Tool{
		Name:        "test_passthrough",
		Description: "fixture",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Handler: func(_ context.Context, _ *middleware.AgentIdentity, _ json.RawMessage) (string, error) {
			return `{"email":"plain@example.com"}`, nil
		},
	}

	identity := &middleware.AgentIdentity{KeyID: "k", SiteID: "s", Capabilities: []string{"read"}}
	rpcBody, _ := json.Marshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      "1",
		Method:  "tools/call",
		Params:  mustJSONRaw(map[string]any{"name": "test_passthrough", "arguments": map[string]any{}}),
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(rpcBody)))
	req = withAgentForTest(req, identity)
	rec := httptest.NewRecorder()
	s.handlePost(rec, req)
	resp := decodeRPC(t, rec.Body.Bytes())
	got := extractContentText(t, resp.Result)
	if !strings.Contains(got, "plain@example.com") {
		t.Fatalf("shield disabled but email was redacted: %s", got)
	}
}

// --- test helpers -----------------------------------------------------------

func mustJSONRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func decodeRPC(t *testing.T, body []byte) Response {
	t.Helper()
	var r Response
	if err := json.Unmarshal(body, &r); err != nil {
		t.Fatalf("decode rpc: %v body=%s", err, body)
	}
	return r
}

func extractContentText(t *testing.T, result any) string {
	t.Helper()
	// Result is one of our typed structs (ToolCallResult). Re-marshal
	// + decode into the loose shape so this helper is reusable for
	// resources too.
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var holder struct {
		Content []Content `json:"content"`
	}
	if err := json.Unmarshal(raw, &holder); err != nil {
		t.Fatalf("decode content: %v body=%s", err, raw)
	}
	if len(holder.Content) == 0 {
		t.Fatalf("no content in result: %s", raw)
	}
	return holder.Content[0].Text
}

func extractMarkerForKind(t *testing.T, body, kind string) string {
	t.Helper()
	prefix := "[shield:" + kind + ":"
	i := strings.Index(body, prefix)
	if i < 0 {
		t.Fatalf("no %s marker in body: %s", kind, body)
	}
	end := strings.Index(body[i:], "]")
	if end < 0 {
		t.Fatalf("malformed marker in body: %s", body)
	}
	return body[i : i+end+1]
}

// withAgentForTest stuffs an AgentIdentity into the request context the
// way middleware.AgentAuthMiddleware would in production.
func withAgentForTest(r *http.Request, id *middleware.AgentIdentity) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.AgentContextKey, id)
	return r.WithContext(ctx)
}

// guard against unused import warnings when iterating; remove once stable.
var _ = fmt.Sprintf
