package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/bright-interaction/slab/internal/store"
)

// TestPreviewTokenManager_MintReturnsUniqueTokens locks the lifecycle
// contract: each Mint returns a fresh 64-char hex token, and the in-memory
// map grows by exactly one per call. Belt for the loopback flow chromedp
// relies on.
func TestPreviewTokenManager_MintReturnsUniqueTokens(t *testing.T) {
	_, q := setupDeployTestDB(t)
	m := NewPreviewTokenManager(q)

	tok1, err := m.Mint("site-a", "page-a")
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	tok2, err := m.Mint("site-a", "page-a")
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if tok1 == tok2 {
		t.Errorf("expected unique tokens; got %q twice", tok1)
	}
	if len(tok1) != 64 || len(tok2) != 64 {
		t.Errorf("expected 64-char hex tokens; got %d / %d", len(tok1), len(tok2))
	}
	if m.ticketCount() != 2 {
		t.Errorf("expected 2 tickets in map; got %d", m.ticketCount())
	}
}

// TestPreviewTokenManager_ServeIsSingleUse confirms the security
// invariant: redeeming a token consumes it, so the second request 404s.
// Critical for the screenshot tool because a leaked token in any log
// becomes uninteresting after first use.
func TestPreviewTokenManager_ServeIsSingleUse(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "previewsite0000000000aaaa"
	seedSite(t, q, siteID)
	pageID := "previewpage000000000aaaa"
	if err := q.CreatePage(context.Background(), store.CreatePageParams{
		ID:        pageID,
		SiteID:    siteID,
		Title:     "Token Preview",
		Slug:      "tokenpreview",
		Layout:    "default",
		SortOrder: 0,
		ShowInNav: 0,
	}); err != nil {
		t.Fatalf("create page: %v", err)
	}

	m := NewPreviewTokenManager(q)
	tok, err := m.Mint(siteID, pageID)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/_preview/{token}", m.Serve)

	// First hit from loopback succeeds.
	req1 := httptest.NewRequest(http.MethodGet, "/_preview/"+tok, nil)
	req1.RemoteAddr = "127.0.0.1:54321"
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		body, _ := io.ReadAll(rec1.Result().Body)
		t.Fatalf("first serve: expected 200, got %d (body=%s)", rec1.Code, string(body))
	}

	// Second hit with the same token is 404 (consumed).
	req2 := httptest.NewRequest(http.MethodGet, "/_preview/"+tok, nil)
	req2.RemoteAddr = "127.0.0.1:54322"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Errorf("second serve: expected 404 (consumed); got %d", rec2.Code)
	}
}

// TestPreviewTokenManager_ServeRejectsNonLoopback proves the
// defense-in-depth gate: a request from a public IP, even with a valid
// token, returns 404. Belt-and-braces guard so the route can't leak a
// draft preview via a reverse-proxy misconfiguration.
func TestPreviewTokenManager_ServeRejectsNonLoopback(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "previewsite11111111aaaa1"
	seedSite(t, q, siteID)
	pageID := "previewpage1111111aaaa1"
	if err := q.CreatePage(context.Background(), store.CreatePageParams{
		ID:        pageID,
		SiteID:    siteID,
		Title:     "Token Preview Non-Loop",
		Slug:      "noloop",
		Layout:    "default",
		SortOrder: 0,
		ShowInNav: 0,
	}); err != nil {
		t.Fatalf("create page: %v", err)
	}
	m := NewPreviewTokenManager(q)
	tok, _ := m.Mint(siteID, pageID)

	r := chi.NewRouter()
	r.Get("/_preview/{token}", m.Serve)

	req := httptest.NewRequest(http.MethodGet, "/_preview/"+tok, nil)
	req.RemoteAddr = "203.0.113.7:54321" // TEST-NET-3, definitely not loopback
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		body, _ := io.ReadAll(rec.Result().Body)
		t.Errorf("non-loopback request: expected 404, got %d (body=%s)", rec.Code, string(body))
	}
	// Token must NOT have been consumed by the rejected request, so a
	// follow-up loopback hit still works.
	req2 := httptest.NewRequest(http.MethodGet, "/_preview/"+tok, nil)
	req2.RemoteAddr = "127.0.0.1:54323"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("post-rejection loopback hit: expected 200, got %d", rec2.Code)
	}
}

// TestPreviewTokenManager_ServeReturnsRenderedHTML asserts the response
// body is the full draft-preview document, not the raw block JSON.
// Locks the contract that the screenshot tool will see real HTML when
// it navigates chromedp to the loopback URL.
func TestPreviewTokenManager_ServeReturnsRenderedHTML(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "previewsite22222222bbbb2"
	seedSite(t, q, siteID)
	pageID := "previewpage2222222bbbb2"
	if err := q.CreatePage(context.Background(), store.CreatePageParams{
		ID:        pageID,
		SiteID:    siteID,
		Title:     "Rendered Body",
		Slug:      "rendered-body",
		Layout:    "default",
		SortOrder: 0,
		ShowInNav: 0,
	}); err != nil {
		t.Fatalf("create page: %v", err)
	}
	if err := q.CreateBlock(context.Background(), store.CreateBlockParams{
		ID:        "blk-hero-preview",
		PageID:    pageID,
		BlockType: "hero",
		DataJson:  `{"headline":"Receipts not promises","subheading":"Own the stack","cta_text":"Calculate","cta_url":"/calc"}`,
		StyleJson: "{}",
		IsVisible: 1,
		SortOrder: 10,
	}); err != nil {
		t.Fatalf("create hero: %v", err)
	}

	m := NewPreviewTokenManager(q)
	tok, _ := m.Mint(siteID, pageID)

	r := chi.NewRouter()
	r.Get("/_preview/{token}", m.Serve)

	req := httptest.NewRequest(http.MethodGet, "/_preview/"+tok, nil)
	req.RemoteAddr = "127.0.0.1:54324"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Result().Body)
	html := string(body)
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Errorf("expected full HTML document; first 200 chars: %s", html[:min(200, len(html))])
	}
	if !strings.Contains(html, "Receipts not promises") {
		t.Errorf("expected hero headline in rendered body")
	}
	if rec.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
		t.Errorf("expected noindex header; got %q", rec.Header().Get("X-Robots-Tag"))
	}
}

// TestPreviewTokenManager_ExpiresStaleTickets confirms the GC sweep
// removes expired tickets. Belt for the in-memory map staying bounded.
func TestPreviewTokenManager_ExpiresStaleTickets(t *testing.T) {
	_, q := setupDeployTestDB(t)
	m := NewPreviewTokenManager(q)

	// Manually inject an expired ticket via the test-only path: we can't
	// reach the internal map without exposing it, so use Mint then mutate
	// the ticket directly through unexported access. Skip if not viable.
	tok, _ := m.Mint("site-x", "page-x")
	m.mu.Lock()
	tt := m.tickets[tok]
	tt.expires = time.Now().Add(-1 * time.Second)
	m.tickets[tok] = tt
	m.mu.Unlock()

	// A fresh Mint triggers the GC sweep; the expired ticket should be
	// gone afterwards.
	_, _ = m.Mint("site-x", "page-y")
	if _, present := m.tickets[tok]; present {
		t.Errorf("expected expired ticket to be GC'd on next Mint")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
