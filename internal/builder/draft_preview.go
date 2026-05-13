package builder

import (
	"context"
	"fmt"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// RenderPageDraft assembles a complete, browser-renderable HTML document
// for a single page using the current draft DB state. NOT a build:
//   - skips bun + astro entirely (no dist write, no compile step)
//   - reuses renderBlock + extractHTMLFromGlobalBlock + BuildCSS, the same
//     primitives the production builder uses, so visual fidelity is high
//   - emits noindex + same-origin-only headers via the calling handler so
//     the document is safe to load via iframe srcdoc inside the admin app
//
// Output is a single self-contained HTML string. No external requests other
// than the site's CSS classes (already inlined) and uploaded fonts. All
// component blocks (Astro `<PascalName />` syntax) degrade to a notice in
// place of a render attempt. Astro compilation is what resolves them, and
// the preview path stops short of that to keep the round-trip fast.
func RenderPageDraft(ctx context.Context, queries *store.Queries, siteID, pageID string) (string, error) {
	page, err := queries.GetPageByID(ctx, pageID)
	if err != nil {
		return "", fmt.Errorf("get page: %w", err)
	}
	if page.SiteID != siteID {
		return "", fmt.Errorf("page does not belong to site")
	}

	site, err := queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return "", fmt.Errorf("get site: %w", err)
	}

	blocks, err := queries.ListBlocksByPage(ctx, pageID)
	if err != nil {
		return "", fmt.Errorf("list blocks: %w", err)
	}

	componentList, _ := queries.ListComponentsBySite(ctx, siteID)
	componentExts := make(map[string]string, len(componentList))
	for _, c := range componentList {
		componentExts[c.Name] = pickComponentExt(c.Template)
	}

	mediaByID := loadBlockMedia(ctx, queries, blocks)

	var blocksHTML strings.Builder
	for _, bl := range blocks {
		if bl.IsVisible == 0 {
			continue
		}
		raw := renderBlock(bl, componentExts, mediaByID)
		// Component blocks emit Astro `<PascalName ... />` which the browser
		// can't render. Detect and replace with an inline notice so the
		// preview stays useful instead of swallowing the block silently.
		if isComponentBlockOutput(raw, componentExts) {
			blocksHTML.WriteString(componentBlockPlaceholder(bl, componentExts))
			continue
		}
		blocksHTML.WriteString(raw)
	}

	var headerHTML, footerHTML string
	if page.HideGlobalBlocks == 0 {
		globalBlocks, _ := queries.ListActiveGlobalBlocksBySite(ctx, siteID)
		for _, gb := range globalBlocks {
			html := extractHTMLFromGlobalBlock(gb)
			switch gb.Slot {
			case "header":
				headerHTML = html
			case "footer":
				footerHTML = html
			}
		}
	}

	css, err := BuildCSS(ctx, queries, siteID)
	if err != nil {
		return "", fmt.Errorf("build css: %w", err)
	}
	fontFace := SiteFontsCSS(ctx, queries, siteID)

	title := strings.TrimSpace(page.Title)
	if title == "" {
		title = strings.TrimSpace(site.Name)
	}
	if title == "" {
		title = "Preview"
	}
	lang := strings.TrimSpace(site.Lang)
	if lang == "" {
		lang = "en"
	}

	var doc strings.Builder
	doc.WriteString("<!DOCTYPE html>\n")
	doc.WriteString(fmt.Sprintf(`<html lang="%s">`, escapeAttr(lang)))
	doc.WriteString("<head>")
	doc.WriteString(`<meta charset="UTF-8">`)
	doc.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1.0">`)
	doc.WriteString(`<meta name="robots" content="noindex, nofollow">`)
	doc.WriteString(fmt.Sprintf(`<title>%s · Preview</title>`, escapeAttr(title)))
	doc.WriteString(`<style data-atomicsite-preview="reset">html,body{margin:0;padding:0}body{min-height:100vh;background:#fafaf9;color:#18181b;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif}*{box-sizing:border-box}img{max-width:100%;height:auto}.atomicsite-preview-component-stub{margin:1.5rem auto;max-width:60rem;padding:1.25rem 1.5rem;border:1px dashed #d6d3d1;border-radius:0.75rem;background:#fff7ed;color:#9a3412;font-family:ui-monospace,Menlo,Consolas,monospace;font-size:0.8125rem;line-height:1.55}</style>`)
	if fontFace != "" {
		doc.WriteString(`<style data-atomicsite-preview="fonts">`)
		doc.WriteString(fontFace)
		doc.WriteString(`</style>`)
	}
	doc.WriteString(`<style data-atomicsite-preview="site">`)
	doc.WriteString(css)
	doc.WriteString(`</style>`)
	doc.WriteString("</head><body>")
	doc.WriteString(headerHTML)
	doc.WriteString(`<main>`)
	doc.WriteString(blocksHTML.String())
	doc.WriteString(`</main>`)
	doc.WriteString(footerHTML)
	doc.WriteString("</body></html>")

	return doc.String(), nil
}

// isComponentBlockOutput recognises renderBlock output that begins with
// `  <PascalName` for any component the site has registered. Component
// blocks need Astro to compile and aren't browser-renderable as-is.
func isComponentBlockOutput(raw string, componentExts map[string]string) bool {
	if len(componentExts) == 0 {
		return false
	}
	trimmed := strings.TrimLeft(raw, " \t\n")
	if !strings.HasPrefix(trimmed, "<") {
		return false
	}
	rest := trimmed[1:]
	end := strings.IndexAny(rest, " />\n")
	if end <= 0 {
		return false
	}
	tag := rest[:end]
	if tag == "" || !isUpperASCII(tag[0]) {
		return false
	}
	for name := range componentExts {
		if pascalCase(name) == tag {
			return true
		}
	}
	return false
}

func componentBlockPlaceholder(bl store.Block, componentExts map[string]string) string {
	compName := extractComponentName(bl)
	if compName == "" {
		compName = bl.BlockType
	}
	return fmt.Sprintf(
		`<aside class="atomicsite-preview-component-stub" data-block-id="%s" data-block-type="%s"><strong>Component block:</strong> &lt;%s /&gt;. Preview shows the rendered output after the next build.</aside>`,
		escapeAttr(bl.ID),
		escapeAttr(bl.BlockType),
		escapeAttr(pascalCase(compName)),
	)
}

func isUpperASCII(b byte) bool {
	return b >= 'A' && b <= 'Z'
}
