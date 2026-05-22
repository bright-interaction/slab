package builder

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// Sprint 3 (multilingual v1, 2026-05-22): helpers for emitting one
// HTML file per (page row, locale) combination using the page_locales
// + block_locales overlay tables. When site_locales is empty, the
// build falls back to the existing slug-prefix convention untouched
// (every existing site keeps building byte-identical output).

// localeBuildSpec describes one render of one page in one locale.
// Default locale renders at outSlug=page.Slug, lang=defaultLang.
// Additional locales render at outSlug="<lang>/<override-or-base>"
// and lang=the additional locale.
type localeBuildSpec struct {
	lang    string
	outSlug string
	page    store.Page
}

// loadPageLocaleOverlays returns a map of (locale -> overlay row) for
// the page. Used to walk locales when site_locales has any rows.
func loadPageLocaleOverlays(ctx context.Context, queries *store.Queries, pageID string) map[string]store.PageLocale {
	rows, err := queries.ListPageLocalesByPage(ctx, pageID)
	if err != nil {
		return nil
	}
	out := make(map[string]store.PageLocale, len(rows))
	for _, r := range rows {
		out[strings.ToLower(strings.TrimSpace(r.Locale))] = r
	}
	return out
}

// loadBlockLocaleOverlays returns a map keyed (block_id, locale) ->
// overlay row for every block on the page. One query covers them all.
func loadBlockLocaleOverlays(ctx context.Context, queries *store.Queries, pageID string) map[string]map[string]store.BlockLocale {
	rows, err := queries.ListBlockLocalesByPage(ctx, pageID)
	if err != nil {
		return nil
	}
	out := make(map[string]map[string]store.BlockLocale)
	for _, r := range rows {
		loc := strings.ToLower(strings.TrimSpace(r.Locale))
		if loc == "" {
			continue
		}
		if out[r.BlockID] == nil {
			out[r.BlockID] = make(map[string]store.BlockLocale)
		}
		out[r.BlockID][loc] = r
	}
	return out
}

// buildLocaleSpecs returns the list of per-locale renders to emit for
// one page. Returns one spec when site has no locales configured (the
// default render). Otherwise: one spec for the default locale at the
// base slug + one spec for every additional locale whose page_locales
// row has status='published' (locales without an overlay row are
// skipped so unpublished translations don't accidentally ship).
func buildLocaleSpecs(page store.Page, defaultLang string, additional []string, pageLocaleOverlays map[string]store.PageLocale) []localeBuildSpec {
	if len(additional) == 0 {
		return []localeBuildSpec{{lang: defaultLang, outSlug: page.Slug, page: page}}
	}
	specs := []localeBuildSpec{{lang: defaultLang, outSlug: page.Slug, page: page}}
	for _, lang := range additional {
		overlay, ok := pageLocaleOverlays[lang]
		if !ok {
			// No row for this locale on this page; do not emit.
			continue
		}
		if strings.ToLower(strings.TrimSpace(overlay.Status)) != "published" {
			continue
		}
		// Apply page-level overlay.
		localized := page
		if overlay.Title != "" {
			localized.Title = overlay.Title
		}
		if overlay.MetaTitle != "" {
			localized.MetaTitle = overlay.MetaTitle
		}
		if overlay.MetaDescription != "" {
			localized.MetaDescription = overlay.MetaDescription
		}
		base := strings.TrimSpace(overlay.SlugOverride)
		if base == "" {
			base = strings.TrimPrefix(page.Slug, "/")
		} else {
			base = strings.TrimPrefix(base, "/")
		}
		localized.Slug = strings.TrimSuffix(lang+"/"+base, "/")
		specs = append(specs, localeBuildSpec{lang: lang, outSlug: localized.Slug, page: localized})
	}
	return specs
}

// applyBlockLocaleOverlay returns a shallow copy of the block with
// data_json + is_visible replaced from the per-locale overlay row.
// When no overlay exists for the locale, returns the block unchanged.
// data_json from the overlay is a full JSON object (not a diff): the
// editor writes the entire localized content blob so the renderer
// doesn't have to merge field-by-field.
func applyBlockLocaleOverlay(bl store.Block, locale string, overlays map[string]map[string]store.BlockLocale) store.Block {
	if overlays == nil {
		return bl
	}
	byLocale, ok := overlays[bl.ID]
	if !ok {
		return bl
	}
	overlay, ok := byLocale[locale]
	if !ok {
		return bl
	}
	out := bl
	trimmed := strings.TrimSpace(overlay.DataJson)
	if trimmed != "" && trimmed != "{}" && trimmed != "null" {
		// Validate the overlay JSON parses; if not, fall back to base.
		var probe map[string]any
		if err := json.Unmarshal([]byte(trimmed), &probe); err == nil {
			out.DataJson = trimmed
		} else {
			slog.Warn("block_locales: invalid JSON, falling back to base", "block_id", bl.ID, "locale", locale, "err", err)
		}
	}
	out.IsVisible = overlay.IsVisible
	return out
}

// applyBlockLocaleOverlays returns a new slice with every block
// replaced by its locale-applied variant. Preserves order.
func applyBlockLocaleOverlays(blocks []store.Block, locale string, overlays map[string]map[string]store.BlockLocale) []store.Block {
	if overlays == nil || locale == "" {
		return blocks
	}
	out := make([]store.Block, len(blocks))
	for i, bl := range blocks {
		out[i] = applyBlockLocaleOverlay(bl, locale, overlays)
	}
	return out
}
