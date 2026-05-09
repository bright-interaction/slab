package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brightinteraction/atomicsite/internal/config"
)

// fakeShieldUpdater implements ShieldKeyUpdater for tests so we can
// observe what the admin_reload handler actually pushes through.
type fakeShieldUpdater struct {
	enabled bool
	gotKey  []byte
	err     error
	calls   int
}

func (f *fakeShieldUpdater) UpdateShieldKey(k []byte) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	dup := make([]byte, len(k))
	copy(dup, k)
	f.gotKey = dup
	return nil
}

func (f *fakeShieldUpdater) ShieldEnabled() bool { return f.enabled }

const testToken = "test-admin-token-32-bytes-long-aaa"

func newReq(t *testing.T, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/admin/reload-secrets", bytes.NewReader(b))
	r.Header.Set("Authorization", "Bearer "+testToken)
	return r
}

func TestAdminReload_ShieldKey_HotSwap(t *testing.T) {
	cfg := &config.Config{AdminReloadToken: testToken}
	mcp := &fakeShieldUpdater{enabled: true}
	h := NewAdminReloadHandler(cfg, nil).WithMCP(mcp)

	const newKey = "abcdefghijklmnopqrstuvwxyz012345" // 32 ASCII chars
	w := httptest.NewRecorder()
	h.ReloadSecrets(w, newReq(t, map[string]any{
		"secrets": map[string]string{"SHIELD_KEY": newKey},
	}))

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
	if mcp.calls != 1 {
		t.Fatalf("UpdateShieldKey calls = %d, want 1", mcp.calls)
	}
	if string(mcp.gotKey) != newKey {
		t.Fatalf("got key %q, want %q", mcp.gotKey, newKey)
	}
}

func TestAdminReload_ShieldKey_RejectedWhenShieldDisabled(t *testing.T) {
	cfg := &config.Config{AdminReloadToken: testToken}
	mcp := &fakeShieldUpdater{enabled: false}
	h := NewAdminReloadHandler(cfg, nil).WithMCP(mcp)

	w := httptest.NewRecorder()
	h.ReloadSecrets(w, newReq(t, map[string]any{
		"secrets": map[string]string{"SHIELD_KEY": "abcdefghijklmnopqrstuvwxyz012345"},
	}))

	// 503 ServiceUnavailable, NOT 204; the rotation engine treats this as
	// a phase-1 failure and won't advance last_rotated_at, forcing retry.
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if mcp.calls != 0 {
		t.Fatalf("UpdateShieldKey called %d times when shield disabled; want 0", mcp.calls)
	}
}

func TestAdminReload_ShieldKey_RejectedWhenMCPNotWired(t *testing.T) {
	cfg := &config.Config{AdminReloadToken: testToken}
	// No WithMCP call: h.mcp is nil. Same shape as a service that
	// hasn't been redeployed yet after this PR.
	h := NewAdminReloadHandler(cfg, nil)

	w := httptest.NewRecorder()
	h.ReloadSecrets(w, newReq(t, map[string]any{
		"secrets": map[string]string{"SHIELD_KEY": "abcdefghijklmnopqrstuvwxyz012345"},
	}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (mcp not wired)", w.Code)
	}
}

func TestAdminReload_AllowlistRejectsUnknown(t *testing.T) {
	cfg := &config.Config{AdminReloadToken: testToken}
	h := NewAdminReloadHandler(cfg, nil).WithMCP(&fakeShieldUpdater{enabled: true})

	w := httptest.NewRecorder()
	h.ReloadSecrets(w, newReq(t, map[string]any{
		"secrets": map[string]string{"SOME_OTHER_KEY": "value"},
	}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (no recognised secrets)", w.Code)
	}
}

func TestAdminReload_BadBearer(t *testing.T) {
	cfg := &config.Config{AdminReloadToken: testToken}
	h := NewAdminReloadHandler(cfg, nil)

	r := httptest.NewRequest(http.MethodPost, "/admin/reload-secrets",
		bytes.NewReader([]byte(`{"secrets":{"SHIELD_KEY":"x"}}`)))
	r.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	h.ReloadSecrets(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
