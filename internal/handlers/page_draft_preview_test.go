package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

// TestPageDraftPreview_RendersFullPageHTML covers the happy path: a page
// with two visible blocks renders into a complete HTML document with
// noindex headers, both blocks' content, and the site CSS pipeline. The
// "preview" handler is the surface the in-app preview iframe loads via
// srcdoc, so the assertions are about what the iframe will actually see
// in the DOM.
func TestPageDraftPreview_RendersFullPageHTML(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "draftprevsite0000000aaaa"
	seedSite(t, q, siteID)

	pageID := "draftprevpage0000aaaa1111"
	if err := q.CreatePage(context.Background(), store.CreatePageParams{
		ID:        pageID,
		SiteID:    siteID,
		Title:     "Hello Preview",
		Slug:      "hello-preview",
		Layout:    "default",
		SortOrder: 0,
		ShowInNav: 0,
	}); err != nil {
		t.Fatalf("create page: %v", err)
	}

	if err := q.CreateBlock(context.Background(), store.CreateBlockParams{
		ID:        "blk-hero-001",
		PageID:    pageID,
		BlockType: "hero",
		DataJson:  `{"headline":"Stop renting your business","subheading":"Own it","cta_text":"Calculate","cta_url":"/calc"}`,
		StyleJson: "{}",
		IsVisible: 1,
		SortOrder: 10,
	}); err != nil {
		t.Fatalf("create hero block: %v", err)
	}
	if err := q.CreateBlock(context.Background(), store.CreateBlockParams{
		ID:        "blk-text-001",
		PageID:    pageID,
		BlockType: "text",
		DataJson:  `{"heading":"Why open source","text":"Receipts, not promises."}`,
		StyleJson: "{}",
		IsVisible: 1,
		SortOrder: 20,
	}); err != nil {
		t.Fatalf("create text block: %v", err)
	}

	h := NewPageHandler(&config.Config{}, q)
	r := chi.NewRouter()
	r.Get("/api/sites/{siteID}/pages/{pageID}/draft-preview", h.PageDraftPreview)

	req := httptest.NewRequest(http.MethodGet,
		"/api/sites/"+siteID+"/pages/"+pageID+"/draft-preview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status: got %d want 200, body=%s", resp.StatusCode, body)
	}

	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("content-type: %q (want text/html)", got)
	}
	if got := resp.Header.Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options: %q (want SAMEORIGIN)", got)
	}
	if got := resp.Header.Get("X-Robots-Tag"); !strings.Contains(got, "noindex") {
		t.Errorf("X-Robots-Tag: %q (want noindex)", got)
	}
	if got := resp.Header.Get("Cache-Control"); !strings.Contains(got, "no-store") {
		t.Errorf("Cache-Control: %q (want no-store)", got)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	wantSubstrings := []string{
		"<!DOCTYPE html>",
		`<html lang="en">`,
		`<meta name="robots" content="noindex, nofollow">`,
		`<title>Hello Preview · Preview</title>`,
		`<main>`,
		`Stop renting your business`,
		`Own it`,
		`Why open source`,
		`Receipts, not promises.`,
		`<style data-slab-preview="reset">`,
		`<style data-slab-preview="site">`,
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(html, s) {
			t.Errorf("preview missing %q\n--- response ---\n%s", s, html)
		}
	}
}

// TestPageDraftPreview_RejectsCrossSitePageID prevents the obvious abuse:
// the URL carries siteID + pageID separately, so we must verify the page
// row's SiteID matches before rendering. Without this check, a low-trust
// site member could read another site's pages just by guessing IDs.
func TestPageDraftPreview_RejectsCrossSitePageID(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteA := "draftprevsiteA000000aaaa"
	siteB := "draftprevsiteB000000bbbb"
	seedSite(t, q, siteA)
	seedSite(t, q, siteB)

	pageBID := "pagebelongstosite-b00aaaa"
	if err := q.CreatePage(context.Background(), store.CreatePageParams{
		ID:     pageBID,
		SiteID: siteB,
		Title:  "Site B page",
		Slug:   "b-page",
		Layout: "default",
	}); err != nil {
		t.Fatalf("create page: %v", err)
	}

	h := NewPageHandler(&config.Config{}, q)
	r := chi.NewRouter()
	r.Get("/api/sites/{siteID}/pages/{pageID}/draft-preview", h.PageDraftPreview)

	req := httptest.NewRequest(http.MethodGet,
		"/api/sites/"+siteA+"/pages/"+pageBID+"/draft-preview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("cross-site lookup: got %d want 404", w.Result().StatusCode)
	}
}

// TestPageDraftPreview_SkipsHiddenBlocks asserts is_visible=0 blocks are
// omitted from the rendered preview. Mirrors the production builder's
// behaviour so the preview stays an honest stand-in for what would ship.
func TestPageDraftPreview_SkipsHiddenBlocks(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "draftprevsite0000000visib"
	seedSite(t, q, siteID)

	pageID := "draftprevpage000000visib1"
	if err := q.CreatePage(context.Background(), store.CreatePageParams{
		ID: pageID, SiteID: siteID, Title: "Visibility test", Slug: "vis",
		Layout: "default",
	}); err != nil {
		t.Fatalf("create page: %v", err)
	}
	for i, vis := range []int64{1, 0, 1} {
		blockID := "blk-vis-" + string(rune('a'+i))
		dataKey := "VISIBLE-" + string(rune('A'+i))
		if vis == 0 {
			dataKey = "HIDDEN-" + string(rune('A'+i))
		}
		if err := q.CreateBlock(context.Background(), store.CreateBlockParams{
			ID:        blockID,
			PageID:    pageID,
			BlockType: "text",
			DataJson:  `{"heading":"` + dataKey + `","text":"body"}`,
			StyleJson: "{}",
			IsVisible: vis,
			SortOrder: int64(i * 10),
		}); err != nil {
			t.Fatalf("create block %d: %v", i, err)
		}
	}

	h := NewPageHandler(&config.Config{}, q)
	r := chi.NewRouter()
	r.Get("/api/sites/{siteID}/pages/{pageID}/draft-preview", h.PageDraftPreview)

	req := httptest.NewRequest(http.MethodGet,
		"/api/sites/"+siteID+"/pages/"+pageID+"/draft-preview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)

	if !strings.Contains(html, "VISIBLE-A") {
		t.Errorf("expected first visible block VISIBLE-A in output, missing")
	}
	if !strings.Contains(html, "VISIBLE-C") {
		t.Errorf("expected third visible block VISIBLE-C in output, missing")
	}
	if strings.Contains(html, "HIDDEN-B") {
		t.Errorf("hidden block leaked into preview: %s", html)
	}
}
