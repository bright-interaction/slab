package handlers

import (
	"database/sql"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/agent"
	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/starterkits"
	"github.com/brightinteraction/atomicsite/internal/store"
)

type SiteHandler struct {
	cfg        *config.Config
	queries    *store.Queries
	db         *sql.DB
	guardrails *agent.GuardrailEngine
}

func NewSiteHandler(cfg *config.Config, queries *store.Queries, db *sql.DB) *SiteHandler {
	return &SiteHandler{
		cfg:        cfg,
		queries:    queries,
		db:         db,
		guardrails: agent.NewGuardrailEngine(queries),
	}
}

var (
	slugPattern  = regexp.MustCompile(`^[a-z0-9-]+$`)
	colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// applyDefaultSiteSettings inserts the security and server best-practice settings
// for a freshly created site using the supplied Queries handle.
func applyDefaultSiteSettings(ctx context.Context, q *store.Queries, siteID string) error {
	defaults := []struct{ category, key, value string }{
		// Security headers
		{"security", "csp_enabled", "true"},
		{"security", "csp_policy", "auto"},
		{"security", "hsts_enabled", "true"},
		{"security", "hsts_max_age", "31536000"},
		{"security", "hsts_preload", "false"},
		{"security", "x_frame_options", "DENY"},
		{"security", "x_content_type", "nosniff"},
		{"security", "referrer_policy", "strict-origin-when-cross-origin"},
		{"security", "permissions_policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()"},
		{"security", "coop", "same-origin"},
		{"security", "corp", "same-origin"},
		// Nginx/server
		{"server", "gzip_enabled", "true"},
		{"server", "brotli_enabled", "false"},
		{"server", "cache_static_max_age", "31536000"},
		{"server", "cache_html_max_age", "3600"},
		{"server", "rate_limit_enabled", "false"},
		{"server", "rate_limit_rps", "10"},
	}
	for _, d := range defaults {
		if err := q.UpsertSetting(ctx, store.UpsertSettingParams{
			ID:       newID(),
			SiteID:   siteID,
			Category: d.category,
			Key:      d.key,
			Value:    d.value,
		}); err != nil {
			return err
		}
	}
	return nil
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
	_ = agent.SeedDefaultGuardrailsWith(r.Context(), h.queries, id)
	_ = agent.SeedDefaultKnowledgebaseWith(r.Context(), h.queries, id)

	// Create default architecture
	_ = h.queries.UpsertSiteArchitecture(r.Context(), store.UpsertSiteArchitectureParams{
		ID:            newID(),
		SiteID:        id,
		StructureType: "soft-silo",
		MaxDepth:      3,
	})

	// Seed default security + server settings (best-practice defaults).
	// All toggles can be flipped via PATCH /api/sites/{id}/settings.
	_ = applyDefaultSiteSettings(r.Context(), h.queries, id)

	site, _ := h.queries.GetSiteByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, site)
}

// seedSiteRequest is the wizard payload for POST /api/sites/seed.
type seedSiteRequest struct {
	Type string `json:"type"`
	Info struct {
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		Domain       string `json:"domain"`
		Lang         string `json:"lang"`
		BusinessName string `json:"business_name"`
		ContactEmail string `json:"contact_email"`
		Country      string `json:"country"`
	} `json:"info"`
	Structure struct {
		StructureType string `json:"structure_type"`
		MaxDepth      int64  `json:"max_depth"`
	} `json:"structure"`
	Silos []struct {
		Name       string `json:"name"`
		SlugPrefix string `json:"slug_prefix"`
	} `json:"silos"`
	Branding struct {
		PrimaryColor   string `json:"primary_color"`
		SecondaryColor string `json:"secondary_color"`
		BgColor        string `json:"bg_color"`
		TextColor      string `json:"text_color"`
		FontHeading    string `json:"font_heading"`
		FontBody       string `json:"font_body"`
	} `json:"branding"`
	StarterKit string `json:"starter_kit,omitempty"`
}

// validateSeedRequest checks the wizard payload and applies defaults.
// It returns a user-facing error message if validation fails.
func (req *seedSiteRequest) normalize() error {
	// Type
	allowedTypes := map[string]bool{
		"b2b": true, "b2c": true, "personal": true,
		"blog": true, "agency": true, "one-pager": true,
	}
	if req.Type == "" {
		req.Type = "b2b"
	}
	if !allowedTypes[req.Type] {
		return errors.New("invalid type")
	}

	// Info
	req.Info.Name = strings.TrimSpace(req.Info.Name)
	req.Info.Slug = strings.ToLower(strings.TrimSpace(req.Info.Slug))
	if req.Info.Name == "" {
		return errors.New("info.name is required")
	}
	if req.Info.Slug == "" || len(req.Info.Slug) > 50 || !slugPattern.MatchString(req.Info.Slug) {
		return errors.New("info.slug must match ^[a-z0-9-]+$ (1-50 chars)")
	}
	if req.Info.Lang == "" {
		req.Info.Lang = "en"
	}

	// Structure
	allowedStructures := map[string]bool{
		"one-pager": true, "soft-silo": true, "hard-silo": true,
	}
	if req.Structure.StructureType == "" {
		req.Structure.StructureType = "soft-silo"
	}
	if !allowedStructures[req.Structure.StructureType] {
		return errors.New("structure.structure_type invalid")
	}
	if req.Structure.MaxDepth == 0 {
		req.Structure.MaxDepth = 3
	}
	if req.Structure.MaxDepth < 1 || req.Structure.MaxDepth > 3 {
		return errors.New("structure.max_depth must be 1, 2, or 3")
	}

	// Silos
	for i, s := range req.Silos {
		s.Name = strings.TrimSpace(s.Name)
		s.SlugPrefix = strings.TrimSpace(s.SlugPrefix)
		if s.Name == "" || s.SlugPrefix == "" {
			return errors.New("each silo requires name and slug_prefix")
		}
		req.Silos[i] = s
	}

	// Branding defaults + validation
	if req.Branding.PrimaryColor == "" {
		req.Branding.PrimaryColor = "#D4AF37"
	}
	if req.Branding.SecondaryColor == "" {
		req.Branding.SecondaryColor = "#935FA7"
	}
	if req.Branding.BgColor == "" {
		req.Branding.BgColor = "#FFFFFF"
	}
	if req.Branding.TextColor == "" {
		req.Branding.TextColor = "#1A1A1A"
	}
	if req.Branding.FontHeading == "" {
		req.Branding.FontHeading = "Inter"
	}
	if req.Branding.FontBody == "" {
		req.Branding.FontBody = "Inter"
	}
	for _, c := range []string{
		req.Branding.PrimaryColor, req.Branding.SecondaryColor,
		req.Branding.BgColor, req.Branding.TextColor,
	} {
		if !colorPattern.MatchString(c) {
			return errors.New("branding colors must match ^#[0-9a-fA-F]{6}$")
		}
	}

	// Starter kit (optional). If set, must reference a registered kit ID.
	req.StarterKit = strings.TrimSpace(req.StarterKit)
	if req.StarterKit != "" {
		if _, ok := starterkits.Default.Get(req.StarterKit); !ok {
			return errors.New("starter_kit references an unknown kit")
		}
	}

	return nil
}

// Seed creates a fully provisioned site (sites + profile + architecture + silos +
// guardrails + knowledgebase + global blocks + essential pages + settings) inside a
// single transaction. Used by the Phase 7 onboarding wizard.
func (h *SiteHandler) Seed(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)

	var req seedSiteRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := req.normalize(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("seed: begin tx", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)
	siteID := newID()

	// 1. sites row
	if err := qtx.CreateSite(ctx, store.CreateSiteParams{
		ID:             siteID,
		Name:           req.Info.Name,
		Slug:           req.Info.Slug,
		Domain:         req.Info.Domain,
		PrimaryColor:   req.Branding.PrimaryColor,
		SecondaryColor: req.Branding.SecondaryColor,
		BgColor:        req.Branding.BgColor,
		TextColor:      req.Branding.TextColor,
		FontHeading:    req.Branding.FontHeading,
		FontBody:       req.Branding.FontBody,
		Lang:           req.Info.Lang,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "A site with this slug already exists")
			return
		}
		slog.Error("seed: create site", "error", err, "slug", req.Info.Slug)
		writeError(w, http.StatusInternalServerError, "Failed to create site")
		return
	}

	// 2. site_profiles row
	if err := qtx.UpsertSiteProfile(ctx, store.UpsertSiteProfileParams{
		ID:           newID(),
		SiteID:       siteID,
		BusinessName: sanitizeLine(req.Info.BusinessName),
		Country:      sanitizeLine(req.Info.Country),
		ContactEmail: strings.TrimSpace(req.Info.ContactEmail),
	}); err != nil {
		slog.Error("seed: upsert profile", "error", err, "site_id", siteID)
		writeError(w, http.StatusInternalServerError, "Failed to create profile")
		return
	}

	// 3. site_architecture row
	if err := qtx.UpsertSiteArchitecture(ctx, store.UpsertSiteArchitectureParams{
		ID:            newID(),
		SiteID:        siteID,
		StructureType: req.Structure.StructureType,
		MaxDepth:      req.Structure.MaxDepth,
	}); err != nil {
		slog.Error("seed: upsert architecture", "error", err, "site_id", siteID)
		writeError(w, http.StatusInternalServerError, "Failed to create architecture")
		return
	}

	// 4. site_silos rows
	for i, s := range req.Silos {
		if err := qtx.CreateSilo(ctx, store.CreateSiloParams{
			ID:         newID(),
			SiteID:     siteID,
			Name:       s.Name,
			SlugPrefix: s.SlugPrefix,
			SortOrder:  int64(i),
		}); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				writeError(w, http.StatusBadRequest, "Duplicate silo slug_prefix")
				return
			}
			slog.Error("seed: create silo", "error", err, "site_id", siteID)
			writeError(w, http.StatusInternalServerError, "Failed to create silo")
			return
		}
	}

	// 5. Phase 2 auto-seeding (guardrails, knowledgebase, global blocks, essential pages, settings)
	if err := agent.SeedDefaultGuardrailsWith(ctx, qtx, siteID); err != nil {
		slog.Error("seed: guardrails", "error", err, "site_id", siteID)
		writeError(w, http.StatusInternalServerError, "Failed to seed guardrails")
		return
	}
	if err := agent.SeedDefaultKnowledgebaseWith(ctx, qtx, siteID); err != nil {
		slog.Error("seed: knowledgebase", "error", err, "site_id", siteID)
		writeError(w, http.StatusInternalServerError, "Failed to seed knowledgebase")
		return
	}
	if err := agent.SeedDefaultGlobalBlocksWith(ctx, qtx, siteID); err != nil {
		slog.Error("seed: global blocks", "error", err, "site_id", siteID)
		writeError(w, http.StatusInternalServerError, "Failed to seed global blocks")
		return
	}
	if err := agent.SeedEssentialPagesWith(ctx, qtx, siteID); err != nil {
		slog.Error("seed: essential pages", "error", err, "site_id", siteID)
		writeError(w, http.StatusInternalServerError, "Failed to seed essential pages")
		return
	}
	if err := applyDefaultSiteSettings(ctx, qtx, siteID); err != nil {
		slog.Error("seed: settings", "error", err, "site_id", siteID)
		writeError(w, http.StatusInternalServerError, "Failed to seed settings")
		return
	}

	// 6. Optional starter kit. If set, apply inside the same tx so a failure
	// rolls back everything we just created.
	if req.StarterKit != "" {
		kit, ok := starterkits.Default.Get(req.StarterKit)
		if !ok {
			// Should never happen because normalize() validated this, but
			// guard against a registry mutation between validate and apply.
			slog.Error("seed: starter kit disappeared", "kit", req.StarterKit, "site_id", siteID)
			writeError(w, http.StatusInternalServerError, "Failed to apply starter kit")
			return
		}
		if err := kit.Apply(ctx, qtx, siteID); err != nil {
			slog.Error("seed: apply starter kit", "error", err, "kit", req.StarterKit, "site_id", siteID)
			writeError(w, http.StatusInternalServerError, "Failed to apply starter kit")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("seed: commit", "error", err, "site_id", siteID)
		writeError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"site_id": siteID})
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

// ListStarterKits returns the catalog of registered starter kits for the
// onboarding wizard. Public endpoint: only kit metadata is exposed and the
// wizard fetches it before any site exists / before login. Concrete kit
// content lives in code.
func (h *SiteHandler) ListStarterKits(w http.ResponseWriter, r *http.Request) {
	kits := starterkits.Default.List()
	out := make([]map[string]any, 0, len(kits))
	for _, k := range kits {
		targets := k.TargetSiteTypes()
		if targets == nil {
			targets = []string{}
		}
		out = append(out, map[string]any{
			"id":                k.ID(),
			"name":              k.Name(),
			"description":       k.Description(),
			"target_site_types": targets,
		})
	}
	writeJSON(w, http.StatusOK, out)
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
