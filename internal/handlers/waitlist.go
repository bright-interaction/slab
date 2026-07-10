// Package handlers: waitlist.go (Phase 30.5, Cloud Tier MVP, 2026-05-06).
//
// Public-facing waitlist capture for the invite-beta signup model.
// Visitors submit email + name + use_case + region from the pricing
// page; admin reviews entries and mints a workspace_invite token via
// "Send invite" so the recipient can redeem at /cloud/signup/{token}.
//
// Routes:
//
//	POST /api/waitlist                                        public, rate-limited
//	GET  /api/admin/waitlist                                  admin-only list
//	POST /api/admin/waitlist/{id}/invite                      admin mints invite
//	DELETE /api/admin/waitlist/{id}                           admin removes
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bright-interaction/atomicsite/internal/config"
	"github.com/bright-interaction/atomicsite/internal/email"
	"github.com/bright-interaction/atomicsite/internal/store"
)

// WaitlistHandler covers the public capture + admin review surface.
type WaitlistHandler struct {
	cfg     *config.Config
	queries *store.Queries
	mailer  *email.MailerSendClient

	// Per-IP rate limiter. 5 submissions per hour per IP. Spam
	// mitigation; legitimate sign-up traffic is one per IP.
	rateMu       sync.Mutex
	rateBucket   map[string][]time.Time
	rateWindow   time.Duration
	rateMax      int
	rateLastSwep time.Time
}

func NewWaitlistHandler(cfg *config.Config, queries *store.Queries, mailer *email.MailerSendClient) *WaitlistHandler {
	return &WaitlistHandler{
		cfg:        cfg,
		queries:    queries,
		mailer:     mailer,
		rateBucket: make(map[string][]time.Time),
		rateWindow: time.Hour,
		rateMax:    5,
	}
}

// Submit: POST /api/waitlist (public, rate-limited).
// Body: { email, name, use_case, region }.
func (h *WaitlistHandler) Submit(w http.ResponseWriter, r *http.Request) {
	if !h.allowSubmit(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "Slow down. Try again in an hour.")
		return
	}
	var req struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		UseCase string `json:"use_case"`
		Region  string `json:"region"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	em := strings.ToLower(strings.TrimSpace(req.Email))
	if !looksLikeEmail(em) {
		writeError(w, http.StatusBadRequest, "Valid email required")
		return
	}
	region := strings.ToLower(strings.TrimSpace(req.Region))
	if region != "eu" && region != "us" {
		region = "eu"
	}
	id := newID()
	if err := h.queries.CreateWaitlistEntry(r.Context(), store.CreateWaitlistEntryParams{
		ID:        id,
		Email:     em,
		Name:      strings.TrimSpace(req.Name),
		UseCase:   strings.TrimSpace(req.UseCase),
		Region:    region,
		SourceIp:  clientIP(r),
		UserAgent: truncateString(r.UserAgent(), 256),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to record waitlist entry")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"status": "ok",
		"hint":   "We'll email you when your invite is ready. Reply if you have questions.",
	})
}

// AdminList: GET /api/admin/waitlist
func (h *WaitlistHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.ListWaitlist(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list waitlist")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"waitlist": rows})
}

// AdminInvite: POST /api/admin/waitlist/{id}/invite. Mints a
// workspace_invite token, marks the waitlist row invited, and emails
// the recipient via MailerSend (or skips email when MailerSend is not
// configured, returning the URL so the admin can copy it manually).
func (h *WaitlistHandler) AdminInvite(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "id")
	row, err := h.queries.GetWaitlistByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Waitlist entry not found")
		return
	}
	if row.InvitedAt != "" {
		writeJSON(w, http.StatusConflict, map[string]string{
			"status": "already-invited",
			"token":  row.InviteToken,
		})
		return
	}
	tokenBytes := make([]byte, 24)
	_, _ = rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)
	if err := h.queries.MarkWaitlistInvited(r.Context(), store.MarkWaitlistInvitedParams{
		ID:          id,
		InviteToken: token,
		Notes:       row.Notes,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to mark invited")
		return
	}
	signupURL := strings.TrimRight(h.cfg.BaseURL, "/") + "/cloud/signup/" + token
	if h.mailer != nil && h.mailer.IsConfigured() {
		if err := h.mailer.SendWaitlistInvite(r.Context(), row.Email, row.Name, signupURL); err != nil {
			// Non-fatal: row is marked invited; the admin can copy
			// the URL from the response and send manually.
			writeJSON(w, http.StatusOK, map[string]any{
				"status": "invited-mail-failed",
				"token":  token,
				"url":    signupURL,
				"error":  err.Error(),
			})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "invited",
		"token":  token,
		"url":    signupURL,
	})
}

// AdminDelete: DELETE /api/admin/waitlist/{id}
func (h *WaitlistHandler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "id")
	if err := h.queries.DeleteWaitlistEntry(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete waitlist entry")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// allowSubmit enforces the per-IP rate limit. Returns true if the
// caller may submit, false if they're over the cap.
func (h *WaitlistHandler) allowSubmit(ip string) bool {
	if ip == "" {
		ip = "unknown"
	}
	now := time.Now()
	cutoff := now.Add(-h.rateWindow)
	h.rateMu.Lock()
	defer h.rateMu.Unlock()
	// Periodic sweep so the map doesn't grow forever for IPs that
	// hit once and never come back. Cheap (linear in map size) and
	// runs at most every minute.
	if now.Sub(h.rateLastSwep) > time.Minute {
		for k, ts := range h.rateBucket {
			fresh := ts[:0]
			for _, t := range ts {
				if t.After(cutoff) {
					fresh = append(fresh, t)
				}
			}
			if len(fresh) == 0 {
				delete(h.rateBucket, k)
			} else {
				h.rateBucket[k] = fresh
			}
		}
		h.rateLastSwep = now
	}
	ts := h.rateBucket[ip]
	fresh := ts[:0]
	for _, t := range ts {
		if t.After(cutoff) {
			fresh = append(fresh, t)
		}
	}
	if len(fresh) >= h.rateMax {
		h.rateBucket[ip] = fresh
		return false
	}
	h.rateBucket[ip] = append(fresh, now)
	return true
}

// looksLikeEmail is a permissive check; real validation happens at
// send time. The MailerSend API will reject malformed addresses.
// (clientIP and truncateString live in handlers/track.go and
// handlers/helpers.go respectively; this file reuses them.)
func looksLikeEmail(s string) bool {
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return false
	}
	dot := strings.LastIndex(s, ".")
	return dot > at
}
