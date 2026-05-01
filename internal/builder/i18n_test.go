package builder

import (
	"strings"
	"testing"
)

// TestComputeAlternates_PathStrategy_HappyPath is the canonical case.
// English default, Swedish + German additional. /about has counterparts
// at /sv/about and /de/about; we expect three hreflang entries
// (self-ref + two locales) plus an x-default pointing at the default
// version.
func TestComputeAlternates_PathStrategy_HappyPath(t *testing.T) {
	c := I18nConfig{
		DefaultLang:     "en",
		AdditionalLangs: []string{"sv", "de"},
		Strategy:        "path",
		PageSlugs: map[string]bool{
			"/about":    true,
			"/sv/about": true,
			"/de/about": true,
		},
	}
	got := c.ComputeAlternates("example.com", "/about")
	if len(got) != 4 {
		t.Fatalf("expected 4 entries (self + 2 locales + x-default), got %d: %#v", len(got), got)
	}
	// Self-ref first
	if got[0].Lang != "en" || got[0].URL != "https://example.com/about" {
		t.Errorf("self-ref wrong: %+v", got[0])
	}
	// Locales sorted alphabetically after self-ref
	if got[1].Lang != "de" || got[2].Lang != "sv" {
		t.Errorf("expected de then sv after self-ref, got %s, %s", got[1].Lang, got[2].Lang)
	}
	// x-default last
	if got[3].Lang != "x-default" || got[3].URL != "https://example.com/about" {
		t.Errorf("x-default wrong: %+v", got[3])
	}
}

// TestComputeAlternates_FromLocaleSubpage covers the inverse: a Swedish
// page at /sv/about should still link back to the English default and
// include itself.
func TestComputeAlternates_FromLocaleSubpage(t *testing.T) {
	c := I18nConfig{
		DefaultLang:     "en",
		AdditionalLangs: []string{"sv"},
		Strategy:        "path",
		PageSlugs: map[string]bool{
			"/about":    true,
			"/sv/about": true,
		},
	}
	got := c.ComputeAlternates("example.com", "/sv/about")
	if len(got) != 3 {
		t.Fatalf("expected 3 entries (self + en + x-default), got %d: %#v", len(got), got)
	}
	if got[0].Lang != "sv" || got[0].URL != "https://example.com/sv/about" {
		t.Errorf("self-ref wrong: %+v", got[0])
	}
	hasEN := false
	hasXDefault := false
	for _, a := range got[1:] {
		if a.Lang == "en" && a.URL == "https://example.com/about" {
			hasEN = true
		}
		if a.Lang == "x-default" && a.URL == "https://example.com/about" {
			hasXDefault = true
		}
	}
	if !hasEN || !hasXDefault {
		t.Errorf("expected en + x-default in alternates, got: %#v", got)
	}
}

// TestComputeAlternates_OnlyEmitsForExistingPages confirms a page with no
// counterparts in any other locale gets an empty list. We never want to
// emit hreflang to a 404 page.
func TestComputeAlternates_OnlyEmitsForExistingPages(t *testing.T) {
	c := I18nConfig{
		DefaultLang:     "en",
		AdditionalLangs: []string{"sv", "de"},
		Strategy:        "path",
		PageSlugs: map[string]bool{
			"/about": true, // only English version exists
		},
	}
	got := c.ComputeAlternates("example.com", "/about")
	if len(got) != 0 {
		t.Errorf("expected no alternates when no counterparts exist, got: %#v", got)
	}
}

// TestComputeAlternates_StrategyOff suppresses everything regardless of
// counterparts. Operators who manage hreflang manually use this.
func TestComputeAlternates_StrategyOff(t *testing.T) {
	c := I18nConfig{
		DefaultLang:     "en",
		AdditionalLangs: []string{"sv"},
		Strategy:        "off",
		PageSlugs: map[string]bool{
			"/about":    true,
			"/sv/about": true,
		},
	}
	got := c.ComputeAlternates("example.com", "/about")
	if len(got) != 0 {
		t.Errorf("strategy=off should suppress alternates, got: %#v", got)
	}
}

// TestComputeAlternates_SubdomainStrategy emits sv.example.com URLs and
// trusts the operator's additional_langs list (no published-page check
// since cross-site visibility isn't possible).
func TestComputeAlternates_SubdomainStrategy(t *testing.T) {
	c := I18nConfig{
		DefaultLang:     "en",
		AdditionalLangs: []string{"sv"},
		Strategy:        "subdomain",
		PageSlugs: map[string]bool{
			"/about": true,
		},
	}
	got := c.ComputeAlternates("https://example.com", "/about")
	if len(got) < 2 {
		t.Fatalf("expected self + sv at minimum, got %d: %#v", len(got), got)
	}
	hasSV := false
	for _, a := range got {
		if a.Lang == "sv" && a.URL == "https://sv.example.com/about" {
			hasSV = true
		}
	}
	if !hasSV {
		t.Errorf("expected sv subdomain URL, got: %#v", got)
	}
}

// TestComputeAlternates_CanonicalBaseOverride lets operators set a CDN-
// rooted canonical base. Hreflang URLs should respect it.
func TestComputeAlternates_CanonicalBaseOverride(t *testing.T) {
	c := I18nConfig{
		DefaultLang:     "en",
		AdditionalLangs: []string{"sv"},
		Strategy:        "path",
		CanonicalBase:   "https://cdn.example.com",
		PageSlugs: map[string]bool{
			"/about":    true,
			"/sv/about": true,
		},
	}
	got := c.ComputeAlternates("example.com", "/about")
	for _, a := range got {
		if !strings.HasPrefix(a.URL, "https://cdn.example.com") {
			t.Errorf("alternate %q didn't honor canonical base: %#v", a.Lang, a)
		}
	}
}

// TestComputeAlternates_HomePage handles the / case, where the locale
// path becomes just "/sv" (no trailing slug).
func TestComputeAlternates_HomePage(t *testing.T) {
	c := I18nConfig{
		DefaultLang:     "en",
		AdditionalLangs: []string{"sv"},
		Strategy:        "path",
		PageSlugs: map[string]bool{
			"/":   true,
			"/sv": true,
		},
	}
	got := c.ComputeAlternates("example.com", "/")
	if len(got) < 2 {
		t.Fatalf("expected self + sv on homepage, got %d: %#v", len(got), got)
	}
	hasSelf := false
	hasSV := false
	for _, a := range got {
		if a.Lang == "en" && a.URL == "https://example.com/" {
			hasSelf = true
		}
		if a.Lang == "sv" && a.URL == "https://example.com/sv/" {
			hasSV = true
		}
	}
	if !hasSelf || !hasSV {
		t.Errorf("expected self + sv homepage entries, got: %#v", got)
	}
}

// TestExpandMetaTemplate_BasicSubstitution is the "About | Acme" case.
func TestExpandMetaTemplate_BasicSubstitution(t *testing.T) {
	got := ExpandMetaTemplate(
		"{page_title} | {site_name}",
		"About",
		MetaTemplateVars{PageTitle: "About us", SiteName: "Acme"},
	)
	want := "About us | Acme"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

// TestExpandMetaTemplate_FallbackOnEmptyTemplate returns the page-level
// value verbatim when the operator has no template set.
func TestExpandMetaTemplate_FallbackOnEmptyTemplate(t *testing.T) {
	got := ExpandMetaTemplate("", "About us", MetaTemplateVars{SiteName: "Acme"})
	if got != "About us" {
		t.Errorf("expected fallback, got %q", got)
	}
}

// TestExpandMetaTemplate_StripsLeftoverSeparators handles the common
// "{page_title} | {site_name}" template when site_name is unset, so the
// output stays clean instead of "About | ".
func TestExpandMetaTemplate_StripsLeftoverSeparators(t *testing.T) {
	got := ExpandMetaTemplate(
		"{page_title} | {site_name}",
		"About",
		MetaTemplateVars{PageTitle: "About"},
	)
	if got != "About" {
		t.Errorf("expected trailing pipe stripped, got %q", got)
	}
}

// TestExpandMetaTemplate_FallbackOnEmptyResult kicks the fallback in
// when every token resolves to empty (e.g. "{page_title}" with no
// page_title supplied).
func TestExpandMetaTemplate_FallbackOnEmptyResult(t *testing.T) {
	got := ExpandMetaTemplate("{page_title}", "Default", MetaTemplateVars{})
	if got != "Default" {
		t.Errorf("expected fallback when all tokens empty, got %q", got)
	}
}

// TestHreflangPropExpression renders a deterministic JSX-ish array.
// Order matters because the per-page Astro file embeds this as an
// array literal that React/Astro will pass to the layout.
func TestHreflangPropExpression(t *testing.T) {
	got := hreflangPropExpression([]HreflangAlternate{
		{Lang: "en", URL: "https://example.com/"},
		{Lang: "sv", URL: "https://example.com/sv"},
	})
	want := `[{lang: "en", url: "https://example.com/"}, {lang: "sv", url: "https://example.com/sv"}]`
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
