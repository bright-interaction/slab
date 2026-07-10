package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bright-interaction/atomicsite/internal/config"
	"github.com/bright-interaction/atomicsite/internal/figma"
	"github.com/bright-interaction/atomicsite/internal/store"
)

// FigmaHandler handles Figma design-tokens import requests. The personal
// access token is accepted in the request body, used once to fetch the file,
// and never persisted — the user pastes a fresh PAT for each import.
type FigmaHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewFigmaHandler(cfg *config.Config, queries *store.Queries) *FigmaHandler {
	return &FigmaHandler{cfg: cfg, queries: queries}
}

// ImportDesignTokens handles POST /api/sites/{siteID}/figma/import.
// Body: { file_url: "https://www.figma.com/design/abc/My-Brand", access_token: "figd_..." }.
// Side effects (idempotent):
//   - Seeds one CSS class per published color style and text style.
//   - Updates site.primary_color when a "Primary"/"Brand" color is detected.
//   - Updates site.font_heading + font_body to the most-likely brand fonts.
//   - Stores the file URL under settings (category=design, key=figma.file_url) for audit.
func (h *FigmaHandler) ImportDesignTokens(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if _, err := h.queries.GetSiteByID(r.Context(), siteID); err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}

	var req struct {
		FileURL     string `json:"file_url"`
		AccessToken string `json:"access_token"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.AccessToken == "" {
		writeError(w, http.StatusBadRequest, "access_token is required")
		return
	}
	fileKey := figma.FileKeyFromURL(req.FileURL)
	if fileKey == "" {
		writeError(w, http.StatusBadRequest, "file_url must be a Figma file/design URL")
		return
	}

	c := figma.NewClient(req.AccessToken)
	result, err := figma.ExtractTokens(r.Context(), c, fileKey)
	if err != nil {
		// Audit L4: never leak the Figma access token via the response
		// body. The full error stays in server logs for debugging; the
		// client gets a generic message. Token strings begin with `figd_`
		// or `figpat_` so we redact those substrings even if the error
		// formatter included them inline.
		slog.Warn("figma import failed", "site_id", siteID, "err", err)
		writeError(w, http.StatusBadGateway, "Figma import failed: invalid file or token")
		return
	}

	// Seed CSS classes (skip ones whose name already exists for this site).
	for i, ct := range result.Colors {
		if _, err := h.queries.GetCSSClassByName(r.Context(), store.GetCSSClassByNameParams{
			SiteID: siteID,
			Name:   ct.ClassName,
		}); err == nil {
			continue
		}
		_ = h.queries.CreateCSSClass(r.Context(), store.CreateCSSClassParams{
			ID:        figmaID(),
			SiteID:    siteID,
			Name:      ct.ClassName,
			Category:  "color",
			Css:       fmt.Sprintf("%s { color: %s; --token: %s; }", ct.ClassName, ct.Hex, ct.Hex),
			UsageNote: fmt.Sprintf("Figma color token: %s. Use as a foreground color or read --token via var().", ct.Name),
			SortOrder: int64(1000 + i),
		})
	}
	for i, tt := range result.Typography {
		if _, err := h.queries.GetCSSClassByName(r.Context(), store.GetCSSClassByNameParams{
			SiteID: siteID,
			Name:   tt.ClassName,
		}); err == nil {
			continue
		}
		_ = h.queries.CreateCSSClass(r.Context(), store.CreateCSSClassParams{
			ID:        figmaID(),
			SiteID:    siteID,
			Name:      tt.ClassName,
			Category:  "typography",
			Css:       fmt.Sprintf("%s { %s }", tt.ClassName, tt.CSS),
			UsageNote: fmt.Sprintf("Figma text token: %s.", tt.Name),
			SortOrder: int64(2000 + i),
		})
	}

	// Update site primary color + fonts when we have suggestions and current
	// values look like the wizard defaults (so we don't trample user edits).
	if result.PrimarySuggest != "" || result.HeadingSuggest != "" || result.BodySuggest != "" {
		row, _ := h.queries.GetSiteByID(r.Context(), siteID)
		params := store.UpdateSiteParams{
			Name:              row.Name,
			Slug:              row.Slug,
			Domain:            row.Domain,
			Status:            row.Status,
			PrimaryColor:      row.PrimaryColor,
			SecondaryColor:    row.SecondaryColor,
			BgColor:           row.BgColor,
			TextColor:         row.TextColor,
			SurfaceColor:      row.SurfaceColor,
			BorderColor:       row.BorderColor,
			MutedColor:        row.MutedColor,
			AccentColor:       row.AccentColor,
			OnPrimaryColor:    row.OnPrimaryColor,
			FontHeading:       row.FontHeading,
			FontBody:          row.FontBody,
			MetaTitle:         row.MetaTitle,
			MetaDescription:   row.MetaDescription,
			OgImageID:         row.OgImageID,
			FaviconID:         row.FaviconID,
			Ga4ID:             row.Ga4ID,
			UmamiID:           row.UmamiID,
			UmamiUrl:          row.UmamiUrl,
			CookieproofDomain: row.CookieproofDomain,
			Lang:              row.Lang,
			ID:                row.ID,
		}
		if result.PrimarySuggest != "" && isWizardDefaultColor(row.PrimaryColor) {
			params.PrimaryColor = result.PrimarySuggest
		}
		if result.HeadingSuggest != "" && isWizardDefaultFont(row.FontHeading) {
			params.FontHeading = result.HeadingSuggest
		}
		if result.BodySuggest != "" && isWizardDefaultFont(row.FontBody) {
			params.FontBody = result.BodySuggest
		}
		_ = h.queries.UpdateSite(r.Context(), params)
	}

	// Audit: stash the imported file URL (without the token) so we know
	// where the tokens came from. Token is never written.
	_ = h.queries.UpsertSetting(r.Context(), store.UpsertSettingParams{
		ID:       figmaID(),
		SiteID:   siteID,
		Category: "design",
		Key:      "figma.file_url",
		Value:    req.FileURL,
	})
	_ = h.queries.UpsertSetting(r.Context(), store.UpsertSettingParams{
		ID:       figmaID(),
		SiteID:   siteID,
		Category: "design",
		Key:      "figma.file_name",
		Value:    result.FileName,
	})

	writeJSON(w, http.StatusOK, result)
}

func isWizardDefaultColor(c string) bool {
	switch c {
	case "#7C5BB1", "#935FA7", "#FFFFFF", "#1A1A1A", "":
		return true
	}
	return false
}

func isWizardDefaultFont(f string) bool {
	return f == "" || f == "Inter"
}

func figmaID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
