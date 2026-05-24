package apps

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrimaryBearer_PrefersWellKnownKeys(t *testing.T) {
	schema := []CredentialField{
		{Key: "app_id", Kind: "string"},
		{Key: "admin_api_key", Kind: "secret"},
		{Key: "api_key", Kind: "secret"},
	}
	creds := map[string]string{
		"app_id":        "ABCDE",
		"admin_api_key": "admin-key",
		"api_key":       "primary-key",
	}
	got := PrimaryBearer(schema, creds)
	if got != "primary-key" {
		t.Fatalf("expected api_key to win, got %q", got)
	}
}

func TestPrimaryBearer_FallsBackToFirstSecretField(t *testing.T) {
	schema := []CredentialField{
		{Key: "app_id", Kind: "string"},
		{Key: "admin_api_key", Kind: "secret"},
	}
	creds := map[string]string{
		"app_id":        "ABCDE",
		"admin_api_key": "admin-key",
	}
	got := PrimaryBearer(schema, creds)
	if got != "admin-key" {
		t.Fatalf("expected admin_api_key to win, got %q", got)
	}
}

func TestPrimaryBearer_ReturnsEmptyWhenNoSecret(t *testing.T) {
	schema := []CredentialField{
		{Key: "app_id", Kind: "string"},
	}
	creds := map[string]string{"app_id": "ABCDE"}
	if PrimaryBearer(schema, creds) != "" {
		t.Fatalf("expected empty bearer when no secret credential")
	}
}

func TestCallUpstreamMCP_HappyPath(t *testing.T) {
	var seenAuth, seenBody, seenCred string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		seenCred = r.Header.Get("X-Atomic-Cred-App-Id")
		raw, _ := io.ReadAll(r.Body)
		seenBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"atomic-use-app","result":{"ok":true,"echo":"hi"}}`))
	}))
	defer srv.Close()

	res, err := CallUpstreamMCP(context.Background(), UpstreamCall{
		URL:      srv.URL,
		ToolName: "ping",
		Arguments: map[string]any{
			"msg": "hi",
		},
		Credentials: map[string]string{
			"api_key": "sk_test_xyz",
			"app_id":  "ABCDE12345",
		},
		Schema: []CredentialField{
			{Key: "app_id", Kind: "string"},
			{Key: "api_key", Kind: "secret"},
		},
	})
	if err != nil {
		t.Fatalf("CallUpstreamMCP returned error: %v", err)
	}
	if res.HTTPStatus != 200 {
		t.Errorf("expected HTTP 200, got %d", res.HTTPStatus)
	}
	if res.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", res.ErrorMessage)
	}
	if seenAuth != "Bearer sk_test_xyz" {
		t.Errorf("expected Bearer header from api_key, got %q", seenAuth)
	}
	if seenCred != "ABCDE12345" {
		t.Errorf("expected X-Atomic-Cred-App-Id header, got %q", seenCred)
	}
	if !strings.Contains(seenBody, `"method":"tools/call"`) {
		t.Errorf("expected tools/call envelope, got %s", seenBody)
	}
	if !strings.Contains(seenBody, `"name":"ping"`) {
		t.Errorf("expected tool name in envelope, got %s", seenBody)
	}
	var result map[string]any
	if err := json.Unmarshal(res.ResultJSON, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result["ok"] != true {
		t.Errorf("expected ok=true in result, got %v", result)
	}
}

func TestCallUpstreamMCP_UpstreamReturnsJSONRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"atomic-use-app","error":{"code":-32601,"message":"Method not found"}}`))
	}))
	defer srv.Close()

	res, err := CallUpstreamMCP(context.Background(), UpstreamCall{
		URL:      srv.URL,
		ToolName: "nonexistent",
	})
	if err != nil {
		t.Fatalf("expected nil Go error for JSON-RPC error envelope, got %v", err)
	}
	if res.ErrorCode != -32601 {
		t.Errorf("expected error code -32601, got %d", res.ErrorCode)
	}
	if res.ErrorMessage != "Method not found" {
		t.Errorf("expected upstream error message, got %q", res.ErrorMessage)
	}
}

func TestCallUpstreamMCP_HTTP500FromUpstream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res, err := CallUpstreamMCP(context.Background(), UpstreamCall{
		URL:      srv.URL,
		ToolName: "any",
	})
	if err != nil {
		t.Fatalf("expected nil Go error for HTTP 500, got %v", err)
	}
	if res.HTTPStatus != 500 {
		t.Errorf("expected HTTPStatus 500, got %d", res.HTTPStatus)
	}
	if !strings.Contains(res.ErrorMessage, "HTTP 500") {
		t.Errorf("expected HTTP 500 in error message, got %q", res.ErrorMessage)
	}
}

func TestCallUpstreamMCP_RejectsOversizedResponse(t *testing.T) {
	big := strings.Repeat("a", maxUpstreamResponseBytes+1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, big)
	}))
	defer srv.Close()

	res, err := CallUpstreamMCP(context.Background(), UpstreamCall{
		URL:      srv.URL,
		ToolName: "flood",
	})
	if err != nil {
		t.Fatalf("expected nil Go error, got %v", err)
	}
	if !strings.Contains(res.ErrorMessage, "exceeded") {
		t.Errorf("expected size-cap error, got %q", res.ErrorMessage)
	}
}

func TestCallUpstreamMCP_RejectsInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "not json at all")
	}))
	defer srv.Close()

	res, err := CallUpstreamMCP(context.Background(), UpstreamCall{
		URL:      srv.URL,
		ToolName: "any",
	})
	if err != nil {
		t.Fatalf("expected nil Go error, got %v", err)
	}
	if !strings.Contains(res.ErrorMessage, "JSON-RPC") {
		t.Errorf("expected JSON-RPC parse error, got %q", res.ErrorMessage)
	}
}

func TestCallUpstreamMCP_NoCredentialsSendsNoBearer(t *testing.T) {
	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"atomic-use-app","result":{}}`))
	}))
	defer srv.Close()

	_, err := CallUpstreamMCP(context.Background(), UpstreamCall{
		URL:      srv.URL,
		ToolName: "anon",
	})
	if err != nil {
		t.Fatalf("CallUpstreamMCP: %v", err)
	}
	if seenAuth != "" {
		t.Errorf("expected no Authorization header when no credentials, got %q", seenAuth)
	}
}

func TestCallUpstreamMCP_RejectsEmptyURL(t *testing.T) {
	_, err := CallUpstreamMCP(context.Background(), UpstreamCall{
		ToolName: "x",
	})
	if err == nil {
		t.Fatalf("expected error for empty URL")
	}
}

func TestCallUpstreamMCP_RejectsEmptyToolName(t *testing.T) {
	_, err := CallUpstreamMCP(context.Background(), UpstreamCall{
		URL: "http://example.invalid",
	})
	if err == nil {
		t.Fatalf("expected error for empty tool name")
	}
}
