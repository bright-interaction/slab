// Package handlers: locales endpoint.
//
// Sprint 3 (multilingual v1, 2026-05-22): exposes site_locales,
// page_locales, and block_locales as REST surfaces so the admin SPA
// can configure the locale list and author per-locale overrides for
// pages + blocks. Backs the locale tabs strip on the page editor and
// the Settings -> Locales page.
//
// The base content rows in `pages` + `blocks` carry the default-locale
// content; per-locale variants live in the overlay tables. When the
// build emits the site, it walks site_locales, joins each page with
// its page_locale + block_locale overlays for that locale, and writes
// one HTML page per (page, locale) combination. Default locale renders
// at /<slug>; additional locales render at /<locale>/<slug> (or under
// the slug_override when the page_locale row sets one).

package handlers

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// localeRegex enforces BCP-47-lite locale codes the builder + nginx
// configs already support. Two-letter language, optional two-letter
// region (lowercase locale form, e.g. "sv", "sv-se", "en-gb").
var localeRegex = regexp.MustCompile(`^[a-z]{2}(-[a-z]{2})?$`)

type LocalesHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewLocalesHandler(cfg *config.Config, queries *store.Queries) *LocalesHandler {
	return &LocalesHandler{cfg: cfg, queries: queries}
}

// --- site_locales -----------------------------------------------------

type siteLocaleOut struct {
	Locale    string `json:"locale"`
	IsDefault bool   `json:"is_default"`
	SortOrder int64  `json:"sort_order"`
	UpdatedAt string `json:"updated_at"`
}

type siteLocaleInput struct {
	Locale    string `json:"locale"`
	IsDefault bool   `json:"is_default"`
	SortOrder int64  `json:"sort_order"`
}

func toSiteLocaleOut(row store.SiteLocale) siteLocaleOut {
	return siteLocaleOut{
		Locale:    row.Locale,
		IsDefault: row.IsDefault == 1,
		SortOrder: row.SortOrder,
		UpdatedAt: row.UpdatedAt,
	}
}

func normalizeLocaleCode(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (h *LocalesHandler) ListSiteLocales(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	rows, err := h.queries.ListSiteLocales(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list locales")
		return
	}
	out := make([]siteLocaleOut, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSiteLocaleOut(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"locales": out, "count": len(out)})
}

func (h *LocalesHandler) UpsertSiteLocale(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	var in siteLocaleInput
	if err := parseJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	locale := normalizeLocaleCode(in.Locale)
	if !localeRegex.MatchString(locale) {
		writeError(w, http.StatusBadRequest, "locale must be a 2-letter code (e.g. sv) or language-region pair (e.g. sv-se)")
		return
	}
	def := int64(0)
	if in.IsDefault {
		def = 1
	}
	if err := h.queries.UpsertSiteLocale(r.Context(), store.UpsertSiteLocaleParams{
		SiteID: siteID, Locale: locale, IsDefault: def, SortOrder: in.SortOrder,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save locale")
		return
	}
	// If this row becomes the default, clear is_default on every
	// other locale for the same site. Done in a second statement
	// because sqlc/sqlite doesn't combine well into one upsert when
	// the secondary rows need a sweep.
	if in.IsDefault {
		if err := h.queries.ClearDefaultSiteLocales(r.Context(), store.ClearDefaultSiteLocalesParams{
			SiteID: siteID, Locale: locale,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear other default locales")
			return
		}
	}
	row, err := h.queries.GetSiteLocale(r.Context(), store.GetSiteLocaleParams{SiteID: siteID, Locale: locale})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Locale not found after save")
		return
	}
	writeJSON(w, http.StatusOK, toSiteLocaleOut(row))
}

func (h *LocalesHandler) DeleteSiteLocale(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	locale := normalizeLocaleCode(urlParam(r, "locale"))
	if locale == "" {
		writeError(w, http.StatusBadRequest, "locale required")
		return
	}
	// Refuse to delete the default locale. Operators must pick a
	// different default first.
	if row, err := h.queries.GetSiteLocale(r.Context(), store.GetSiteLocaleParams{SiteID: siteID, Locale: locale}); err == nil && row.IsDefault == 1 {
		writeError(w, http.StatusConflict, "cannot delete the default locale; pick a different default first")
		return
	}
	if err := h.queries.DeleteSiteLocale(r.Context(), store.DeleteSiteLocaleParams{SiteID: siteID, Locale: locale}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete locale")
		return
	}
	// Cascade through page_locales + block_locales so removing a
	// locale from the site list also clears any in-progress
	// translations rather than leaving orphan overlay rows.
	_ = h.queries.DeletePageLocalesByLocale(r.Context(), store.DeletePageLocalesByLocaleParams{SiteID: siteID, Locale: locale})
	_ = h.queries.DeleteBlockLocalesByLocale(r.Context(), store.DeleteBlockLocalesByLocaleParams{SiteID: siteID, Locale: locale})
	w.WriteHeader(http.StatusNoContent)
}

// --- page_locales -----------------------------------------------------

type pageLocaleOut struct {
	PageID          string `json:"page_id"`
	Locale          string `json:"locale"`
	SlugOverride    string `json:"slug_override"`
	Title           string `json:"title"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	Status          string `json:"status"`
	UpdatedAt       string `json:"updated_at"`
}

type pageLocaleInput struct {
	SlugOverride    string `json:"slug_override"`
	Title           string `json:"title"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	Status          string `json:"status"`
}

func toPageLocaleOut(row store.PageLocale) pageLocaleOut {
	return pageLocaleOut{
		PageID:          row.PageID,
		Locale:          row.Locale,
		SlugOverride:    row.SlugOverride,
		Title:           row.Title,
		MetaTitle:       row.MetaTitle,
		MetaDescription: row.MetaDescription,
		Status:          row.Status,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (h *LocalesHandler) verifyPageInSite(r *http.Request, pageID string) bool {
	siteID := urlParam(r, "siteID")
	p, err := h.queries.GetPageByID(r.Context(), pageID)
	if err != nil {
		return false
	}
	return p.SiteID == siteID
}

func (h *LocalesHandler) ListPageLocales(w http.ResponseWriter, r *http.Request) {
	pageID := urlParam(r, "pageID")
	if !h.verifyPageInSite(r, pageID) {
		writeError(w, http.StatusNotFound, "Page not found")
		return
	}
	rows, err := h.queries.ListPageLocalesByPage(r.Context(), pageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list page locales")
		return
	}
	out := make([]pageLocaleOut, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPageLocaleOut(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"page_locales": out, "count": len(out)})
}

func (h *LocalesHandler) UpsertPageLocale(w http.ResponseWriter, r *http.Request) {
	pageID := urlParam(r, "pageID")
	locale := normalizeLocaleCode(urlParam(r, "locale"))
	if !h.verifyPageInSite(r, pageID) {
		writeError(w, http.StatusNotFound, "Page not found")
		return
	}
	if !localeRegex.MatchString(locale) {
		writeError(w, http.StatusBadRequest, "locale invalid")
		return
	}
	var in pageLocaleInput
	if err := parseJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if status == "" {
		status = "draft"
	}
	if status != "draft" && status != "published" && status != "archived" {
		writeError(w, http.StatusBadRequest, "status must be draft|published|archived")
		return
	}
	if err := h.queries.UpsertPageLocale(r.Context(), store.UpsertPageLocaleParams{
		PageID:          pageID,
		Locale:          locale,
		SlugOverride:    strings.TrimSpace(in.SlugOverride),
		Title:           strings.TrimSpace(in.Title),
		MetaTitle:       strings.TrimSpace(in.MetaTitle),
		MetaDescription: strings.TrimSpace(in.MetaDescription),
		Status:          status,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save page locale")
		return
	}
	row, err := h.queries.GetPageLocale(r.Context(), store.GetPageLocaleParams{PageID: pageID, Locale: locale})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Locale not found after save")
		return
	}
	writeJSON(w, http.StatusOK, toPageLocaleOut(row))
}

func (h *LocalesHandler) DeletePageLocale(w http.ResponseWriter, r *http.Request) {
	pageID := urlParam(r, "pageID")
	locale := normalizeLocaleCode(urlParam(r, "locale"))
	if !h.verifyPageInSite(r, pageID) {
		writeError(w, http.StatusNotFound, "Page not found")
		return
	}
	if err := h.queries.DeletePageLocale(r.Context(), store.DeletePageLocaleParams{PageID: pageID, Locale: locale}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete page locale")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- block_locales ----------------------------------------------------

type blockLocaleOut struct {
	BlockID   string `json:"block_id"`
	Locale    string `json:"locale"`
	DataJson  string `json:"data_json"`
	IsVisible bool   `json:"is_visible"`
	UpdatedAt string `json:"updated_at"`
}

type blockLocaleInput struct {
	DataJson  string `json:"data_json"`
	IsVisible *bool  `json:"is_visible,omitempty"`
}

func toBlockLocaleOut(row store.BlockLocale) blockLocaleOut {
	return blockLocaleOut{
		BlockID:   row.BlockID,
		Locale:    row.Locale,
		DataJson:  row.DataJson,
		IsVisible: row.IsVisible == 1,
		UpdatedAt: row.UpdatedAt,
	}
}

func (h *LocalesHandler) verifyBlockInSite(r *http.Request, blockID string) bool {
	siteID := urlParam(r, "siteID")
	bl, err := h.queries.GetBlockByID(r.Context(), blockID)
	if err != nil {
		return false
	}
	p, err := h.queries.GetPageByID(r.Context(), bl.PageID)
	if err != nil {
		return false
	}
	return p.SiteID == siteID
}

func (h *LocalesHandler) ListBlockLocales(w http.ResponseWriter, r *http.Request) {
	blockID := urlParam(r, "blockID")
	if !h.verifyBlockInSite(r, blockID) {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}
	rows, err := h.queries.ListBlockLocalesByBlock(r.Context(), blockID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list block locales")
		return
	}
	out := make([]blockLocaleOut, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBlockLocaleOut(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"block_locales": out, "count": len(out)})
}

// ListBlockLocalesByPage returns every block_locale overlay for every
// block on a given page in one round trip. Used by the per-locale
// page editor to avoid an N+1 fetch over N blocks.
func (h *LocalesHandler) ListBlockLocalesByPage(w http.ResponseWriter, r *http.Request) {
	pageID := urlParam(r, "pageID")
	if !h.verifyPageInSite(r, pageID) {
		writeError(w, http.StatusNotFound, "Page not found")
		return
	}
	rows, err := h.queries.ListBlockLocalesByPage(r.Context(), pageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list block locales")
		return
	}
	out := make([]blockLocaleOut, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBlockLocaleOut(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"block_locales": out, "count": len(out)})
}

func (h *LocalesHandler) UpsertBlockLocale(w http.ResponseWriter, r *http.Request) {
	blockID := urlParam(r, "blockID")
	locale := normalizeLocaleCode(urlParam(r, "locale"))
	if !h.verifyBlockInSite(r, blockID) {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}
	if !localeRegex.MatchString(locale) {
		writeError(w, http.StatusBadRequest, "locale invalid")
		return
	}
	var in blockLocaleInput
	if err := parseJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	dataJSON := strings.TrimSpace(in.DataJson)
	if dataJSON == "" {
		dataJSON = "{}"
	}
	visible := int64(1)
	if in.IsVisible != nil && !*in.IsVisible {
		visible = 0
	}
	if err := h.queries.UpsertBlockLocale(r.Context(), store.UpsertBlockLocaleParams{
		BlockID:   blockID,
		Locale:    locale,
		DataJson:  dataJSON,
		IsVisible: visible,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save block locale")
		return
	}
	row, err := h.queries.GetBlockLocale(r.Context(), store.GetBlockLocaleParams{BlockID: blockID, Locale: locale})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Locale not found after save")
		return
	}
	writeJSON(w, http.StatusOK, toBlockLocaleOut(row))
}

func (h *LocalesHandler) DeleteBlockLocale(w http.ResponseWriter, r *http.Request) {
	blockID := urlParam(r, "blockID")
	locale := normalizeLocaleCode(urlParam(r, "locale"))
	if !h.verifyBlockInSite(r, blockID) {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}
	if err := h.queries.DeleteBlockLocale(r.Context(), store.DeleteBlockLocaleParams{BlockID: blockID, Locale: locale}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete block locale")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
