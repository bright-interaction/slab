package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

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

	for _, page := range pages {
		blocks, err := queries.ListBlocksByPage(ctx, page.ID)
		if err != nil {
			return 0, fmt.Errorf("list blocks for page %s: %w", page.Slug, err)
		}
		slog.Info("build: rendering page", "slug", page.Slug, "blocks", len(blocks))

		content := renderPage(page, blocks, componentNames)
		pagePath := slugToFilePath(page.Slug, wsDir)

		if err := WriteFile(pagePath, content); err != nil {
			return 0, fmt.Errorf("write page %s: %w", page.Slug, err)
		}
	}

	return len(pages), nil
}

func renderPage(page store.Page, blocks []store.Block, components map[string]bool) string {
	var b strings.Builder

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString("import Base from '../layouts/Base.astro';\n")

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
		b.WriteString(fmt.Sprintf("import %s from '../components/%s.astro';\n", pascalCase(name), name))
	}

	b.WriteString("---\n\n")

	// Page wrapper
	title := page.MetaTitle
	if title == "" {
		title = page.Title
	}
	b.WriteString(fmt.Sprintf("<Base title=\"%s\"", escapeAttr(title)))
	if page.MetaDescription != "" {
		b.WriteString(fmt.Sprintf(" description=\"%s\"", escapeAttr(page.MetaDescription)))
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

	// Check if this block uses a component
	compName := extractComponentName(bl)
	if compName != "" && components[compName] {
		return renderComponentBlock(compName, data)
	}

	// Check for raw HTML
	if html, ok := data["html"].(string); ok {
		return fmt.Sprintf("  %s\n", html)
	}

	// Default: render as a section with data-driven content
	return renderDataBlock(bl.BlockType, data)
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
