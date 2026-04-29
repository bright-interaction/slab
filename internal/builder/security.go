package builder

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// SecurityHeaders holds the canonical header set for a built site.
// Generated from site_settings + allowed_scripts, written as both a _headers
// file (Caddy/Netlify/Cloudflare format) and nginx.conf snippet, and also
// emitted as <meta http-equiv> tags in the layout for belt-and-suspenders.
type SecurityHeaders struct {
	CSP                  string
	HSTS                 string
	XFrameOptions        string
	XContentTypeOptions  string
	ReferrerPolicy       string
	PermissionsPolicy    string
	COOP                 string
	CORP                 string
	COEP                 string
}

// BuildSecurityHeaders assembles all security headers from settings + allowed scripts.
func BuildSecurityHeaders(ctx context.Context, queries *store.Queries, siteID string) (SecurityHeaders, error) {
	settings, _ := queries.ListSettingsBySite(ctx, siteID)
	sm := make(map[string]string)
	for _, s := range settings {
		sm[s.Category+"."+s.Key] = s.Value
	}

	// Group active trusted domains by kind so each kind feeds the right CSP
	// directive. 'all' is fanned out into every list.
	rows, _ := queries.ListAllowedScriptsBySite(ctx, siteID)
	allow := AllowedDomains{}
	for _, s := range rows {
		if s.IsActive != 1 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(s.Kind)) {
		case "frame":
			allow.Frame = append(allow.Frame, s.Domain)
		case "image":
			allow.Image = append(allow.Image, s.Domain)
		case "media":
			allow.Media = append(allow.Media, s.Domain)
		case "connect":
			allow.Connect = append(allow.Connect, s.Domain)
		case "all":
			allow.Script = append(allow.Script, s.Domain)
			allow.Connect = append(allow.Connect, s.Domain)
			allow.Frame = append(allow.Frame, s.Domain)
			allow.Image = append(allow.Image, s.Domain)
			allow.Media = append(allow.Media, s.Domain)
		default: // "" or "script"
			allow.Script = append(allow.Script, s.Domain)
			allow.Connect = append(allow.Connect, s.Domain)
		}
	}

	h := SecurityHeaders{}

	// CSP: if fully overridden in settings, use that; otherwise auto-build
	// from the structured allowlist + per-site CSP settings.
	if override := sm["security.csp_policy"]; override != "" && override != "auto" {
		h.CSP = override
	} else if boolSetting(sm["security.csp_enabled"], true) {
		opts := CSPOptions{
			Allow:           allow,
			FrameAncestors:  orDefault(sm["security.frame_ancestors"], "'none'"),
			ExtraDirectives: sm["security.csp_extra_directives"],
		}
		h.CSP = buildCSP(opts)
	}

	if boolSetting(sm["security.hsts_enabled"], true) {
		maxAge := sm["security.hsts_max_age"]
		if maxAge == "" {
			maxAge = "31536000"
		}
		h.HSTS = fmt.Sprintf("max-age=%s; includeSubDomains", maxAge)
		if boolSetting(sm["security.hsts_preload"], false) {
			h.HSTS += "; preload"
		}
	}

	h.XFrameOptions = orDefault(sm["security.x_frame_options"], "DENY")
	h.XContentTypeOptions = orDefault(sm["security.x_content_type"], "nosniff")
	h.ReferrerPolicy = orDefault(sm["security.referrer_policy"], "strict-origin-when-cross-origin")
	h.PermissionsPolicy = orDefault(sm["security.permissions_policy"], "camera=(), microphone=(), geolocation=(), interest-cohort=()")
	h.COOP = orDefault(sm["security.coop"], "same-origin")
	h.CORP = orDefault(sm["security.corp"], "same-origin")
	h.COEP = orDefault(sm["security.coep"], "") // Off by default: breaks third-party embeds

	return h, nil
}

// CSPForMeta returns the CSP with directives stripped that browsers ignore in
// <meta http-equiv> form (frame-ancestors, sandbox, report-uri, report-to,
// upgrade-insecure-requests only works inline). Use this for layout meta tags;
// use the full CSP for HTTP headers.
func CSPForMeta(full string) string {
	if full == "" {
		return ""
	}
	drop := map[string]bool{
		"frame-ancestors":            true,
		"sandbox":                    true,
		"report-uri":                 true,
		"report-to":                  true,
		"upgrade-insecure-requests":  true,
	}
	parts := strings.Split(full, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		first := strings.SplitN(p, " ", 2)[0]
		if drop[strings.ToLower(first)] {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, "; ")
}

// AllowedDomains groups trusted external domains by which CSP directive
// they should land in. Built by BuildSecurityHeaders from
// allowed_scripts rows partitioned by their kind column.
type AllowedDomains struct {
	Script  []string // -> script-src
	Connect []string // -> connect-src
	Frame   []string // -> frame-src (iframes: cal.com / YouTube / Stripe Checkout)
	Image   []string // -> img-src (extra hosts beyond the default https: wildcard)
	Media   []string // -> media-src (audio/video sources)
}

// CSPOptions is the input to buildCSP. Allow contributes directive-specific
// hostnames; FrameAncestors controls who can embed THIS site in an iframe
// (default 'none' is anti-clickjacking; operators can set 'self' or a host
// list when they need to allow embedding); ExtraDirectives is an opaque
// suffix the operator can paste in to escape any limitation here, joined
// with semicolons.
type CSPOptions struct {
	Allow           AllowedDomains
	FrameAncestors  string
	ExtraDirectives string
}

// buildCSP constructs a Content-Security-Policy from the structured
// allowlist + per-site CSP settings. Each domain lands in exactly the
// directive it was registered for so an iframe domain doesn't accidentally
// gain script-execution privileges.
func buildCSP(opts CSPOptions) string {
	join := func(base string, extra []string) string {
		if len(extra) == 0 {
			return base
		}
		return base + " " + strings.Join(extra, " ")
	}

	scriptSrc := join("'self'", opts.Allow.Script)
	connectSrc := join("'self'", opts.Allow.Connect)
	imgSrc := join("'self' data: https:", opts.Allow.Image)
	fontSrc := "'self' https: data:"
	styleSrc := "'self' 'unsafe-inline'"

	frameAncestors := strings.TrimSpace(opts.FrameAncestors)
	if frameAncestors == "" {
		frameAncestors = "'none'"
	}

	directives := []string{
		"default-src 'self'",
		"script-src " + scriptSrc,
		"style-src " + styleSrc,
		"img-src " + imgSrc,
		"connect-src " + connectSrc,
		"font-src " + fontSrc,
		"frame-ancestors " + frameAncestors,
		"base-uri 'self'",
		"form-action 'self'",
		"object-src 'none'",
		"upgrade-insecure-requests",
	}

	// frame-src and media-src are opt-in. Without explicit hosts the
	// browser falls back to default-src ('self'), which blocks every
	// third-party iframe. Adding these only when there's a host list keeps
	// the directive count tight on simple sites.
	if len(opts.Allow.Frame) > 0 {
		directives = append(directives, "frame-src 'self' "+strings.Join(opts.Allow.Frame, " "))
	}
	if len(opts.Allow.Media) > 0 {
		directives = append(directives, "media-src 'self' "+strings.Join(opts.Allow.Media, " "))
	}

	out := strings.Join(directives, "; ")
	if extra := strings.TrimSpace(opts.ExtraDirectives); extra != "" {
		// Strip a trailing semicolon from the operator paste so we don't
		// emit "...; ; " and confuse browsers.
		extra = strings.TrimRight(extra, "; ")
		out += "; " + extra
	}
	return out
}

// RenderSecurityHeaders writes the _headers file (Netlify/Cloudflare Pages/Caddy)
// to the workspace. This makes the built site carry security headers whenever
// deployed to one of those platforms. For nginx-based deployments, see RenderNginxConfig.
func RenderSecurityHeaders(ctx context.Context, queries *store.Queries, siteID string, wsDir string) error {
	h, err := BuildSecurityHeaders(ctx, queries, siteID)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("# Auto-generated by Atomicsite\n")
	b.WriteString("/*\n")

	writeHeader := func(name, value string) {
		if value != "" {
			b.WriteString(fmt.Sprintf("  %s: %s\n", name, value))
		}
	}

	writeHeader("Content-Security-Policy", h.CSP)
	writeHeader("Strict-Transport-Security", h.HSTS)
	writeHeader("X-Frame-Options", h.XFrameOptions)
	writeHeader("X-Content-Type-Options", h.XContentTypeOptions)
	writeHeader("Referrer-Policy", h.ReferrerPolicy)
	writeHeader("Permissions-Policy", h.PermissionsPolicy)
	writeHeader("Cross-Origin-Opener-Policy", h.COOP)
	writeHeader("Cross-Origin-Resource-Policy", h.CORP)
	writeHeader("Cross-Origin-Embedder-Policy", h.COEP)

	// Long-cache for hashed assets (Astro emits them under /_assets/)
	b.WriteString("\n/_assets/*\n")
	b.WriteString("  Cache-Control: public, max-age=31536000, immutable\n")

	// Long-cache for media (content-addressed)
	b.WriteString("\n/media/*\n")
	b.WriteString("  Cache-Control: public, max-age=31536000, immutable\n")

	return WriteFile(filepath.Join(wsDir, "public", "_headers"), b.String())
}

// RenderHumansTxt writes a humans.txt file with site profile info.
// All user fields are scrubbed of CR/LF/tab to prevent injection.
func RenderHumansTxt(ctx context.Context, queries *store.Queries, siteID string, wsDir string) error {
	profile, _ := queries.GetSiteProfile(ctx, siteID)
	site, err := queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("/* TEAM */\n")
	if v := stripCtl(profile.BusinessName); v != "" {
		b.WriteString("  Organization: " + v + "\n")
	}
	if v := stripCtl(profile.ContactEmail); v != "" {
		b.WriteString("  Contact: " + v + "\n")
	}
	city := stripCtl(profile.City)
	country := stripCtl(profile.Country)
	if city != "" || country != "" {
		loc := strings.TrimSpace(strings.Trim(city+", "+country, ", "))
		b.WriteString("  Location: " + loc + "\n")
	}

	b.WriteString("\n/* SITE */\n")
	b.WriteString("  Name: " + stripCtl(site.Name) + "\n")
	b.WriteString("  Language: " + stripCtl(site.Lang) + "\n")
	b.WriteString("  Built with: Atomicsite (https://brightinteraction.com/atomicsite)\n")

	return WriteFile(filepath.Join(wsDir, "public", "humans.txt"), b.String())
}

// RenderLLMsTxt writes an llms.txt file per the https://llmstxt.org/ spec for
// AI search engines (Perplexity, ChatGPT web search, Claude, Gemini).
func RenderLLMsTxt(ctx context.Context, queries *store.Queries, siteID string, wsDir string) error {
	site, err := queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return err
	}
	profile, _ := queries.GetSiteProfile(ctx, siteID)
	settings, _ := queries.ListSettingsBySite(ctx, siteID)
	sm := make(map[string]string)
	for _, s := range settings {
		sm[s.Category+"."+s.Key] = s.Value
	}

	// Check for a user-provided override
	if override := sm["seo.llms_txt"]; override != "" {
		return WriteFile(filepath.Join(wsDir, "public", "llms.txt"), override)
	}

	// Auto-generate from site data
	domain := site.Domain
	if domain == "" {
		domain = "localhost"
	}
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}

	var b strings.Builder
	b.WriteString("# " + stripCtl(site.Name) + "\n\n")

	if v := stripCtl(site.MetaDescription); v != "" {
		b.WriteString("> " + v + "\n\n")
	} else if v := stripCtl(profile.BusinessName); v != "" {
		b.WriteString("> Official site of " + v + ".\n\n")
	}

	// List published pages as reference docs, excluding noindex pages.
	pages, _ := queries.ListPublishedPagesBySite(ctx, siteID)
	var indexable []store.Page
	for _, p := range pages {
		if p.NoIndex == 1 {
			continue
		}
		indexable = append(indexable, p)
	}
	if len(indexable) > 0 {
		b.WriteString("## Pages\n\n")
		for _, p := range indexable {
			title := stripCtl(p.Title)
			if v := stripCtl(p.MetaTitle); v != "" {
				title = v
			}
			desc := stripCtl(p.MetaDescription)
			if desc == "" {
				desc = title
			}
			b.WriteString(fmt.Sprintf("- [%s](%s%s): %s\n", title, domain, p.Slug, desc))
		}
		b.WriteString("\n")
	}

	if v := stripCtl(profile.ContactEmail); v != "" {
		b.WriteString("## Contact\n\n")
		b.WriteString(v + "\n")
	}

	return WriteFile(filepath.Join(wsDir, "public", "llms.txt"), b.String())
}

// --- helpers ---

// stripCtl removes CR, LF, tab, and unicode line/paragraph separators to
// prevent injection into single-line text files.
func stripCtl(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\u2028", " ") // LINE SEPARATOR
	s = strings.ReplaceAll(s, "\u2029", " ") // PARAGRAPH SEPARATOR
	s = strings.ReplaceAll(s, "\u0085", " ") // NEL (Next Line)
	return strings.TrimSpace(s)
}

func boolSetting(v string, def bool) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return def
	}
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

func orDefault(v, def string) string {
	if v == "" || v == "auto" {
		return def
	}
	return v
}
