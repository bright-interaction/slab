package handlers

import (
	"net/http"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/store"
)

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
	page, err := h.queries.GetPageByID(r.Context(), urlParam(r, "pageID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "Page not found")
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
	pageID := urlParam(r, "pageID")
	existing, err := h.queries.GetPageByID(r.Context(), pageID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Page not found")
		return
	}

	var req struct {
		Title           *string `json:"title"`
		Slug            *string `json:"slug"`
		Status          *string `json:"status"`
		MetaTitle       *string `json:"meta_title"`
		MetaDescription *string `json:"meta_description"`
		OgImageID       *string `json:"og_image_id"`
		Layout          *string `json:"layout"`
		SortOrder       *int64  `json:"sort_order"`
		ShowInNav       *bool   `json:"show_in_nav"`
		NavLabel        *string `json:"nav_label"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	params := store.UpdatePageParams{
		Title:           existing.Title,
		Slug:            existing.Slug,
		Status:          existing.Status,
		MetaTitle:       existing.MetaTitle,
		MetaDescription: existing.MetaDescription,
		OgImageID:       existing.OgImageID,
		Layout:          existing.Layout,
		SortOrder:       existing.SortOrder,
		ShowInNav:       existing.ShowInNav,
		NavLabel:        existing.NavLabel,
		ID:              pageID,
	}

	if req.Title != nil {
		params.Title = *req.Title
	}
	if req.Slug != nil {
		params.Slug = *req.Slug
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

	if err := h.queries.UpdatePage(r.Context(), params); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update page")
		return
	}

	page, _ := h.queries.GetPageByID(r.Context(), pageID)
	writeJSON(w, http.StatusOK, page)
}

func (h *PageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	pageID := urlParam(r, "pageID")
	if _, err := h.queries.GetPageByID(r.Context(), pageID); err != nil {
		writeError(w, http.StatusNotFound, "Page not found")
		return
	}

	if err := h.queries.DeletePage(r.Context(), pageID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete page")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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
