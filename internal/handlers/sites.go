package handlers

import (
	"net/http"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/agent"
	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/store"
)

type SiteHandler struct {
	cfg        *config.Config
	queries    *store.Queries
	guardrails *agent.GuardrailEngine
}

func NewSiteHandler(cfg *config.Config, queries *store.Queries) *SiteHandler {
	return &SiteHandler{
		cfg:        cfg,
		queries:    queries,
		guardrails: agent.NewGuardrailEngine(queries),
	}
}

func (h *SiteHandler) List(w http.ResponseWriter, r *http.Request) {
	sites, err := h.queries.ListSites(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list sites")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sites": sites})
}

func (h *SiteHandler) Get(w http.ResponseWriter, r *http.Request) {
	site, err := h.queries.GetSiteByID(r.Context(), urlParam(r, "siteID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}
	writeJSON(w, http.StatusOK, site)
}

type createSiteRequest struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Domain         string `json:"domain"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	BgColor        string `json:"bg_color"`
	TextColor      string `json:"text_color"`
	FontHeading    string `json:"font_heading"`
	FontBody       string `json:"font_body"`
	Lang           string `json:"lang"`
}

func (h *SiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSiteRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Name == "" || req.Slug == "" {
		writeError(w, http.StatusBadRequest, "Name and slug are required")
		return
	}

	// Defaults
	if req.PrimaryColor == "" {
		req.PrimaryColor = "#D4AF37"
	}
	if req.SecondaryColor == "" {
		req.SecondaryColor = "#935FA7"
	}
	if req.BgColor == "" {
		req.BgColor = "#FFFFFF"
	}
	if req.TextColor == "" {
		req.TextColor = "#1A1A1A"
	}
	if req.FontHeading == "" {
		req.FontHeading = "Inter"
	}
	if req.FontBody == "" {
		req.FontBody = "Inter"
	}
	if req.Lang == "" {
		req.Lang = "en"
	}

	id := newID()
	slug := strings.ToLower(strings.ReplaceAll(req.Slug, " ", "-"))

	if err := h.queries.CreateSite(r.Context(), store.CreateSiteParams{
		ID:             id,
		Name:           req.Name,
		Slug:           slug,
		Domain:         req.Domain,
		PrimaryColor:   req.PrimaryColor,
		SecondaryColor: req.SecondaryColor,
		BgColor:        req.BgColor,
		TextColor:      req.TextColor,
		FontHeading:    req.FontHeading,
		FontBody:       req.FontBody,
		Lang:           req.Lang,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "A site with this slug already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create site")
		return
	}

	// Seed defaults for the new site
	_ = h.guardrails.SeedDefaultGuardrails(r.Context(), id)
	_ = h.guardrails.SeedDefaultKnowledgebase(r.Context(), id)

	// Create default architecture
	_ = h.queries.UpsertSiteArchitecture(r.Context(), store.UpsertSiteArchitectureParams{
		ID:            newID(),
		SiteID:        id,
		StructureType: "soft-silo",
		MaxDepth:      3,
	})

	site, _ := h.queries.GetSiteByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, site)
}

func (h *SiteHandler) Update(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	existing, err := h.queries.GetSiteByID(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}

	var req struct {
		Name            *string `json:"name"`
		Slug            *string `json:"slug"`
		Domain          *string `json:"domain"`
		Status          *string `json:"status"`
		PrimaryColor    *string `json:"primary_color"`
		SecondaryColor  *string `json:"secondary_color"`
		BgColor         *string `json:"bg_color"`
		TextColor       *string `json:"text_color"`
		FontHeading     *string `json:"font_heading"`
		FontBody        *string `json:"font_body"`
		MetaTitle       *string `json:"meta_title"`
		MetaDescription *string `json:"meta_description"`
		OgImageID       *string `json:"og_image_id"`
		FaviconID       *string `json:"favicon_id"`
		Ga4ID           *string `json:"ga4_id"`
		UmamiID         *string `json:"umami_id"`
		UmamiURL        *string `json:"umami_url"`
		CookieproofDomain *string `json:"cookieproof_domain"`
		Lang            *string `json:"lang"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	params := store.UpdateSiteParams{
		Name:            existing.Name,
		Slug:            existing.Slug,
		Domain:          existing.Domain,
		Status:          existing.Status,
		PrimaryColor:    existing.PrimaryColor,
		SecondaryColor:  existing.SecondaryColor,
		BgColor:         existing.BgColor,
		TextColor:       existing.TextColor,
		FontHeading:     existing.FontHeading,
		FontBody:        existing.FontBody,
		MetaTitle:       existing.MetaTitle,
		MetaDescription: existing.MetaDescription,
		OgImageID:       existing.OgImageID,
		FaviconID:       existing.FaviconID,
		Ga4ID:           existing.Ga4ID,
		UmamiID:         existing.UmamiID,
		UmamiUrl:        existing.UmamiUrl,
		CookieproofDomain: existing.CookieproofDomain,
		Lang:            existing.Lang,
		ID:              siteID,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.Slug != nil {
		params.Slug = *req.Slug
	}
	if req.Domain != nil {
		params.Domain = *req.Domain
	}
	if req.Status != nil {
		params.Status = *req.Status
	}
	if req.PrimaryColor != nil {
		params.PrimaryColor = *req.PrimaryColor
	}
	if req.SecondaryColor != nil {
		params.SecondaryColor = *req.SecondaryColor
	}
	if req.BgColor != nil {
		params.BgColor = *req.BgColor
	}
	if req.TextColor != nil {
		params.TextColor = *req.TextColor
	}
	if req.FontHeading != nil {
		params.FontHeading = *req.FontHeading
	}
	if req.FontBody != nil {
		params.FontBody = *req.FontBody
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
	if req.FaviconID != nil {
		params.FaviconID = *req.FaviconID
	}
	if req.Ga4ID != nil {
		params.Ga4ID = *req.Ga4ID
	}
	if req.UmamiID != nil {
		params.UmamiID = *req.UmamiID
	}
	if req.UmamiURL != nil {
		params.UmamiUrl = *req.UmamiURL
	}
	if req.CookieproofDomain != nil {
		params.CookieproofDomain = *req.CookieproofDomain
	}
	if req.Lang != nil {
		params.Lang = *req.Lang
	}

	if err := h.queries.UpdateSite(r.Context(), params); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "A site with this slug already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update site")
		return
	}

	site, _ := h.queries.GetSiteByID(r.Context(), siteID)
	writeJSON(w, http.StatusOK, site)
}

func (h *SiteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if _, err := h.queries.GetSiteByID(r.Context(), siteID); err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}

	if err := h.queries.DeleteSite(r.Context(), siteID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete site")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
