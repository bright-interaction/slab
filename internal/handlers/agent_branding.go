// Agent-side branding / fonts / design references API (Phase 12.9).
//
// Mirrors the admin endpoints so AI agents can do everything the human
// admin can do via the Branding UI. Auth via X-Agent-Key (the agent
// middleware already runs on /api/agent/*); writes require the "write"
// capability declared on the agent key.
//
// SiteID is read from the agent identity, not from the URL, so an agent
// scoped to site A cannot manipulate site B even if the URL targets it.

package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bright-interaction/slab/internal/store"
)

// --- Branding read / write ---

type agentBrandingPayload struct {
	PrimaryColor   *string `json:"primary_color,omitempty"`
	SecondaryColor *string `json:"secondary_color,omitempty"`
	BgColor        *string `json:"bg_color,omitempty"`
	TextColor      *string `json:"text_color,omitempty"`
	SurfaceColor   *string `json:"surface_color,omitempty"`
	BorderColor    *string `json:"border_color,omitempty"`
	MutedColor     *string `json:"muted_color,omitempty"`
	AccentColor    *string `json:"accent_color,omitempty"`
	OnPrimaryColor *string `json:"on_primary_color,omitempty"`
	FontHeading    *string `json:"font_heading,omitempty"`
	FontBody       *string `json:"font_body,omitempty"`
}

// GET /api/agent/branding -> current branding fields for the agent's site.
func (h *AgentHandler) GetBranding(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	site, err := h.queries.GetSiteByID(r.Context(), a.SiteID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}
	writeJSON(w, http.StatusOK, agentBrandingPayload{
		PrimaryColor:   strPtr(site.PrimaryColor),
		SecondaryColor: strPtr(site.SecondaryColor),
		BgColor:        strPtr(site.BgColor),
		TextColor:      strPtr(site.TextColor),
		SurfaceColor:   strPtr(site.SurfaceColor),
		BorderColor:    strPtr(site.BorderColor),
		MutedColor:     strPtr(site.MutedColor),
		AccentColor:    strPtr(site.AccentColor),
		OnPrimaryColor: strPtr(site.OnPrimaryColor),
		FontHeading:    strPtr(site.FontHeading),
		FontBody:       strPtr(site.FontBody),
	})
}

// PATCH /api/agent/branding -> partial update. Same hex / font validation
// as the admin endpoint via the underlying UpdateSite path.
func (h *AgentHandler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil || !requireWrite(w, a) {
		return
	}
	var req agentBrandingPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	row, err := h.queries.GetSiteByID(r.Context(), a.SiteID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}
	hexRE := colorPattern
	check := func(name string, v *string) bool {
		if v == nil {
			return true
		}
		if !hexRE.MatchString(*v) && *v != "" {
			writeError(w, http.StatusBadRequest, name+" must be #RRGGBB hex or empty")
			return false
		}
		return true
	}
	if !check("primary_color", req.PrimaryColor) || !check("secondary_color", req.SecondaryColor) ||
		!check("bg_color", req.BgColor) || !check("text_color", req.TextColor) ||
		!check("surface_color", req.SurfaceColor) || !check("border_color", req.BorderColor) ||
		!check("muted_color", req.MutedColor) || !check("accent_color", req.AccentColor) ||
		!check("on_primary_color", req.OnPrimaryColor) {
		return
	}

	params := store.UpdateSiteParams{
		Name: row.Name, Slug: row.Slug, Domain: row.Domain, Status: row.Status,
		PrimaryColor: row.PrimaryColor, SecondaryColor: row.SecondaryColor,
		BgColor: row.BgColor, TextColor: row.TextColor,
		SurfaceColor: row.SurfaceColor, BorderColor: row.BorderColor,
		MutedColor: row.MutedColor, AccentColor: row.AccentColor,
		OnPrimaryColor: row.OnPrimaryColor,
		FontHeading:    row.FontHeading, FontBody: row.FontBody,
		MetaTitle: row.MetaTitle, MetaDescription: row.MetaDescription,
		OgImageID: row.OgImageID, FaviconID: row.FaviconID,
		Ga4ID: row.Ga4ID, UmamiID: row.UmamiID, UmamiUrl: row.UmamiUrl,
		CookieproofDomain: row.CookieproofDomain, Lang: row.Lang,
		ID: row.ID,
	}
	apply := func(field *string, target *string) {
		if field != nil {
			*target = *field
		}
	}
	apply(req.PrimaryColor, &params.PrimaryColor)
	apply(req.SecondaryColor, &params.SecondaryColor)
	apply(req.BgColor, &params.BgColor)
	apply(req.TextColor, &params.TextColor)
	apply(req.SurfaceColor, &params.SurfaceColor)
	apply(req.BorderColor, &params.BorderColor)
	apply(req.MutedColor, &params.MutedColor)
	apply(req.AccentColor, &params.AccentColor)
	apply(req.OnPrimaryColor, &params.OnPrimaryColor)
	apply(req.FontHeading, &params.FontHeading)
	apply(req.FontBody, &params.FontBody)

	if err := h.queries.UpdateSite(r.Context(), params); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update branding")
		return
	}
	updated, _ := h.queries.GetSiteByID(r.Context(), a.SiteID)
	writeJSON(w, http.StatusOK, updated)
}

// --- Fonts ---

// GET /api/agent/fonts
func (h *AgentHandler) ListFonts(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	rows, _ := h.queries.ListSiteFonts(r.Context(), a.SiteID)
	writeJSON(w, http.StatusOK, map[string]any{"fonts": rows})
}

// POST /api/agent/fonts
// Multipart upload identical to the admin endpoint. Validates wOF2 magic
// header so the agent can't smuggle non-woff2 binaries.
func (h *AgentHandler) UploadFont(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil || !requireWrite(w, a) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, fontUploadMaxBytes+8192)
	if err := r.ParseMultipartForm(fontUploadMaxBytes); err != nil {
		writeError(w, http.StatusBadRequest, "Upload too large or malformed multipart body")
		return
	}
	familyName := strings.TrimSpace(r.FormValue("family_name"))
	if !validFamilyName.MatchString(familyName) {
		writeError(w, http.StatusBadRequest, "family_name must be 1 to 64 chars: letters, digits, spaces, hyphen, underscore")
		return
	}
	weight := 400
	if v := strings.TrimSpace(r.FormValue("weight")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 100 && n <= 900 {
			weight = n
		}
	}
	style := strings.TrimSpace(r.FormValue("style"))
	if style != "italic" {
		style = "normal"
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file field required")
		return
	}
	defer file.Close()
	body, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read uploaded file")
		return
	}
	if len(body) < 4 || !startsWith(body, woff2Magic) {
		writeError(w, http.StatusBadRequest, "Only woff2 fonts are accepted (signature 'wOF2' at offset 0)")
		return
	}
	id := newFontID()
	siteFontsDir := filepath.Join(h.cfg.FontsDir, a.SiteID)
	if err := os.MkdirAll(siteFontsDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to prepare fonts directory")
		return
	}
	outPath := filepath.Join(siteFontsDir, id+".woff2")
	if err := os.WriteFile(outPath, body, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to write font file")
		return
	}
	if err := h.queries.CreateSiteFont(r.Context(), store.CreateSiteFontParams{
		ID: id, SiteID: a.SiteID, FamilyName: familyName,
		Weight: int64(weight), Style: style, FilePath: outPath,
		FileSize: int64(len(body)), OriginalName: cleanFilename(header.Filename),
	}); err != nil {
		_ = os.Remove(outPath)
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "A font with this family/weight/style already exists for this site")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to record font")
		return
	}
	row, _ := h.queries.GetSiteFont(r.Context(), store.GetSiteFontParams{ID: id, SiteID: a.SiteID})
	writeJSON(w, http.StatusCreated, row)
}

// DELETE /api/agent/fonts/{fontID}
func (h *AgentHandler) DeleteFont(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil || !requireWrite(w, a) {
		return
	}
	fontID := urlParam(r, "fontID")
	row, err := h.queries.GetSiteFont(r.Context(), store.GetSiteFontParams{ID: fontID, SiteID: a.SiteID})
	if err != nil {
		writeError(w, http.StatusNotFound, "Font not found")
		return
	}
	if err := h.queries.DeleteSiteFont(r.Context(), store.DeleteSiteFontParams{ID: fontID, SiteID: a.SiteID}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete font row")
		return
	}
	if row.FilePath != "" {
		_ = os.Remove(row.FilePath)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Design references ---

// GET /api/agent/design-references
func (h *AgentHandler) ListDesignReferences(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	rows, _ := h.queries.ListDesignReferences(r.Context(), a.SiteID)
	writeJSON(w, http.StatusOK, map[string]any{"references": rows})
}

// POST /api/agent/design-references
// Body { url, label, ref_type }. Same fetch + bundle store as the admin endpoint.
func (h *AgentHandler) AddDesignReference(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil || !requireWrite(w, a) {
		return
	}
	var req struct {
		URL     string `json:"url"`
		Label   string `json:"label"`
		RefType string `json:"ref_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	owner, repo, branch, ok := parseGitHubURL(req.URL)
	if !ok {
		writeError(w, http.StatusBadRequest, "url must be a public GitHub repo URL (https://github.com/owner/repo)")
		return
	}
	if req.RefType == "" {
		req.RefType = "design-system"
	}
	switch req.RefType {
	case "design-system", "component-library", "reference-site":
	default:
		writeError(w, http.StatusBadRequest, "ref_type must be design-system, component-library, or reference-site")
		return
	}
	bundle, err := fetchGitHubBundle(r.Context(), owner, repo, branch)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	bundleBytes, _ := json.Marshal(bundle)
	id := newRefID()
	if err := h.queries.CreateDesignReference(r.Context(), store.CreateDesignReferenceParams{
		ID:          id,
		SiteID:      a.SiteID,
		Url:         (&url.URL{Scheme: "https", Host: "github.com", Path: "/" + owner + "/" + repo}).String(),
		Label:       strings.TrimSpace(req.Label),
		RefType:     req.RefType,
		FetchedJson: string(bundleBytes),
		FetchedAt:   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "This repo is already a design reference for the site")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to record design reference")
		return
	}
	row, _ := h.queries.GetDesignReference(r.Context(), store.GetDesignReferenceParams{ID: id, SiteID: a.SiteID})
	writeJSON(w, http.StatusCreated, row)
}

// DELETE /api/agent/design-references/{refID}
func (h *AgentHandler) DeleteDesignReference(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil || !requireWrite(w, a) {
		return
	}
	refID := urlParam(r, "refID")
	if _, err := h.queries.GetDesignReference(r.Context(), store.GetDesignReferenceParams{ID: refID, SiteID: a.SiteID}); err != nil {
		writeError(w, http.StatusNotFound, "Design reference not found")
		return
	}
	if err := h.queries.DeleteDesignReference(r.Context(), store.DeleteDesignReferenceParams{ID: refID, SiteID: a.SiteID}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete reference")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func strPtr(s string) *string {
	return &s
}
