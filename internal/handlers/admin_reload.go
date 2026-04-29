package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/brightinteraction/atomicsite/internal/config"
)

// AdminReloadHandler accepts authenticated POSTs from Dockyard's rotation
// engine and hot-swaps the BrightCRM HMAC pair without a container restart.
//
// Atomicsite uses a single shared secret for BOTH directions (signing
// outbound /webhooks/atomicsite calls and verifying inbound /t/inbound
// pings), so a rotation event updates two in-memory targets:
//
//  1. The TrackHandler's inboundVerifier (current + previous slots)
//  2. The TrackHandler's crmClient signing secret (current only)
//
// Request shape mirrors the BrightCRM-side handler:
//
//	POST /admin/reload-secrets
//	Authorization: Bearer <ADMIN_RELOAD_TOKEN>
//	{
//	  "secrets": {
//	    "BRIGHTCRM_WEBHOOK_SECRET":          "new-value",
//	    "BRIGHTCRM_WEBHOOK_SECRET_PREVIOUS": "old-value"
//	  }
//	}
//
// 204 success, 401 bad bearer, 503 if ADMIN_RELOAD_TOKEN unset.
type AdminReloadHandler struct {
	cfg   *config.Config
	track *TrackHandler
}

func NewAdminReloadHandler(cfg *config.Config, track *TrackHandler) *AdminReloadHandler {
	return &AdminReloadHandler{cfg: cfg, track: track}
}

type reloadRequest struct {
	Secrets map[string]string `json:"secrets"`
}

var atomicsiteReloadAllowlist = map[string]struct{}{
	"BRIGHTCRM_WEBHOOK_SECRET":          {},
	"BRIGHTCRM_WEBHOOK_SECRET_PREVIOUS": {},
}

func (h *AdminReloadHandler) ReloadSecrets(w http.ResponseWriter, r *http.Request) {
	if h.cfg.AdminReloadToken == "" {
		writeError(w, http.StatusServiceUnavailable, "admin reload not configured")
		return
	}

	const prefix = "Bearer "
	hdr := r.Header.Get("Authorization")
	if len(hdr) <= len(prefix) || hdr[:len(prefix)] != prefix {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	if subtle.ConstantTimeCompare([]byte(hdr[len(prefix):]), []byte(h.cfg.AdminReloadToken)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid bearer token")
		return
	}

	var req reloadRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Secrets == nil {
		writeError(w, http.StatusBadRequest, "missing secrets")
		return
	}

	staged := make(map[string]string, len(req.Secrets))
	for k, v := range req.Secrets {
		if _, ok := atomicsiteReloadAllowlist[k]; ok {
			staged[k] = v
		}
	}
	if len(staged) == 0 {
		writeError(w, http.StatusBadRequest, "no recognised secrets in payload")
		return
	}

	primary, primaryProvided := staged["BRIGHTCRM_WEBHOOK_SECRET"]
	secondary, _ := staged["BRIGHTCRM_WEBHOOK_SECRET_PREVIOUS"]

	if !primaryProvided {
		// We can't atomically half-update; require both keys (secondary
		// may be empty string to signal "clear").
		writeError(w, http.StatusBadRequest, "rotation must supply BRIGHTCRM_WEBHOOK_SECRET")
		return
	}

	// Inbound verifier (accepts both during grace window).
	if v := h.track.InboundVerifier(); v != nil {
		v.Update(primary, secondary)
	}
	// Outbound signer (always current only).
	if c := h.track.CRMClient(); c != nil {
		c.UpdateSecret(primary)
	}

	slog.Info("admin: reloaded secrets",
		"keys", keysOf(staged),
		"verifier_secondary_active", secondary != "")
	w.WriteHeader(http.StatusNoContent)
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
