package knowledge

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed docs/*.md
var embedded embed.FS

// docs is the resolved catalog, populated at init from the metadata
// table below cross-referenced with the embedded markdown files.
//
// Adding a new doc is two steps:
//  1. Drop the markdown at docs/<slug>.md with a top-level "# Title".
//  2. Add an entry to docMetadata below with the slug, category, order,
//     and a one-line summary that surfaces in the catalog.
//
// init panics if metadata references a missing file or vice versa, so
// drift between the catalog and the filesystem fails the build, not
// production.
var docs map[string]Doc

// docMetadata is the authoritative ordering and categorisation. Body
// and Title come from the embedded markdown so the agent reads exactly
// what an author wrote.
type docMetadata struct {
	slug     string
	category Category
	order    int
	summary  string
}

var docMetadataTable = []docMetadata{
	// Stack curriculum (expand as docs land in docs/<slug>.md).
	{"astro-conventions", CategoryStack, 10, "Astro 4 file routing, layout slots, hydration directives, what the builder writes vs author-supplied."},
	{"typescript-strict", CategoryStack, 20, "Strict-mode TypeScript patterns for Astro components: no any, type-only imports, prop typing."},
	{"css-variable-system", CategoryStack, 30, "The 6 color slots, container widths, breakpoints, how custom properties drive the entire visual system."},
	{"block-renderer-patterns", CategoryStack, 40, "The 19 block types, their schema kinds, how to extend with a custom block, when to use raw_astro."},
	{"collection-design-patterns", CategoryStack, 45, "Custom Collections (CMS layer): when to pick Collection vs Component vs global block, schema design, slug + locale model, agent bulk-import flow."},
	{"schema-org-per-collection-type", CategoryStack, 47, "Field-name conventions per schema.org type (Article, Product, Person, Event, JobPosting, Recipe, FAQPage, Review) so JSON-LD auto-emits."},
	{"i18n-authoring", CategoryStack, 50, "Path vs subdomain vs off strategies, hreflang trailing-slash rule, default-lang routing."},
	{"security-authoring", CategoryStack, 60, "The 11 A+ headers, when to whitelist via allowed_scripts vs csp_extra_directives, why raw_astro is admin-only."},
	{"personalization", CategoryStack, 70, "Visitor-state hooks the builder leaves available, the do-not-touch boundary on identified-tier PII."},
	{"cookieproof-integration", CategoryStack, 80, "IMY 2026 equal-prominence rule, cookie-table model, language toggle, override surface."},
	{"analytics-conversions-and-identify", CategoryStack, 85, "Conversion goals (3 match types), atomic.track JS API, /t/identify + form-auto-identify, end-to-end attribution chain BrightCRM email -> click -> pageview -> form -> CRM."},

	// UX curriculum: the discipline that makes output feel premium.
	{"typography-scale", CategoryUX, 10, "Inter/Geist/JetBrains-Mono baseline, modular scale, line-heights, font-display:swap, woff2-only policy."},
	{"color-system", CategoryUX, 20, "Palette construction from a primary, neutrals vs accents, contrast ratios, dark-mode token derivation."},
	{"spacing-rhythm", CategoryUX, 30, "4/8/16 base scale, vertical rhythm, when to break the grid intentionally."},
	{"motion-curves", CategoryUX, 40, "Timing tokens, easing curves, hardware-accelerated transforms, prefers-reduced-motion."},
	{"accessibility-patterns", CategoryUX, 50, "Landmarks, focus-visible, skip links, form labels, JS-injected anchor labels."},
	{"performance-budgets", CategoryUX, 60, "HTML/CSS size limits, async/defer scripts, lazy-load below fold, LCP image hints, what eval measures."},
	{"forms-ux", CategoryUX, 70, "Autocomplete attributes, error states, single-column layout, validation timing."},
	{"nav-ux", CategoryUX, 80, "Header-scroll behaviour, mobile breakpoint patterns, sticky nav rules, in-page anchor scroll-margin."},
	{"dark-mode", CategoryUX, 90, "Token-pair patterns, theme toggle UX, [data-theme] attribute, preserving brand identity."},
	{"premium-design-principles", CategoryUX, 100, "What separates AI-generic from high-end agency: asymmetric layouts, type-scale contrast, calibrated color, perpetual micro-motion."},
}

func init() {
	docs = make(map[string]Doc, len(docMetadataTable))
	seen := map[string]bool{}

	// Walk the embedded fs to validate every metadata entry has a file
	// and every file has a metadata entry. Both sides must match.
	files := map[string]string{}
	_ = fs.WalkDir(embedded, "docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		name := strings.TrimSuffix(strings.TrimPrefix(path, "docs/"), ".md")
		body, rerr := embedded.ReadFile(path)
		if rerr != nil {
			panic(fmt.Sprintf("knowledge: read %s: %v", path, rerr))
		}
		files[name] = string(body)
		return nil
	})

	for _, m := range docMetadataTable {
		body, ok := files[m.slug]
		if !ok {
			panic(fmt.Sprintf("knowledge: metadata references missing file docs/%s.md", m.slug))
		}
		if seen[m.slug] {
			panic(fmt.Sprintf("knowledge: duplicate slug %q in metadata", m.slug))
		}
		seen[m.slug] = true
		title := firstHeading(body)
		if title == "" {
			panic(fmt.Sprintf("knowledge: docs/%s.md is missing a top-level # heading", m.slug))
		}
		docs[m.slug] = Doc{
			Slug:     m.slug,
			Title:    title,
			Category: m.category,
			Order:    m.order,
			Summary:  m.summary,
			Body:     body,
		}
	}
	for name := range files {
		if !seen[name] {
			panic(fmt.Sprintf("knowledge: docs/%s.md has no metadata entry; add one to docMetadataTable", name))
		}
	}
}
