package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/bright-interaction/slab/internal/agent"
	"github.com/bright-interaction/slab/internal/builder"
	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

func encodeWarnings(vs []agent.Violation) string {
	b, _ := json.Marshal(vs)
	return string(b)
}

// pageSlugSegment matches one URL segment of a page slug. Pages can be
// nested ("blog/post"), so the full slug is segments joined by "/". Each
// segment must be lowercase alnum + dashes/underscores; ".." is rejected
// outright. Empty slug ("" -> index page) is allowed and validated as a
// special case before the regex runs.
var pageSlugSegment = regexp.MustCompile(`^[a-z0-9][-a-z0-9_]{0,79}$`)

// validatePageSlug rejects path traversal and chars that would corrupt the
// builder's slugToFilePath. Allowed: empty (root index), "blog/post-name",
// "level-1/level-2/level-3". Rejected: "..", "../etc/passwd", trailing
// slashes, double slashes, control chars, anything not [a-z0-9-_].
func validatePageSlug(s string) error {
	if s == "" {
		return nil
	}
	if len(s) > 250 {
		return fmt.Errorf("page slug too long (max 250 chars)")
	}
	if strings.ContainsAny(s, "\r\n\t") {
		return fmt.Errorf("page slug contains control characters")
	}
	for _, seg := range strings.Split(s, "/") {
		if seg == "" {
			return fmt.Errorf("page slug has empty segment (leading/trailing/double slash)")
		}
		if seg == "." || seg == ".." {
			return fmt.Errorf("page slug segment %q is not allowed", seg)
		}
		if !pageSlugSegment.MatchString(seg) {
			return fmt.Errorf("page slug segment %q must match ^[a-z0-9][-a-z0-9_]{0,79}$", seg)
		}
	}
	return nil
}

type PageHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewPageHandler(cfg *config.Config, queries *store.Queries) *PageHandler {
	return &PageHandler{cfg: cfg, queries: queries}
}

func (h *PageHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	pages, err := h.queries.ListPagesBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list pages")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pages": pages})
}

func (h *PageHandler) Get(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	page, ok := pageInSite(r.Context(), h.queries, w, urlParam(r, "pageID"), siteID)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *PageHandler) Create(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")

	// Verify site exists
	if _, err := h.queries.GetSiteByID(r.Context(), siteID); err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}

	var req struct {
		Title     string `json:"title"`
		Slug      string `json:"slug"`
		Layout    string `json:"layout"`
		ShowInNav *bool  `json:"show_in_nav"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "Title is required")
		return
	}
	if req.Slug == "" {
		req.Slug = strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))
	}
	req.Slug = strings.Trim(strings.ToLower(req.Slug), "/")
	if err := validatePageSlug(req.Slug); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Layout == "" {
		req.Layout = "default"
	}

	showInNav := int64(1)
	if req.ShowInNav != nil && !*req.ShowInNav {
		showInNav = 0
	}

	// Get next sort order
	count, _ := h.queries.CountPagesBySite(r.Context(), siteID)

	id := newID()
	if err := h.queries.CreatePage(r.Context(), store.CreatePageParams{
		ID:        id,
		SiteID:    siteID,
		Title:     req.Title,
		Slug:      req.Slug,
		Layout:    req.Layout,
		SortOrder: count,
		ShowInNav: showInNav,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "A page with this slug already exists in this site")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create page")
		return
	}

	page, _ := h.queries.GetPageByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, page)
}

func (h *PageHandler) Update(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	pageID := urlParam(r, "pageID")
	existing, ok := pageInSite(r.Context(), h.queries, w, pageID, siteID)
	if !ok {
		return
	}

	var req struct {
		Title            *string `json:"title"`
		Slug             *string `json:"slug"`
		Status           *string `json:"status"`
		MetaTitle        *string `json:"meta_title"`
		MetaDescription  *string `json:"meta_description"`
		OgImageID        *string `json:"og_image_id"`
		Layout           *string `json:"layout"`
		SortOrder        *int64  `json:"sort_order"`
		ShowInNav        *bool   `json:"show_in_nav"`
		NavLabel         *string `json:"nav_label"`
		NoIndex          *bool   `json:"no_index"`
		CanonicalURL     *string `json:"canonical_url"`
		HideGlobalBlocks *bool   `json:"hide_global_blocks"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	params := store.UpdatePageParams{
		Title:            existing.Title,
		Slug:             existing.Slug,
		Status:           existing.Status,
		MetaTitle:        existing.MetaTitle,
		MetaDescription:  existing.MetaDescription,
		OgImageID:        existing.OgImageID,
		Layout:           existing.Layout,
		SortOrder:        existing.SortOrder,
		ShowInNav:        existing.ShowInNav,
		NavLabel:         existing.NavLabel,
		NoIndex:          existing.NoIndex,
		CanonicalUrl:     existing.CanonicalUrl,
		HideGlobalBlocks: existing.HideGlobalBlocks,
		ID:               pageID,
	}

	if req.Title != nil {
		params.Title = *req.Title
	}
	if req.Slug != nil {
		slug := strings.Trim(strings.ToLower(*req.Slug), "/")
		if err := validatePageSlug(slug); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		params.Slug = slug
	}
	if req.Status != nil {
		params.Status = *req.Status
	}
	if req.MetaTitle != nil {
		params.MetaTitle = *req.MetaTitle
	}
	if req.MetaDescription != nil {
		params.MetaDescription = *req.MetaDescription
	}
	if req.OgImageID != nil {
		params.OgImageID = *req.OgImageID
	}
	if req.Layout != nil {
		params.Layout = *req.Layout
	}
	if req.SortOrder != nil {
		params.SortOrder = *req.SortOrder
	}
	if req.ShowInNav != nil {
		if *req.ShowInNav {
			params.ShowInNav = 1
		} else {
			params.ShowInNav = 0
		}
	}
	if req.NavLabel != nil {
		params.NavLabel = *req.NavLabel
	}
	if req.NoIndex != nil {
		if *req.NoIndex {
			params.NoIndex = 1
		} else {
			params.NoIndex = 0
		}
	}
	if req.CanonicalURL != nil {
		params.CanonicalUrl = *req.CanonicalURL
	}
	if req.HideGlobalBlocks != nil {
		if *req.HideGlobalBlocks {
			params.HideGlobalBlocks = 1
		} else {
			params.HideGlobalBlocks = 0
		}
	}

	// Input-time SEO guardrails (mirrors Site Inspector eval engine).
	violations := agent.ValidatePageMeta(agent.PageMetaInput{
		Title:           params.Title,
		MetaTitle:       params.MetaTitle,
		MetaDescription: params.MetaDescription,
	})
	if agent.HasErrors(violations) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":      "Page meta failed guardrail validation",
			"violations": violations,
		})
		return
	}

	if err := h.queries.UpdatePage(r.Context(), params); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update page")
		return
	}

	page, _ := h.queries.GetPageByID(r.Context(), pageID)
	if len(violations) > 0 {
		w.Header().Set("X-Atomicsite-Warnings", encodeWarnings(violations))
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *PageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	pageID := urlParam(r, "pageID")
	if _, ok := pageInSite(r.Context(), h.queries, w, pageID, siteID); !ok {
		return
	}

	if err := h.queries.DeletePage(r.Context(), pageID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete page")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// PreviewSource returns the assembled .astro source for a single page so the
// admin can show "View source" without triggering a full build. Uses the
// same renderPage routine the build pipeline writes to disk; output is the
// canonical source-of-truth for what bun build will see.
func (h *PageHandler) PreviewSource(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	pageID := urlParam(r, "pageID")

	page, err := h.queries.GetPageByID(r.Context(), pageID)
	if err != nil || page.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Page not found")
		return
	}

	src, err := builder.RenderPagePreview(r.Context(), h.queries, siteID, pageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to render page")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"page_id": pageID,
		"slug":    page.Slug,
		"title":   page.Title,
		"astro":   src,
	})
}

func (h *PageHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Order []struct {
			ID        string `json:"id"`
			SortOrder int64  `json:"sort_order"`
		} `json:"order"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	for _, item := range req.Order {
		_ = h.queries.UpdatePageOrder(r.Context(), store.UpdatePageOrderParams{
			SortOrder: item.SortOrder,
			ID:        item.ID,
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
