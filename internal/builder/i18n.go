package builder

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// HreflangAlternate is one entry in the alternates list emitted as
// <link rel="alternate" hreflang="X" href="Y"> in the page head.
type HreflangAlternate struct {
	Lang string
	URL  string
}

// I18nConfig groups every settings + state input that drives multi-language
// rendering. Built once per site by LoadI18nConfig and consumed by the
// per-page hreflang + canonical computation.
type I18nConfig struct {
	// DefaultLang is the site's primary language code (sites.lang). Pages
	// without a language prefix in their slug belong to this locale.
	DefaultLang string
	// AdditionalLangs is the operator-declared list of other languages
	// the site publishes (CSV in general.additional_langs). Each code
	// becomes a hreflang candidate; whether a candidate actually emits
	// for a given page depends on Strategy + page coverage.
	AdditionalLangs []string
	// Strategy is how alternate URLs are built: "path" (default,
	// /<lang>/<slug>), "subdomain" (<lang>.domain.tld/<slug>), or "off"
	// (no hreflang at all). Path is correct for ~95% of sites; subdomain
	// is opt-in for operators who already run sv.example.com etc.
	Strategy string
	// CanonicalBase overrides the default https://{domain} when the
	// operator hosts the site at a non-trivial root (e.g. behind a CDN
	// path prefix). Read from seo.canonical_base.
	CanonicalBase string
	// PageSlugs is the set of every published page slug, used to gate
	// hreflang emission so we never link to a locale page that doesn't
	// exist.
	PageSlugs map[string]bool
}

// LoadI18nConfig reads sites + settings + published pages and assembles
// everything ComputeAlternates needs. Cheap; called once per build.
func LoadI18nConfig(ctx context.Context, queries *store.Queries, siteID string) (I18nConfig, error) {
	site, err := queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return I18nConfig{}, err
	}
	settings, _ := queries.ListSettingsBySite(ctx, siteID)
	sm := make(map[string]string, len(settings))
	for _, s := range settings {
		sm[s.Category+"."+s.Key] = s.Value
	}

	pages, _ := queries.ListPublishedPagesBySite(ctx, siteID)
	slugs := make(map[string]bool, len(pages))
	for _, p := range pages {
		slugs[normalizeSlug(p.Slug)] = true
	}
	// Sprint 3 (multilingual v1, 2026-05-22): seed PageSlugs with the
	// locale-variant paths emitted from page_locales overlay rows so
	// ComputeAlternates can find published counterparts even when no
	// literal /<lang>/* page row exists. Each row contributes one
	// path under its locale prefix, using slug_override when set.
	pageLocales, _ := queries.ListPageLocalesBySite(ctx, siteID)
	if len(pageLocales) > 0 {
		// Build page_id -> base slug map once.
		baseByID := make(map[string]string, len(pages))
		for _, p := range pages {
			baseByID[p.ID] = strings.TrimPrefix(p.Slug, "/")
		}
		for _, pl := range pageLocales {
			if strings.ToLower(strings.TrimSpace(pl.Status)) != "published" {
				continue
			}
			base := strings.TrimPrefix(strings.TrimSpace(pl.SlugOverride), "/")
			if base == "" {
				base = baseByID[pl.PageID]
			}
			if base == "" {
				continue
			}
			localePath := "/" + strings.ToLower(pl.Locale) + "/" + base
			slugs[normalizeSlug(localePath)] = true
		}
	}

	defaultLang := strings.ToLower(strings.TrimSpace(site.Lang))
	if defaultLang == "" {
		defaultLang = "en"
	}

	// Sprint 3 (multilingual v1, 2026-05-22): prefer site_locales rows
	// when configured; fall back to the general.additional_langs CSV so
	// pre-Sprint-3 sites keep working. When site_locales has a row
	// with is_default=1, that locale overrides sites.lang as the build
	// default; this lets a tenant change the default locale without
	// touching the site row.
	var addl []string
	siteLocales, _ := queries.ListSiteLocales(ctx, siteID)
	if len(siteLocales) > 0 {
		for _, sl := range siteLocales {
			if sl.IsDefault == 1 {
				defaultLang = strings.ToLower(strings.TrimSpace(sl.Locale))
			}
		}
		for _, sl := range siteLocales {
			loc := strings.ToLower(strings.TrimSpace(sl.Locale))
			if loc != "" && loc != defaultLang {
				addl = append(addl, loc)
			}
		}
	} else {
		addl = splitCSV(sm["general.additional_langs"])
	}
	// De-dup, drop default, normalise to lowercase. We never list the
	// default lang in the additional set; the alternates loop adds it
	// back as a self-referencing link.
	seen := map[string]bool{defaultLang: true}
	clean := make([]string, 0, len(addl))
	for _, l := range addl {
		l = strings.ToLower(strings.TrimSpace(l))
		if l == "" || seen[l] {
			continue
		}
		seen[l] = true
		clean = append(clean, l)
	}

	strategy := strings.ToLower(strings.TrimSpace(sm["seo.hreflang_strategy"]))
	switch strategy {
	case "path", "subdomain", "off":
	default:
		strategy = "path"
	}

	return I18nConfig{
		DefaultLang:     defaultLang,
		AdditionalLangs: clean,
		Strategy:        strategy,
		CanonicalBase:   strings.TrimRight(strings.TrimSpace(sm["seo.canonical_base"]), "/"),
		PageSlugs:       slugs,
	}, nil
}

// LangForSlug returns the locale code for a page slug, derived from
// its prefix when one of AdditionalLangs matches, otherwise DefaultLang.
// Used by the layout to set <html lang> per-page so multi-language
// sites stamp /en/* with lang="en" and /sv/* with lang="sv".
func (c I18nConfig) LangForSlug(slug string) string {
	if len(c.AdditionalLangs) == 0 {
		return c.DefaultLang
	}
	_, lang := stripLocalePrefix(slug, c.DefaultLang, c.AdditionalLangs)
	if lang != "" {
		return lang
	}
	return c.DefaultLang
}

// ComputeAlternates returns the hreflang list for one page slug. The
// list always includes:
//   - one self-referencing entry for the page's own locale
//   - one entry per OTHER locale that has a counterpart page published
//   - an x-default entry pointing at the default-lang version
//
// Empty list when Strategy is "off", when the site is mono-lingual, or
// when the page has no counterparts in any other locale.
func (c I18nConfig) ComputeAlternates(siteDomain, pageSlug string) []HreflangAlternate {
	if c.Strategy == "off" {
		return nil
	}
	if len(c.AdditionalLangs) == 0 {
		return nil
	}

	base := c.CanonicalBase
	if base == "" {
		base = canonicalDomain(siteDomain)
	}
	if base == "" {
		return nil
	}

	// Strip locale prefix from the slug to get the "shared" path.
	shared, sourceLang := stripLocalePrefix(pageSlug, c.DefaultLang, c.AdditionalLangs)

	// First pass: collect the actual counterparts (excluding self). If
	// none exist, emit nothing so we don't pollute pages whose locale
	// counterparts haven't been published yet.
	counterparts := []HreflangAlternate{}
	for _, lang := range c.AdditionalLangs {
		if lang == sourceLang {
			continue
		}
		if !c.localeHasPage(lang, shared) {
			continue
		}
		counterparts = append(counterparts, HreflangAlternate{
			Lang: lang,
			URL:  buildLocaleURL(c, base, lang, shared),
		})
	}
	// Default-lang counterpart, only when the source isn't already default.
	if sourceLang != c.DefaultLang && c.localeHasPage(c.DefaultLang, shared) {
		counterparts = append(counterparts, HreflangAlternate{
			Lang: c.DefaultLang,
			URL:  buildLocaleURL(c, base, c.DefaultLang, shared),
		})
	}
	if len(counterparts) == 0 {
		return nil
	}

	// Self-referencing entry first so Site Inspector's
	// "Hreflang Self-Referencing" check passes.
	out := []HreflangAlternate{
		{
			Lang: sourceLang,
			URL:  buildLocaleURL(c, base, sourceLang, shared),
		},
	}
	out = append(out, counterparts...)

	// Sort everything except the self-reference (which stays first) by lang
	// so the output is deterministic across builds.
	sort.SliceStable(out[1:], func(i, j int) bool {
		return out[i+1].Lang < out[j+1].Lang
	})

	// x-default points at the default-lang version. Skip when there's no
	// default-lang counterpart for this page.
	if c.localeHasPage(c.DefaultLang, shared) {
		out = append(out, HreflangAlternate{
			Lang: "x-default",
			URL:  buildLocaleURL(c, base, c.DefaultLang, shared),
		})
	}
	return out
}

// localeHasPage reports whether the published-pages set contains the page
// at the locale-prefixed slug for the given locale. Path-based check.
func (c I18nConfig) localeHasPage(lang, shared string) bool {
	if c.Strategy == "subdomain" {
		// In subdomain mode there's no in-database way to know whether
		// a sister site at <lang>.example.com has the page. Trust the
		// operator-declared additional_langs list and emit a link
		// optimistically; if the alt 404s, search engines just drop it.
		return true
	}
	target := buildLocalePath(lang, c.DefaultLang, shared)
	return c.PageSlugs[normalizeSlug(target)]
}

// buildLocaleURL turns a locale + shared path into the absolute URL the
// hreflang link should point at.
func buildLocaleURL(c I18nConfig, base, lang, shared string) string {
	switch c.Strategy {
	case "subdomain":
		return localeSubdomainURL(base, lang, shared)
	default:
		return strings.TrimRight(base, "/") + buildLocalePath(lang, c.DefaultLang, shared)
	}
}

// buildLocalePath returns "/<lang>/<shared>" for non-default locales and
// "/<shared>" for the default. shared starts with "/".
//
// Trailing-slash convention: locale roots emit "/<lang>/" (with slash) so
// hreflang URLs match the canonical URL Astro emits for the same page
// ("/<lang>/" with slash). Without this, Site Inspector's "Hreflang
// Self-Referencing" check fails because hreflang="<lang>" points to
// "/<lang>" while canonical points to "/<lang>/".
func buildLocalePath(lang, defaultLang, shared string) string {
	if shared == "" || shared == "/" {
		if lang == defaultLang {
			return "/"
		}
		return "/" + lang + "/"
	}
	if lang == defaultLang {
		return shared
	}
	return "/" + lang + shared
}

// localeSubdomainURL turns https://example.com / sv / /about into
// https://sv.example.com/about. Falls back to the path strategy if the
// base URL doesn't have a parseable host.
func localeSubdomainURL(base, lang, shared string) string {
	scheme, host := splitScheme(base)
	if host == "" {
		return strings.TrimRight(base, "/") + shared
	}
	if shared == "" {
		shared = "/"
	}
	return scheme + lang + "." + host + shared
}

// stripLocalePrefix detects whether a slug starts with /<lang>/ for any
// known locale and returns the shared path + the matched lang. If no
// locale prefix matches, the slug is treated as belonging to the default
// language.
func stripLocalePrefix(slug, defaultLang string, additional []string) (shared, lang string) {
	slug = normalizeSlug(slug)
	if slug == "" || slug == "/" {
		return "/", defaultLang
	}
	parts := strings.SplitN(strings.TrimPrefix(slug, "/"), "/", 2)
	if len(parts) > 0 {
		first := strings.ToLower(parts[0])
		for _, l := range additional {
			if first == l {
				rest := ""
				if len(parts) == 2 {
					rest = "/" + parts[1]
				} else {
					rest = "/"
				}
				return rest, l
			}
		}
	}
	return slug, defaultLang
}

// normalizeSlug ensures a slug starts with "/" and has no trailing slash
// (except the root "/"). Empty -> "/".
func normalizeSlug(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "/"
	}
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	if len(s) > 1 {
		s = strings.TrimRight(s, "/")
	}
	return s
}

// splitCSV splits "sv, de , fr" into ["sv","de","fr"]. Empty entries
// are dropped.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func canonicalDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ""
	}
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}
	return strings.TrimRight(domain, "/")
}

// splitScheme returns ("https://", "example.com") from "https://example.com".
// Returns ("", "") if no scheme.
func splitScheme(u string) (scheme, host string) {
	for _, p := range []string{"https://", "http://"} {
		if strings.HasPrefix(u, p) {
			return p, strings.TrimRight(u[len(p):], "/")
		}
	}
	return "", ""
}

// MetaTemplateVars is the substitution context used when expanding the
// site-level meta_title_template / meta_description_template at page
// render time.
type MetaTemplateVars struct {
	PageTitle       string
	PageDescription string
	SiteName        string
	Lang            string
	Separator       string
}

// ExpandMetaTemplate substitutes {page_title} / {site_name} / {lang} /
// {page_description} / {separator} tokens in tpl. Empty tpl returns
// fallback (the page-level value or the site default). Tokens with empty
// values are stripped along with adjacent " | " or "  " runs so a missing
// {site_name} doesn't leave a trailing pipe.
func ExpandMetaTemplate(tpl string, fallback string, vars MetaTemplateVars) string {
	tpl = strings.TrimSpace(tpl)
	if tpl == "" {
		return fallback
	}
	if vars.Separator == "" {
		vars.Separator = "|"
	}
	repls := map[string]string{
		"{page_title}":       vars.PageTitle,
		"{page_description}": vars.PageDescription,
		"{site_name}":        vars.SiteName,
		"{lang}":             vars.Lang,
		"{separator}":        vars.Separator,
	}
	out := tpl
	for k, v := range repls {
		out = strings.ReplaceAll(out, k, v)
	}
	// Collapse separator runs left over by empty tokens. Common patterns:
	// "  | Brand", "Title | ", "  ".
	for _, dup := range []string{
		" | | ", "| | ", " | |", "||",
		"   ", "  ",
	} {
		for strings.Contains(out, dup) {
			out = strings.ReplaceAll(out, dup, strings.TrimSpace(dup[:len(dup)/2+1])+" ")
		}
	}
	out = strings.TrimSpace(out)
	out = strings.Trim(out, "|")
	out = strings.TrimSpace(out)
	if out == "" {
		return fallback
	}
	return out
}

// hreflangPropExpression turns alternates into the JSX array literal
// renderPage embeds in <Base alternates={...}>. Each entry escapes the
// URL via escapeAttr (the same helper used elsewhere in builder).
func hreflangPropExpression(alts []HreflangAlternate) string {
	if len(alts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[")
	for i, a := range alts {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf(`{lang: %q, url: %q}`, a.Lang, a.URL))
	}
	b.WriteString("]")
	return b.String()
}
