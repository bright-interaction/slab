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

	// Security headers as meta
	if v := settingsMap["security.referrer_policy"]; v != "" {
		b.WriteString(fmt.Sprintf("  <meta name=\"referrer\" content=\"%s\" />\n", v))
	}

	// Analytics
	if site.UmamiID != "" && site.UmamiUrl != "" {
		b.WriteString(fmt.Sprintf("  <script defer src=\"%s/script.js\" data-website-id=\"%s\"></script>\n", site.UmamiUrl, site.UmamiID))
	}
	if site.CookieproofDomain != "" {
		b.WriteString(fmt.Sprintf("  <script defer src=\"https://%s/cookieproof.js\"></script>\n", site.CookieproofDomain))
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
	if html, ok := data["html"].(string); ok {
		return html
	}
	// If no raw HTML, try to serialize the data as a simple representation
	return ""
}

func escapeAstroString(s string) string {
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
