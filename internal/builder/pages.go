package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// RenderSingleBlock returns the rendered Astro source for a single block.
// Used by the admin "View source" feature so developers can see the exact
// code each block produces without triggering a full build. The output is
// the same string the build pipeline writes to disk for this block.
func RenderSingleBlock(ctx context.Context, queries *store.Queries, siteID, blockID string) (string, error) {
	block, err := queries.GetBlockByID(ctx, blockID)
	if err != nil {
		return "", fmt.Errorf("get block: %w", err)
	}
	components, _ := queries.ListComponentsBySite(ctx, siteID)
	componentNames := make(map[string]bool, len(components))
	for _, c := range components {
		componentNames[c.Name] = true
	}
	return renderBlock(block, componentNames), nil
}

// RenderPagePreview returns the rendered Astro source for a single page.
// Same output renderPageWithContext produces during a build, but
// materialised in-memory so the admin "View source" dialog can show the
// assembled page without triggering bun build. Bypasses status filters
// so drafts also render. Loads the full pageRenderContext so the preview
// reflects current meta-template + hreflang settings.
func RenderPagePreview(ctx context.Context, queries *store.Queries, siteID, pageID string) (string, error) {
	page, err := queries.GetPageByID(ctx, pageID)
	if err != nil {
		return "", fmt.Errorf("get page: %w", err)
	}
	if page.SiteID != siteID {
		return "", fmt.Errorf("page does not belong to site")
	}
	blocks, err := queries.ListBlocksByPage(ctx, pageID)
	if err != nil {
		return "", fmt.Errorf("list blocks: %w", err)
	}
	components, _ := queries.ListComponentsBySite(ctx, siteID)
	componentNames := make(map[string]bool, len(components))
	for _, c := range components {
		componentNames[c.Name] = true
	}
	site, err := queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return "", fmt.Errorf("get site: %w", err)
	}
	settings, _ := queries.ListSettingsBySite(ctx, siteID)
	sm := make(map[string]string, len(settings))
	for _, s := range settings {
		sm[s.Category+"."+s.Key] = s.Value
	}
	i18n, _ := LoadI18nConfig(ctx, queries, siteID)
	pageCtx := pageRenderContext{
		site:     site,
		i18n:     i18n,
		titleTpl: sm["seo.meta_title_template"],
		descTpl:  sm["seo.meta_description_template"],
	}
	return renderPageWithContext(page, blocks, componentNames, pageCtx), nil
}

// RenderPages generates .astro page files from published pages and their blocks.
func RenderPages(ctx context.Context, queries *store.Queries, siteID string, wsDir string) (int, error) {
	pages, err := queries.ListPublishedPagesBySite(ctx, siteID)
	if err != nil {
		return 0, fmt.Errorf("list pages: %w", err)
	}

	components, _ := queries.ListComponentsBySite(ctx, siteID)
	componentNames := make(map[string]bool)
	for _, c := range components {
		componentNames[c.Name] = true
	}

	site, err := queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return 0, fmt.Errorf("get site: %w", err)
	}

	settings, _ := queries.ListSettingsBySite(ctx, siteID)
	sm := make(map[string]string, len(settings))
	for _, s := range settings {
		sm[s.Category+"."+s.Key] = s.Value
	}

	i18n, err := LoadI18nConfig(ctx, queries, siteID)
	if err != nil {
		return 0, fmt.Errorf("load i18n config: %w", err)
	}

	pageCtx := pageRenderContext{
		site:      site,
		i18n:      i18n,
		titleTpl:  sm["seo.meta_title_template"],
		descTpl:   sm["seo.meta_description_template"],
	}

	for _, page := range pages {
		blocks, err := queries.ListBlocksByPage(ctx, page.ID)
		if err != nil {
			return 0, fmt.Errorf("list blocks for page %s: %w", page.Slug, err)
		}
		slog.Info("build: rendering page", "slug", page.Slug, "blocks", len(blocks))

		content := renderPageWithContext(page, blocks, componentNames, pageCtx)
		pagePath := slugToFilePath(page.Slug, wsDir)

		if err := WriteFile(pagePath, content); err != nil {
			return 0, fmt.Errorf("write page %s: %w", page.Slug, err)
		}
	}

	return len(pages), nil
}

// pageRenderContext bundles everything needed to render one page that
// isn't on the page row itself: the parent site, the i18n config, and
// the seo template strings. RenderPages computes this once per build
// and passes a copy to every renderPageWithContext call so renderPage
// stays cheap and free of context.Context plumbing.
type pageRenderContext struct {
	site     store.Site
	i18n     I18nConfig
	titleTpl string
	descTpl  string
}

// renderPage is the legacy entry point used by the standalone preview
// helpers (RenderSingleBlock / RenderPagePreview). It renders a page
// without site-level context (no meta-template expansion, no hreflang).
// The build pipeline calls renderPageWithContext for the full output.
func renderPage(page store.Page, blocks []store.Block, components map[string]bool) string {
	return renderPageWithContext(page, blocks, components, pageRenderContext{})
}

// renderPageWithContext builds the page Astro source with the full
// site-level rendering context: meta-title/description templates,
// hreflang alternates, canonical override. Empty pageRenderContext
// degrades to the legacy behaviour for the preview helpers.
func renderPageWithContext(page store.Page, blocks []store.Block, components map[string]bool, ctx pageRenderContext) string {
	var b strings.Builder

	// Depth-aware import prefix. Pages live under src/pages/{slug}.astro;
	// nested slugs like /tjanster/avtal land at src/pages/tjanster/avtal.astro
	// which needs ../../components/ to climb back to src/. Without this,
	// every page deeper than one level fails astro build with "could not
	// resolve '../components/foo.astro'".
	prefix := strings.Repeat("../", pageDepth(page.Slug)+1)

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("import Base from '%slayouts/Base.astro';\n", prefix))

	// Collect component imports
	imports := make(map[string]bool)
	for _, bl := range blocks {
		if bl.IsVisible == 0 {
			continue
		}
		compName := extractComponentName(bl)
		if compName != "" && components[compName] {
			imports[compName] = true
		}
	}
	for name := range imports {
		b.WriteString(fmt.Sprintf("import %s from '%scomponents/%s.astro';\n", pascalCase(name), prefix, name))
	}

	b.WriteString("---\n\n")

	// Page wrapper. Title + description run through the operator-set
	// templates from seo.meta_title_template / seo.meta_description_template
	// when those settings are non-empty. Tokens supported:
	// {page_title}, {page_description}, {site_name}, {lang}, {separator}.
	pageTitle := page.MetaTitle
	if pageTitle == "" {
		pageTitle = page.Title
	}
	pageDesc := page.MetaDescription

	vars := MetaTemplateVars{
		PageTitle:       pageTitle,
		PageDescription: pageDesc,
		SiteName:        ctx.site.Name,
		Lang:            ctx.site.Lang,
	}
	title := ExpandMetaTemplate(ctx.titleTpl, pageTitle, vars)
	description := ExpandMetaTemplate(ctx.descTpl, pageDesc, vars)

	b.WriteString(fmt.Sprintf("<Base title=\"%s\"", escapeAttr(title)))
	if description != "" {
		b.WriteString(fmt.Sprintf(" description=\"%s\"", escapeAttr(description)))
	}
	if page.NoIndex == 1 {
		b.WriteString(` robots="noindex, nofollow"`)
	}
	// Per-page suppression of global blocks. Used by landing pages that
	// want to drop the site-wide nav and footer for higher conversion.
	// Layout reads {hideGlobalBlocks} and skips header + footer when true.
	if page.HideGlobalBlocks == 1 {
		b.WriteString(` hideGlobalBlocks={true}`)
	}
	// Hreflang alternates. Multi-language sites get one <link rel="alternate">
	// per locale that has a counterpart page, plus an x-default. Mono-language
	// sites and hreflang_strategy=off drop the prop entirely.
	if alts := ctx.i18n.ComputeAlternates(ctx.site.Domain, page.Slug); len(alts) > 0 {
		b.WriteString(" alternates={")
		b.WriteString(hreflangPropExpression(alts))
		b.WriteString("}")
	}
	b.WriteString(">\n")

	// Render blocks
	for _, bl := range blocks {
		if bl.IsVisible == 0 {
			continue
		}
		b.WriteString(renderBlock(bl, components))
	}

	b.WriteString("</Base>\n")
	return b.String()
}

func renderBlock(bl store.Block, components map[string]bool) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(bl.DataJson), &data); err != nil {
		return fmt.Sprintf("  <!-- block %s: invalid data -->\n", bl.ID)
	}

	var inner string
	// Check if this block uses a component
	compName := extractComponentName(bl)
	switch {
	case compName != "" && components[compName]:
		inner = renderComponentBlock(compName, data)
	case dataHasHTML(data):
		// Check for raw HTML
		inner = fmt.Sprintf("  %s\n", data["html"])
	default:
		// Default: render as a section with data-driven content
		inner = renderDataBlock(bl.BlockType, data)
	}

	// Phase 18.3: optional per-visitor condition. The hydration script
	// loaded by RenderVisitorHydration walks every [data-asp-when] node
	// and removes `hidden` when the embedded DSL matches. Empty / missing
	// condition renders verbatim, no wrapper.
	if cond, ok := data["condition"].(string); ok && strings.TrimSpace(cond) != "" {
		return fmt.Sprintf("  <div data-asp-when=\"%s\" hidden>\n%s  </div>\n", escapeAttr(cond), inner)
	}
	return inner
}

func dataHasHTML(data map[string]any) bool {
	if v, ok := data["html"]; ok {
		if s, ok := v.(string); ok && s != "" {
			return true
		}
	}
	return false
}

func renderComponentBlock(name string, data map[string]any) string {
	pascal := pascalCase(name)
	props, ok := data["props"].(map[string]any)
	if !ok {
		// Treat the entire data as props
		props = data
		delete(props, "component")
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  <%s", pascal))
	for k, v := range props {
		switch val := v.(type) {
		case string:
			b.WriteString(fmt.Sprintf(" %s=\"%s\"", k, escapeAttr(val)))
		case float64:
			b.WriteString(fmt.Sprintf(" %s={%g}", k, val))
		case bool:
			if val {
				b.WriteString(fmt.Sprintf(" %s", k))
			}
		default:
			// Complex values passed as JSON expression
			j, _ := json.Marshal(val)
			b.WriteString(fmt.Sprintf(" %s={%s}", k, string(j)))
		}
	}
	b.WriteString(" />\n")
	return b.String()
}

func renderDataBlock(blockType string, data map[string]any) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  <section class=\"block block--%s\">\n", escapeAttr(blockType)))

	if heading, ok := data["heading"].(string); ok {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if subheading, ok := data["subheading"].(string); ok {
		b.WriteString(fmt.Sprintf("    <p>%s</p>\n", escapeHTML(subheading)))
	}
	if text, ok := data["text"].(string); ok {
		b.WriteString(fmt.Sprintf("    <div>%s</div>\n", escapeHTML(text)))
	}
	if ctaText, ok := data["cta_text"].(string); ok {
		ctaURL, _ := data["cta_url"].(string)
		if ctaURL == "" {
			ctaURL = "#"
		}
		b.WriteString(fmt.Sprintf("    <a href=\"%s\" class=\"btn-primary\">%s</a>\n",
			escapeURL(ctaURL), escapeHTML(ctaText)))
	}

	b.WriteString("  </section>\n")
	return b.String()
}

// escapeHTML escapes text content for safe injection into HTML body.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// escapeURL sanitizes a URL for href/src attribute use. Rejects javascript: and
// data: schemes, then escapes for attribute context.
func escapeURL(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	// Block dangerous schemes; fall back to a safe anchor.
	for _, bad := range []string{"javascript:", "data:", "vbscript:"} {
		if strings.HasPrefix(lower, bad) {
			return "#"
		}
	}
	return escapeAttr(s)
}

func extractComponentName(bl store.Block) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(bl.DataJson), &data); err != nil {
		return ""
	}
	if comp, ok := data["component"].(string); ok {
		return comp
	}
	return ""
}

func slugToFilePath(slug string, wsDir string) string {
	slug = strings.Trim(slug, "/")
	if slug == "" {
		return filepath.Join(wsDir, "src", "pages", "index.astro")
	}
	return filepath.Join(wsDir, "src", "pages", slug+".astro")
}

// pageDepth returns how many subdirectories deep under src/pages the page
// file sits. Used to emit the correct number of ../ in import paths so
// pages at any depth can resolve ../layouts/Base.astro and
// ../components/{name}.astro consistently.
func pageDepth(slug string) int {
	slug = strings.Trim(slug, "/")
	if slug == "" {
		return 0
	}
	return strings.Count(slug, "/")
}

func pascalCase(s string) string {
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func escapeAttr(s string) string {
	// Order matters: & must be first, otherwise we double-escape the entities below.
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
