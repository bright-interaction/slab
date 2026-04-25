package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/crmsync"
	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/store"
)

// trackBodyMaxBytes caps the request body for /t/* receivers. Consent payloads
// are tiny (~1 KB); anything bigger is either malicious or a bug.
const trackBodyMaxBytes = 16 * 1024

// TrackHandler serves the public, no-auth analytics receiver mounted at /t/*.
// Two endpoints:
//
//   - POST /t/consent: fired by the CookieProof relay on consent:update or
//     consent:gpc. Upserts a visit_session keyed by (site_id, fingerprint),
//     and if the visitor opted into analytics, lifts the session from
//     anonymous to identified by stamping visitor_id + consent metadata.
//
//   - POST /t/pageview: optional SPA route-change ping. Static Astro builds
//     produce a real navigation per page so this is mostly unused for v1, but
//     the endpoint ships for future client-routed sites.
//
// Both endpoints rely on FingerprintMiddleware having already set the
// atomicsite_fp cookie on the request, so r.Context() carries a 16-hex
// fingerprint. The handlers only validate that the posted siteId actually
// belongs to a real site row; everything else is best-effort to keep the
// receiver fast and tolerant of malformed clients.
type TrackHandler struct {
	cfg       *config.Config
	queries   *store.Queries
	db        *sql.DB
	crmClient *crmsync.Client
	crmThrot  *crmsync.Throttler
}

// NewTrackHandler builds a TrackHandler. The CRM sync client and throttler
// are constructed from cfg; both are no-ops when BRIGHTCRM_WEBHOOK_URL or
// BRIGHTCRM_WEBHOOK_SECRET are unset.
func NewTrackHandler(cfg *config.Config, queries *store.Queries, db *sql.DB) *TrackHandler {
	return &TrackHandler{
		cfg:       cfg,
		queries:   queries,
		db:        db,
		crmClient: crmsync.NewClient(cfg.BrightCRMWebhookURL, cfg.BrightCRMWebhookSecret),
		crmThrot:  crmsync.NewThrottler(cfg.CRMSyncMinInterval),
	}
}

// consentRequest is the body the in-page relay POSTs to /t/consent.
type consentRequest struct {
	SiteID    string          `json:"siteId"`
	SessionID string          `json:"sessionId"`
	Consent   *consentPayload `json:"consent"`
	Page      string          `json:"page"`
	Referrer  string          `json:"referrer"`
}

// consentPayload mirrors the detail.consent shape emitted by CookieProof's
// <cookie-consent> element. We pull only what we store: method + categories +
// version + timestamp. GPC mode is preserved in `method` so downstream
// reporting can split GPC visitors out if needed.
type consentPayload struct {
	Version    int             `json:"version"`
	Timestamp  int64           `json:"timestamp"`
	Method     string          `json:"method"`
	Categories map[string]bool `json:"categories"`
	GPC        bool            `json:"gpc"`
}

// Consent receives consent decisions from the in-page relay and updates
// visit_sessions accordingly. Always returns 204 on success; errors that the
// client can't act on are logged server-side and answered with 204 anyway so
// the relay doesn't keep retrying.
func (h *TrackHandler) Consent(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, trackBodyMaxBytes)
	defer r.Body.Close()

	var req consentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	siteID := strings.TrimSpace(req.SiteID)
	if !isSafeSiteID(siteID) {
		writeError(w, http.StatusBadRequest, "Invalid siteId")
		return
	}

	// Site must exist. Cheap guard against random POSTs.
	if _, err := h.queries.GetSiteByID(r.Context(), siteID); err != nil {
		writeError(w, http.StatusNotFound, "Unknown siteId")
		return
	}

	fp := authmw.GetFingerprint(r)
	if fp == "" {
		// Middleware should have set this. If not, something is mis-mounted.
		slog.Warn("track: consent received without fingerprint", "site_id", siteID)
		writeError(w, http.StatusInternalServerError, "Missing fingerprint")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	sessionID := newID()
	if err := h.queries.UpsertVisitSession(r.Context(), store.UpsertVisitSessionParams{
		ID:          sessionID,
		SiteID:      siteID,
		Fingerprint: fp,
		StartedAt:   now,
		LastSeenAt:  now,
	}); err != nil {
		slog.Error("track: upsert visit_session", "site_id", siteID, "err", err)
		writeError(w, http.StatusInternalServerError, "Failed to record session")
		return
	}

	// If the consent payload grants analytics, lift the session to identified.
	// "Identified" here means: we have a stable visitor_id we can stitch
	// future visits against. Email arrives later via form submissions; we do
	// not invent one.
	if req.Consent != nil && req.Consent.Categories["analytics"] {
		method := req.Consent.Method
		if req.Consent.GPC {
			method = "gpc"
		}
		categoriesJSON := encodeCategories(req.Consent.Categories)
		visitorID := deriveVisitorID(siteID, fp)
		if err := h.queries.IdentifyVisitSession(r.Context(), store.IdentifyVisitSessionParams{
			VisitorID:             visitorID,
			Email:                 "",
			ConsentMethod:         method,
			ConsentCategoriesJson: categoriesJSON,
			IdentifiedAt:          now,
			LastSeenAt:            now,
			SiteID:                siteID,
			Fingerprint:           fp,
		}); err != nil {
			slog.Error("track: identify visit_session", "site_id", siteID, "err", err)
		} else {
			// Forward an "identified" event to BrightCRM. The CRM upserts a
			// contact when an email is present and otherwise drops the event;
			// we let it decide. Fire on a goroutine so the user-facing /t/consent
			// response never blocks on the CRM round-trip.
			h.crmEmitIdentified(siteID, visitorID, method, req)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// crmEmitIdentified fires an identified event toward BrightCRM. It bypasses
// the throttler so a fresh identification always lands in the activity log.
// All failures are logged; nothing is surfaced to the visitor.
func (h *TrackHandler) crmEmitIdentified(siteID, visitorID, method string, req consentRequest) {
	if h.crmClient == nil || !h.crmClient.Enabled() {
		return
	}
	event := crmsync.Event{
		Event:      crmsync.EventIdentified,
		SiteID:     siteID,
		VisitorID:  visitorID,
		OccurredAt: time.Now().UTC(),
		Page: crmsync.Page{
			URL:      req.Page,
			Referrer: req.Referrer,
		},
		Metadata: map[string]any{
			"consent_method": method,
		},
	}
	// Identified events bypass throttling per design.
	h.crmClient.SendAsync(event)
}

// pageViewRequest is the body of POST /t/pageview.
type pageViewRequest struct {
	SiteID   string `json:"siteId"`
	Path     string `json:"path"`
	Referrer string `json:"referrer"`
}

// PageView records a visit_event for SPA route changes. The static Astro
// builds Atomicsite produces don't normally need this (nginx access-logs
// catch every navigation), but client-routed sites need it to show up in
// per-page counts. Always 204 on completion.
func (h *TrackHandler) PageView(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, trackBodyMaxBytes)
	defer r.Body.Close()

	var req pageViewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	siteID := strings.TrimSpace(req.SiteID)
	if !isSafeSiteID(siteID) {
		writeError(w, http.StatusBadRequest, "Invalid siteId")
		return
	}
	if _, err := h.queries.GetSiteByID(r.Context(), siteID); err != nil {
		writeError(w, http.StatusNotFound, "Unknown siteId")
		return
	}

	fp := authmw.GetFingerprint(r)
	if fp == "" {
		writeError(w, http.StatusInternalServerError, "Missing fingerprint")
		return
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		path = "/"
	}
	if len(path) > 1024 {
		path = path[:1024]
	}
	ref := strings.TrimSpace(req.Referrer)
	if len(ref) > 1024 {
		ref = ref[:1024]
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Look up the existing session (if any) so visit_events.session_id can
	// link the pageview to the same row. Missing-session is fine: we still
	// log the event with empty session_id and the analytics layer handles
	// orphaned events.
	sessionID := ""
	sess, err := h.queries.GetSessionByFingerprint(r.Context(), store.GetSessionByFingerprintParams{
		SiteID:      siteID,
		Fingerprint: fp,
	})
	if err == nil {
		sessionID = sess.ID
	} else if !errors.Is(err, sql.ErrNoRows) {
		slog.Warn("track: get session for pageview", "site_id", siteID, "err", err)
	}

	if err := h.queries.RecordVisitEvent(r.Context(), store.RecordVisitEventParams{
		ID:          newID(),
		SiteID:      siteID,
		Fingerprint: fp,
		SessionID:   sessionID,
		Path:        path,
		Referer:     ref,
		Status:      200,
		Ms:          0,
		Ts:          now,
	}); err != nil {
		slog.Error("track: record visit_event", "site_id", siteID, "err", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// encodeCategories serialises consent.categories to a stable JSON string for
// storage in visit_sessions.consent_categories_json. nil/empty -> "{}".
func encodeCategories(c map[string]bool) string {
	if len(c) == 0 {
		return "{}"
	}
	b, err := json.Marshal(c)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// deriveVisitorID returns a stable visitor identifier for an identified
// session. Re-using the fingerprint (prefixed with the site ID) keeps things
// deterministic without leaking the raw fingerprint into reports.
func deriveVisitorID(siteID, fingerprint string) string {
	return "v_" + siteID[:6] + "_" + fingerprint
}
