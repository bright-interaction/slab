package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/brightinteraction/atomicsite/internal/config"
	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// trackRouter mirrors the production wiring: FingerprintMiddleware on the
// /t/* group, then the two POST handlers.
func trackRouter(cfg *config.Config, h *TrackHandler) chi.Router {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(authmw.FingerprintMiddleware(cfg))
		r.Post("/t/consent", h.Consent)
		r.Post("/t/pageview", h.PageView)
	})
	return r
}

// postBody POSTs JSON to the test router and returns the response.
func postBody(t *testing.T, r chi.Router, path string, body any, prevCookies []*http.Cookie) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent/1.0")
	req.Header.Set("Accept-Language", "en-US")
	req.RemoteAddr = "203.0.113.10:54321"
	for _, c := range prevCookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Result()
}

// TestTrackHandler_Consent_AnonymousSession exercises the path where the
// posted consent has no analytics opt-in: a session row should be upserted
// but identified_at stays empty.
func TestTrackHandler_Consent_AnonymousSession(t *testing.T) {
	sqlDB, q := setupDeployTestDB(t)
	seedSiteWithSafeID(t, q, testSiteID)

	cfg := &config.Config{
		BaseURL:       "http://localhost:8080",
		AnalyticsSalt: "test-salt",
	}
	h := NewTrackHandler(cfg, q, sqlDB)
	r := trackRouter(cfg, h)

	resp := postBody(t, r, "/t/consent", map[string]any{
		"siteId": testSiteID,
		"consent": map[string]any{
			"version":    1,
			"timestamp":  1700000000,
			"method":     "reject-all",
			"categories": map[string]bool{"necessary": true, "analytics": false},
		},
		"page": "https://example.com/",
	}, nil)

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Cookie should be set.
	var fp string
	for _, c := range resp.Cookies() {
		if c.Name == authmw.FingerprintCookieName {
			fp = c.Value
		}
	}
	if fp == "" {
		t.Fatalf("expected fingerprint cookie to be set")
	}
	if len(fp) != 16 {
		t.Errorf("fingerprint not 16 hex chars: %q", fp)
	}

	// visit_session row should exist but be unidentified.
	sess, err := q.GetSessionByFingerprint(context.Background(), store.GetSessionByFingerprintParams{
		SiteID:      testSiteID,
		Fingerprint: fp,
	})
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess.IdentifiedAt != "" {
		t.Errorf("anonymous session should have empty identified_at, got %q", sess.IdentifiedAt)
	}
	if sess.PageCount != 1 {
		t.Errorf("expected page_count=1, got %d", sess.PageCount)
	}
}

// TestTrackHandler_Consent_LiftsToIdentified exercises the analytics-opt-in
// path: the second POST with the same fingerprint should set visitor_id +
// consent_method.
func TestTrackHandler_Consent_LiftsToIdentified(t *testing.T) {
	sqlDB, q := setupDeployTestDB(t)
	seedSiteWithSafeID(t, q, testSiteID)

	cfg := &config.Config{
		BaseURL:       "http://localhost:8080",
		AnalyticsSalt: "test-salt",
	}
	h := NewTrackHandler(cfg, q, sqlDB)
	r := trackRouter(cfg, h)

	body := map[string]any{
		"siteId": testSiteID,
		"consent": map[string]any{
			"version":    1,
			"timestamp":  1700000000,
			"method":     "accept-all",
			"categories": map[string]bool{"necessary": true, "analytics": true, "marketing": true},
		},
		"page": "https://example.com/",
	}

	// First POST: cookie minted, session identified in one shot.
	resp := postBody(t, r, "/t/consent", body, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	cookies := resp.Cookies()
	var fp string
	for _, c := range cookies {
		if c.Name == authmw.FingerprintCookieName {
			fp = c.Value
		}
	}
	if fp == "" {
		t.Fatalf("expected fingerprint cookie to be set")
	}

	sess, err := q.GetSessionByFingerprint(context.Background(), store.GetSessionByFingerprintParams{
		SiteID:      testSiteID,
		Fingerprint: fp,
	})
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess.IdentifiedAt == "" {
		t.Errorf("expected session to be identified after analytics opt-in")
	}
	if sess.VisitorID == "" {
		t.Errorf("expected visitor_id to be populated, got empty")
	}
	if sess.ConsentMethod != "accept-all" {
		t.Errorf("expected consent_method=accept-all, got %q", sess.ConsentMethod)
	}

	// Second POST with the same cookie -> page_count should bump but session stays identified.
	priorVisitorID := sess.VisitorID
	resp2 := postBody(t, r, "/t/consent", body, cookies)
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp2.StatusCode)
	}
	sess2, err := q.GetSessionByFingerprint(context.Background(), store.GetSessionByFingerprintParams{
		SiteID:      testSiteID,
		Fingerprint: fp,
	})
	if err != nil {
		t.Fatalf("get session 2: %v", err)
	}
	if sess2.PageCount != 2 {
		t.Errorf("expected page_count=2 after second POST, got %d", sess2.PageCount)
	}
	if sess2.VisitorID != priorVisitorID {
		t.Errorf("visitor_id changed across requests: %q -> %q", priorVisitorID, sess2.VisitorID)
	}
}

// TestTrackHandler_Consent_RejectsBadSiteID asserts the handler refuses to
// touch the DB when siteId is not a 24-hex value.
func TestTrackHandler_Consent_RejectsBadSiteID(t *testing.T) {
	sqlDB, q := setupDeployTestDB(t)
	cfg := &config.Config{BaseURL: "http://localhost:8080", AnalyticsSalt: "x"}
	h := NewTrackHandler(cfg, q, sqlDB)
	r := trackRouter(cfg, h)

	resp := postBody(t, r, "/t/consent", map[string]any{
		"siteId": "not-hex",
	}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// TestTrackHandler_Consent_RejectsUnknownSite asserts the handler 404s when
// the siteId is well-formed but doesn't map to any row.
func TestTrackHandler_Consent_RejectsUnknownSite(t *testing.T) {
	sqlDB, q := setupDeployTestDB(t)
	cfg := &config.Config{BaseURL: "http://localhost:8080", AnalyticsSalt: "x"}
	h := NewTrackHandler(cfg, q, sqlDB)
	r := trackRouter(cfg, h)

	resp := postBody(t, r, "/t/consent", map[string]any{
		"siteId": "bbbbbbbbbbbbbbbbbbbbbbbb",
	}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// TestTrackHandler_PageView records a visit_event with the session_id it
// stitches via the existing session row.
func TestTrackHandler_PageView_LinksToSession(t *testing.T) {
	sqlDB, q := setupDeployTestDB(t)
	seedSiteWithSafeID(t, q, testSiteID)
	cfg := &config.Config{BaseURL: "http://localhost:8080", AnalyticsSalt: "x"}
	h := NewTrackHandler(cfg, q, sqlDB)
	r := trackRouter(cfg, h)

	// Prime a session via consent.
	resp := postBody(t, r, "/t/consent", map[string]any{
		"siteId": testSiteID,
		"consent": map[string]any{
			"version": 1, "timestamp": 1700000000,
			"method":     "accept-all",
			"categories": map[string]bool{"necessary": true, "analytics": true},
		},
	}, nil)
	cookies := resp.Cookies()

	// Pageview should reuse the cookie + log the event.
	resp2 := postBody(t, r, "/t/pageview", map[string]any{
		"siteId":   testSiteID,
		"path":     "/about",
		"referrer": "https://google.com",
	}, cookies)
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp2.StatusCode)
	}

	rows, err := q.ListVisitsBySite(context.Background(), store.ListVisitsBySiteParams{
		SiteID: testSiteID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list visits: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 visit_event, got %d", len(rows))
	}
	if rows[0].Path != "/about" {
		t.Errorf("expected path=/about, got %q", rows[0].Path)
	}
	if rows[0].SessionID == "" {
		t.Errorf("expected session_id to be linked, got empty")
	}
}
