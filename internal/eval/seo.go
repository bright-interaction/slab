package eval

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// RunSEOChecks evaluates on-page + technical SEO across all pages, reporting
// per-page failures so the agent can see exactly which pages need fixes.
func RunSEOChecks(site *SiteContext) []CheckResult {
	var checks []CheckResult

	// Site-wide once-only checks
	checks = append(checks, checkRobotsTxt(site))
	checks = append(checks, checkSitemap(site))
	checks = append(checks, checkLLMsTxt(site))

	// Per-page checks aggregated into single results (reports first failing page)
	checks = append(checks, perPageCheck("Has Title", "on-page", 2, site, func(p PageContext) (bool, string) {
		t := titleOf(p.Doc)
		if t == "" {
			return false, "missing <title>"
		}
		return true, ""
	}, "Add a <title> tag via meta_title in the page settings."))

	checks = append(checks, perPageCheck("Title Length 30-60", "on-page", 1, site, func(p PageContext) (bool, string) {
		t := titleOf(p.Doc)
		n := utf8.RuneCountInString(t)
		if n < 30 || n > 60 {
			return false, fmt.Sprintf("title is %d chars (want 30-60)", n)
		}
		return true, ""
	}, "Aim for 30-60 character titles for optimal SERP display."))

	checks = append(checks, perPageCheck("Has Meta Description", "on-page", 2, site, func(p PageContext) (bool, string) {
		d := findMetaContent(p.Doc, "description")
		if d == "" {
			return false, "missing meta description"
		}
		return true, ""
	}, "Set meta_description in page settings."))

	checks = append(checks, perPageCheck("Meta Description 120-160", "on-page", 1, site, func(p PageContext) (bool, string) {
		d := findMetaContent(p.Doc, "description")
		n := utf8.RuneCountInString(d)
		if d == "" || n < 120 || n > 160 {
			return false, fmt.Sprintf("description is %d chars (want 120-160)", n)
		}
		return true, ""
	}, "Aim for 120-160 character meta descriptions."))

	checks = append(checks, perPageCheck("Has H1", "on-page", 2, site, func(p PageContext) (bool, string) {
		h1s := elementsByTag(p.Doc, "h1")
		if len(h1s) == 0 {
			return false, "no <h1> on page"
		}
		return true, ""
	}, "Every page must have exactly one <h1>."))

	checks = append(checks, perPageCheck("Single H1", "on-page", 1, site, func(p PageContext) (bool, string) {
		h1s := elementsByTag(p.Doc, "h1")
		if len(h1s) != 1 {
			return false, fmt.Sprintf("%d <h1> elements (want exactly 1)", len(h1s))
		}
		return true, ""
	}, "Use only one <h1> per page; subsequent headings should be <h2>+."))

	checks = append(checks, perPageCheck("Heading Hierarchy", "on-page", 2, site, func(p PageContext) (bool, string) {
		if ok, msg := validateHeadingOrder(p.Doc); !ok {
			return false, msg
		}
		return true, ""
	}, "Don't skip heading levels (e.g., H1 -> H3 without an H2 between)."))

	checks = append(checks, perPageCheck("Images Have Alt Text", "on-page", 2, site, func(p PageContext) (bool, string) {
		missing := 0
		for _, img := range elementsByTag(p.Doc, "img") {
			if !hasAttr(img, "alt") {
				missing++
			}
		}
		if missing > 0 {
			return false, fmt.Sprintf("%d image(s) missing alt attribute", missing)
		}
		return true, ""
	}, "Add alt text to every <img>. Use alt=\"\" for decorative images."))

	checks = append(checks, perPageCheck("Canonical URL", "technical", 2, site, func(p PageContext) (bool, string) {
		if findLinkHref(p.Doc, "canonical") == "" {
			return false, "no <link rel=canonical>"
		}
		return true, ""
	}, "Layouts.go emits canonical from the site domain."))

	checks = append(checks, perPageCheck("Language Declared", "technical", 1, site, func(p PageContext) (bool, string) {
		htmlNode := firstElementByTag(p.Doc, "html")
		if htmlNode == nil || attr(htmlNode, "lang") == "" {
			return false, "no lang attribute on <html>"
		}
		return true, ""
	}, "Set the lang= attribute on <html> from site settings."))

	checks = append(checks, perPageCheck("Viewport Meta", "technical", 2, site, func(p PageContext) (bool, string) {
		v := findMetaContent(p.Doc, "viewport")
		if v == "" {
			return false, "no viewport meta tag"
		}
		return true, ""
	}, "Add <meta name=viewport content=\"width=device-width, initial-scale=1\">."))

	checks = append(checks, perPageCheck("Charset Declared", "technical", 1, site, func(p PageContext) (bool, string) {
		for _, m := range elementsByTag(p.Doc, "meta") {
			if attr(m, "charset") != "" {
				return true, ""
			}
		}
		return false, "no <meta charset>"
	}, "Add <meta charset=\"UTF-8\"> as the first child of <head>."))

	checks = append(checks, perPageCheck("HTML5 Doctype", "technical", 1, site, func(p PageContext) (bool, string) {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(p.HTML)), "<!doctype html>") {
			return false, "missing <!DOCTYPE html>"
		}
		return true, ""
	}, "Ensure pages start with <!DOCTYPE html>."))

	checks = append(checks, perPageCheck("Open Graph Tags", "social", 1, site, func(p PageContext) (bool, string) {
		missing := []string{}
		for _, prop := range []string{"og:title", "og:description", "og:image", "og:type", "og:url"} {
			if findMetaContent(p.Doc, prop) == "" {
				missing = append(missing, prop)
			}
		}
		if len(missing) > 0 {
			return false, "missing OG tags: " + strings.Join(missing, ", ")
		}
		return true, ""
	}, "Add Open Graph tags via the layout (og:title, og:description, og:image, og:type, og:url)."))

	checks = append(checks, perPageCheck("Not Noindexed", "technical", 3, site, func(p PageContext) (bool, string) {
		// Multiple sources: <meta name=robots>, the Base layout's robots prop
		robots := strings.ToLower(findMetaContent(p.Doc, "robots"))
		if strings.Contains(robots, "noindex") {
			return false, "page has noindex meta tag"
		}
		return true, ""
	}, "If you want this page indexed, set no_index=false on the page."))

	checks = append(checks, perPageCheck("URL Length < 75", "url", 1, site, func(p PageContext) (bool, string) {
		if len(p.Slug) >= 75 {
			return false, fmt.Sprintf("URL is %d chars (want < 75)", len(p.Slug))
		}
		return true, ""
	}, "Shorter URLs improve sharing and SERP display."))

	checks = append(checks, perPageCheck("URL Lowercase", "url", 1, site, func(p PageContext) (bool, string) {
		if p.Slug != strings.ToLower(p.Slug) {
			return false, "URL contains uppercase letters"
		}
		return true, ""
	}, "Use lowercase URLs only."))

	checks = append(checks, perPageCheck("URL No Underscores", "url", 1, site, func(p PageContext) (bool, string) {
		if strings.Contains(p.Slug, "_") {
			return false, "URL contains underscores"
		}
		return true, ""
	}, "Use hyphens (-), not underscores (_), in URLs."))

	checks = append(checks, perPageCheck("Has Internal Links", "on-page", 1, site, func(p PageContext) (bool, string) {
		count := 0
		for _, a := range elementsByTag(p.Doc, "a") {
			href := attr(a, "href")
			if href == "" || strings.HasPrefix(href, "#") {
				continue
			}
			if strings.HasPrefix(href, "/") || !strings.Contains(href, "://") {
				count++
			}
		}
		if count == 0 {
			return false, "no internal links on page"
		}
		return true, ""
	}, "Add internal links between pages to support crawling and topic clustering."))

	return checks
}

// perPageCheck runs predicate on every page, fails on first page that fails.
// Detail reports which page failed and why; recommendation is shared.
func perPageCheck(name, section string, weight int, site *SiteContext,
	predicate func(PageContext) (bool, string), recommendation string) CheckResult {
	for _, p := range site.Pages {
		ok, detail := predicate(p)
		if !ok {
			r := Fail(name, section, weight, SeverityError, detail, recommendation)
			r.Page = p.Slug
			return r
		}
	}
	if len(site.Pages) == 0 {
		return Info(name, section, "no pages to evaluate")
	}
	return Pass(name, section, weight, fmt.Sprintf("All %d pages pass", len(site.Pages)))
}

// titleOf reads the <title> text content.
func titleOf(doc *html.Node) string {
	t := firstElementByTag(doc, "title")
	if t == nil {
		return ""
	}
	return textContent(t)
}

// validateHeadingOrder checks that h1-h6 levels never skip downward
// (e.g. h1 -> h3 without an h2 between). Returns (ok, errorMessage).
func validateHeadingOrder(doc *html.Node) (bool, string) {
	var levels []int
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil {
			return
		}
		if n.Type == html.ElementNode && len(n.Data) == 2 && n.Data[0] == 'h' &&
			n.Data[1] >= '1' && n.Data[1] <= '6' {
			levels = append(levels, int(n.Data[1]-'0'))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	for i := 1; i < len(levels); i++ {
		if levels[i] > levels[i-1]+1 {
			return false, fmt.Sprintf("h%d follows h%d (skipped a level)", levels[i], levels[i-1])
		}
	}
	return true, ""
}

// checkRobotsTxt validates robots.txt presence + minimal content.
func checkRobotsTxt(site *SiteContext) CheckResult {
	if site.RobotsTxt == "" {
		return Fail("robots.txt", "files", 2, SeverityError,
			"no robots.txt in dist/", "Atomicsite generates this -- check build pipeline.")
	}
	if !strings.Contains(strings.ToLower(site.RobotsTxt), "user-agent") {
		return Fail("robots.txt", "files", 2, SeverityWarning,
			"robots.txt has no User-agent directive", "Ensure at least one User-agent rule.")
	}
	return Pass("robots.txt", "files", 2, "robots.txt present and valid")
}

// checkSitemap validates sitemap.xml or sitemap-index.xml.
func checkSitemap(site *SiteContext) CheckResult {
	if site.SitemapXML == "" {
		return Fail("XML Sitemap", "files", 2, SeverityWarning,
			"no sitemap-index.xml or sitemap.xml found", "Astro's @astrojs/sitemap should emit one. Verify build succeeded.")
	}
	return Pass("XML Sitemap", "files", 2, "Sitemap present")
}

// checkLLMsTxt checks for AI-search-ready llms.txt.
func checkLLMsTxt(site *SiteContext) CheckResult {
	if site.LLMsTxt == "" {
		return Fail("llms.txt", "files", 1, SeverityInfo,
			"no /llms.txt", "Atomicsite generates this from site profile + pages.")
	}
	return Pass("llms.txt", "files", 1, "llms.txt present (AI search readiness)")
}
