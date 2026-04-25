package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// RenderLayouts generates Astro layout files from site settings and global blocks.
func RenderLayouts(ctx context.Context, queries *store.Queries, siteID string, wsDir string) error {
	site, err := queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return fmt.Errorf("get site: %w", err)
	}

	globalBlocks, _ := queries.ListActiveGlobalBlocksBySite(ctx, siteID)
	settings, _ := queries.ListSettingsBySite(ctx, siteID)

	// Build settings map for quick lookup
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Category+"."+s.Key] = s.Value
	}

	// Find header and footer global blocks
	var headerHTML, footerHTML string
	for _, gb := range globalBlocks {
		html := extractHTMLFromGlobalBlock(gb)
		switch gb.Slot {
		case "header":
			headerHTML = html
		case "footer":
			footerHTML = html
		}
	}

	// Build the layout
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("import '../styles/global.css';\n\n")
	b.WriteString("interface Props {\n")
	b.WriteString("  title: string;\n")
	b.WriteString("  description?: string;\n")
	b.WriteString("  ogImage?: string;\n")
	b.WriteString("  robots?: string;\n")
	b.WriteString("}\n\n")
	b.WriteString("const {\n")
	b.WriteString(fmt.Sprintf("  title = '%s',\n", escapeAstroString(site.MetaTitle)))
	b.WriteString(fmt.Sprintf("  description = '%s',\n", escapeAstroString(site.MetaDescription)))
	b.WriteString(fmt.Sprintf("  ogImage = '%s',\n", site.OgImageID))
	b.WriteString("  robots = 'index, follow',\n")
	b.WriteString("} = Astro.props;\n")

	// Compute canonical
	domain := site.Domain
	if domain == "" {
		domain = "localhost"
	}
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}
	b.WriteString(fmt.Sprintf("\nconst canonicalURL = new URL(Astro.url.pathname, '%s');\n", domain))
	b.WriteString("---\n\n")

	// HTML
	b.WriteString(fmt.Sprintf("<!DOCTYPE html>\n<html lang=\"%s\">\n<head>\n", site.Lang))
	b.WriteString("  <meta charset=\"UTF-8\" />\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" />\n")
	b.WriteString("  <title>{title}</title>\n")
	b.WriteString("  {description && <meta name=\"description\" content={description} />}\n")
	b.WriteString("  <meta name=\"robots\" content={robots} />\n")
	b.WriteString("  <link rel=\"canonical\" href={canonicalURL.href} />\n\n")

	// Open Graph
	b.WriteString("  <meta property=\"og:title\" content={title} />\n")
	b.WriteString("  {description && <meta property=\"og:description\" content={description} />}\n")
	b.WriteString("  {ogImage && <meta property=\"og:image\" content={ogImage} />}\n")
	b.WriteString("  <meta property=\"og:type\" content=\"website\" />\n")
	b.WriteString("  <meta property=\"og:url\" content={canonicalURL.href} />\n\n")

	// Security headers as meta (belt-and-suspenders: headers also sent via
	// _headers file and nginx.conf for directives browsers honor only in
	// headers, e.g. frame-ancestors, HSTS, X-Frame-Options).
	if headers, err := BuildSecurityHeaders(ctx, queries, siteID); err == nil {
		if headers.ReferrerPolicy != "" {
			b.WriteString(fmt.Sprintf("  <meta name=\"referrer\" content=\"%s\" />\n", escapeAttr(headers.ReferrerPolicy)))
		}
		// Use meta-safe CSP subset -- frame-ancestors etc. only work in headers.
		if metaCSP := CSPForMeta(headers.CSP); metaCSP != "" {
			b.WriteString(fmt.Sprintf("  <meta http-equiv=\"Content-Security-Policy\" content=\"%s\" />\n", escapeAttr(metaCSP)))
		}
	}

	// Analytics
	if site.UmamiID != "" && site.UmamiUrl != "" {
		b.WriteString(fmt.Sprintf("  <script defer src=\"%s/script.js\" data-website-id=\"%s\"></script>\n", site.UmamiUrl, site.UmamiID))
	}

	// CookieProof + Atomicsite consent relay. Gated on
	// analytics.cookieproof_enabled. siteDomain comes from the site row
	// (CookieProof keys multi-tenant configs by domain), site_id is baked
	// into the relay so /t/consent knows which site posted.
	cookieProofEnabled := boolSetting(settingsMap["analytics.cookieproof_enabled"], false)
	if cookieProofEnabled {
		cpDomain := site.CookieproofDomain
		if cpDomain == "" {
			cpDomain = site.Domain
		}
		trackPath := orDefault(settingsMap["analytics.track_path"], "/t")
		snippet := RenderCookieProofSnippet(site.ID, cpDomain, trackPath)
		// Indent two spaces so the snippet sits flush with surrounding <head> children.
		for _, line := range strings.Split(strings.TrimRight(snippet, "\n"), "\n") {
			if line == "" {
				b.WriteString("\n")
				continue
			}
			b.WriteString("  " + line + "\n")
		}
	}

	b.WriteString("</head>\n<body>\n")

	// Header
	if headerHTML != "" {
		b.WriteString("  <header>\n")
		b.WriteString("    " + headerHTML + "\n")
		b.WriteString("  </header>\n")
	}

	b.WriteString("  <main>\n    <slot />\n  </main>\n")

	// Footer
	if footerHTML != "" {
		b.WriteString("  <footer>\n")
		b.WriteString("    " + footerHTML + "\n")
		b.WriteString("  </footer>\n")
	}

	b.WriteString("</body>\n</html>\n")

	return WriteFile(filepath.Join(wsDir, "src", "layouts", "Base.astro"), b.String())
}

func extractHTMLFromGlobalBlock(gb store.GlobalBlock) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(gb.DataJson), &data); err != nil {
		return ""
	}
	// Raw HTML override: if the block stores `data.html`, use that verbatim.
	if html, ok := data["html"].(string); ok && html != "" {
		return html
	}

	// Structured data path. Render the canonical header/footer shapes so the
	// deployed site has a real <header><nav> and <footer> instead of empty
	// strings.
	switch gb.Slot {
	case "header":
		return renderHeaderHTML(data)
	case "footer":
		return renderFooterHTML(data)
	}
	return ""
}

func renderHeaderHTML(data map[string]any) string {
	links := extractNavLinks(data["links"])
	if len(links) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<header class="site-header"><div class="container"><nav class="site-nav" aria-label="Primary"><ul>`)
	for _, l := range links {
		b.WriteString(`<li><a href="`)
		b.WriteString(escapeAttr(l.href))
		b.WriteString(`">`)
		b.WriteString(escapeText(l.label))
		b.WriteString(`</a></li>`)
	}
	b.WriteString(`</ul></nav></div></header>`)
	return b.String()
}

func renderFooterHTML(data map[string]any) string {
	links := extractNavLinks(data["links"])
	copyright, _ := data["copyright"].(string)
	if len(links) == 0 && copyright == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<footer class="site-footer"><div class="container">`)
	if len(links) > 0 {
		b.WriteString(`<nav aria-label="Footer"><ul>`)
		for _, l := range links {
			b.WriteString(`<li><a href="`)
			b.WriteString(escapeAttr(l.href))
			b.WriteString(`">`)
			b.WriteString(escapeText(l.label))
			b.WriteString(`</a></li>`)
		}
		b.WriteString(`</ul></nav>`)
	}
	if copyright != "" {
		b.WriteString(`<p class="site-footer-copy">&copy; `)
		b.WriteString(escapeText(copyright))
		b.WriteString(`</p>`)
	}
	b.WriteString(`</div></footer>`)
	return b.String()
}

type navLink struct {
	label string
	href  string
}

func extractNavLinks(raw any) []navLink {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]navLink, 0, len(arr))
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		label, _ := obj["label"].(string)
		href, _ := obj["href"].(string)
		label = strings.TrimSpace(label)
		href = strings.TrimSpace(href)
		if label == "" || href == "" {
			continue
		}
		out = append(out, navLink{label: label, href: href})
	}
	return out
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func escapeAstroString(s string) string {
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
