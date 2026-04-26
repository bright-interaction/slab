package perfectfoundation

import (
	"context"

	"github.com/bright-interaction/slab/internal/store"
)

// referenceKnowledgebase is the canonical knowledgebase derived from
// brightinteraction.com (140/140 across 5 Site Inspector categories). Every
// new site auto-seeds this on creation so any AI agent has the patterns
// needed to keep that score, regardless of which starter kit they chose.
//
// One entry per "teach" check in CheckOwnership, plus a few cross-cutting
// pattern entries (when the same advice covers multiple checks).
var referenceKnowledgebase = []KBEntry{
	{
		Category: "seo",
		Title:    "Page meta title and description rules",
		Content: `Every page needs:
- title: 30 to 60 characters, page-specific (NOT just the site name).
- meta_description: 120 to 160 characters, includes a verb / call to action.

Format the title as "Specific value | Brand". Example from brightinteraction.com: "Replace SaaS with Open Source You Own | Bright Interaction" (58 chars).

Atomicsite's guardrails will refuse a Page create/update with title or description outside these ranges.`,
	},
	{
		Category: "seo",
		Title:    "Headings (kb.go:headings)",
		Content: `Every page must have exactly one <h1> with non-empty text. Subsections use <h2>, then <h3>, etc — never skip a level.

The H1 should match the user's primary intent for the page (not the brand name). Use 1 H2 per ~300 words of content (this also satisfies the AI-friendly-formatting check).`,
	},
	{
		Category: "seo",
		Title:    "Internal linking (kb.go:internal-links)",
		Content: `Every non-leaf page should link to at least 2 other pages on the site, with descriptive anchor text. Avoid "click here", "read more", "this link". Anchor text should describe the destination ("View pricing", "How the scanner works", "Read the GDPR guide").

The eval engine flags pages with < 2 internal links and pages with generic anchor text.`,
	},
	{
		Category: "seo",
		Title:    "Robots directives (kb.go:robots)",
		Content: `Default robots value is "index, follow, max-snippet:-1, max-image-preview:large, max-video-preview:-1". Atomicsite bakes this into every Layout.astro automatically.

Use "noindex, nofollow" only when:
- The page is intentionally hidden (e.g., demo pages, internal landing pages from cold outreach).
- A page is an orphan (not linked from any silo) — atomicsite auto-noindexes orphans.

Never noindex the homepage. Never noindex pages in the main sitemap.`,
	},
	{
		Category: "geo",
		Title:    "BreadcrumbList schema (kb.go:schema-breadcrumb)",
		Content: `Every page deeper than the homepage emits a BreadcrumbList JSON-LD with at least 2 items (Home → current page) so AI search engines can build the navigation context.

Atomicsite's builder generates this automatically from the page's URL path. The home page is exempt and the eval engine skips the check there.`,
	},
	{
		Category: "geo",
		Title:    "Content freshness signal (kb.go:content-freshness)",
		Content: `Articles, blog posts, and news items must include a dateModified ISO-8601 timestamp in their JSON-LD (Article / BlogPosting / NewsArticle schema). Pages older than 18 months without an update are flagged.

When you edit an article, set its updated_at — atomicsite emits this as schema.org dateModified automatically. Do NOT bulk-update the modified date across many articles in one batch — Google flags that as manipulation.`,
	},
	{
		Category: "geo",
		Title:    "AI-friendly formatting (kb.go:ai-formatting)",
		Content: `LLMs reward content that is easy to parse:
- Lists with 3 or more items (use <ul> or <ol>, not bullet-emoji lines).
- Tables with proper <th> headers.
- Definition lists (<dl>) for FAQs or glossary content.
- Headings every ~300 words.

A page passes if it has any one of these. The eval engine fails pages that read as a single wall of text with no structure.`,
	},
	{
		Category: "performance",
		Title:    "HTML size budget (kb.go:html-size)",
		Content: `Keep each rendered HTML page under 200KB. The eval engine fails larger pages.

Common culprits:
- Inline base64 images (always upload to media library, never paste data: URIs).
- Pasted SVG icons rendered server-side instead of referenced from a sprite.
- Long product / data lists rendered in full instead of paginated.

Atomicsite's image pipeline auto-converts to WebP/AVIF and emits responsive variants — never bake images into HTML.`,
	},
	{
		Category: "performance",
		Title:    "Render-blocking scripts (kb.go:render-blocking)",
		Content: `Every <script> in <head> must have async or defer (or be type="module"). Inline scripts block paint until they parse.

Atomicsite's Layout.astro adds defer to first-party scripts automatically. For third-party scripts, add them via the allowed-scripts settings page — atomicsite emits them with defer + integrity SRI.

Heavy SDKs (Cal.com, Stripe Elements, chat widgets) should be lazy-loaded on user interaction (button click, scroll-into-view) rather than on page load.`,
	},
	{
		Category: "performance",
		Title:    "Lazy-loaded images (kb.go:lazy-images)",
		Content: `Images below the fold should have loading="lazy". Above-the-fold images (hero, LCP candidate) should NOT be lazy — they need priority.

Atomicsite components use loading="lazy" by default for image blocks. Hero images are excluded. Add fetchpriority="high" to the LCP image when known.`,
	},
	{
		Category: "performance",
		Title:    "Inline CSS budget (kb.go:inline-css)",
		Content: `Inline <style> blocks must stay under 50KB. Use them only for critical CSS that affects above-the-fold render. Everything else goes in a global CSS class (managed via atomicsite's CSS Classes page).

Atomicsite's builder inlines critical CSS automatically and externalises the rest.`,
	},
	{
		Category: "accessibility",
		Title:    "Landmarks (kb.go:landmarks)",
		Content: `Every page needs four landmarks: <header> (or role="banner"), <nav>, <main>, <footer> (or role="contentinfo"). Screen readers use these to skip between sections.

Atomicsite's default Layout.astro provides header / nav / main / footer wrappers. When you add a top-level component, never use <div> where a semantic tag fits — <main> contains the page's primary content, not the global header.`,
	},
	{
		Category: "accessibility",
		Title:    "Form labels (kb.go:form-labels)",
		Content: `Every <input>, <textarea>, <select> needs an associated label. Three options:
- <label for="email">Email</label> with matching id.
- Wrap: <label>Email <input type="email" /></label>
- aria-label="Email" or aria-labelledby="some-id".

Placeholder text is NOT a label — it disappears once the user types. Atomicsite's form components emit proper <label for>. When you build custom forms, follow the same pattern.`,
	},
	{
		Category: "accessibility",
		Title:    "Tabindex (kb.go:tabindex)",
		Content: `Never use tabindex="1", "2", "3" etc — positive tabindex breaks keyboard navigation. Only valid values are "0" (focusable in DOM order) and "-1" (programmatically focusable, not in tab order).

If keyboard order needs to differ from DOM order, restructure the DOM instead of overriding with tabindex.`,
	},
	{
		Category: "accessibility",
		Title:    "Table headers (kb.go:table-headers)",
		Content: `Data tables (not layout tables) need <th> cells in the header row. Use <th scope="col"> for column headers and <th scope="row"> for row headers. Screen readers use scope to associate cells with their headers.

For purely visual layout grids, use CSS grid / flex — never <table>.`,
	},
	{
		Category: "accessibility",
		Title:    "JS-injected anchor labels",
		Content: `If a link's href and visible text are injected by JavaScript at runtime (anti-scraping pattern for emails / phone numbers), the static HTML has an empty <a>. Add an aria-label so the link still has an accessible name.

Example: <a data-contact-email aria-label="Send email to Acme Inc"></a>. The eval engine flags JS-empty links without aria-label.`,
	},
	{
		Category: "security",
		Title:    "Subresource Integrity (kb.go:sri)",
		Content: `Third-party scripts loaded from a CDN must include integrity="sha384-..." so the browser refuses tampered files. Atomicsite computes SRI at build time for first-party and known-stable third-party URLs.

Skip SRI for:
- Analytics scripts that ship without versioned URLs (gtag.js, posthog) — they change without warning.
- Scripts loaded via consent gating after page load.

The eval engine knows about these exceptions and won't flag them.`,
	},
	{
		Category: "privacy",
		Title:    "Tracker count (kb.go:tracker-count)",
		Content: `Keep third-party trackers below 3 per page. The eval engine warns at 3+, fails at 5+.

Common excessive setups:
- Google Analytics + GTM + Facebook Pixel + LinkedIn Insight + Hotjar.

Atomicsite's pattern: one analytics provider (Umami or GA4 via consent gating), one consent platform (CookieProof). Add others only with explicit business justification.`,
	},
	{
		Category: "privacy",
		Title:    "AI training bots blocked",
		Content: `robots.txt blocks GPTBot, ChatGPT-User, anthropic-ai, Google-Extended, and CCBot by default. This prevents your site content from being scraped for LLM training without consent.

Override only if you actively want your content in LLM training corpora (rare). Atomicsite's RenderRobotsTxt baker handles this automatically.`,
	},
	{
		Category: "schema",
		Title:    "Organization JSON-LD completeness",
		Content: `The Organization schema baked into every page must include: name, url, logo (with width/height), sameAs (>=1 social URL), and one of {founder, address, contactPoint}.

Atomicsite reads these from:
- Site profile (BusinessName, address, contact).
- Settings: seo.same_as (newline-separated social URLs), seo.logo_url.

Fill those in during onboarding to pass the GEO check on first build.`,
	},
	{
		Category: "schema",
		Title:    "FAQ schema must match visible content",
		Content: `If a page emits FAQPage JSON-LD, the questions and answers in the schema must match the visible <details> / <h3>+<p> content on the page. Schema-only FAQ (with no visible counterpart) is a Google penalty.

Atomicsite's FAQ component generates both visible content and matching schema from one data source — never edit one without the other.`,
	},
	{
		Category: "schema",
		Title:    "ProfessionalService / LocalBusiness schema",
		Content: `Service businesses should add ProfessionalService (B2B) or LocalBusiness (B2C) schema with: name, address (PostalAddress), telephone, areaServed, hasOfferCatalog (with itemListElement Service Offers).

For local businesses, add openingHoursSpecification and geo (latitude/longitude) for map pack visibility.`,
	},
	{
		Category: "design",
		Title:    "Font loading pattern",
		Content: `For custom fonts:
1. Self-host woff2 files in /public/fonts/.
2. Preload above-the-fold variants: <link rel="preload" as="font" type="font/woff2" crossorigin fetchpriority="high">.
3. @font-face with font-display:swap so text renders immediately with a fallback.
4. Use system-ui as the ultimate fallback in your font-family stack.

Never @import from Google Fonts at build time — it adds a render-blocking request. Atomicsite's onboarding generates these preloads automatically when you set Font Heading / Font Body.`,
	},
	{
		Category: "design",
		Title:    "Image dimensions are mandatory",
		Content: `Every <img> needs width and height attributes (or aspect-ratio CSS) so the browser reserves layout space before the image loads. Without them, the page reflows when images arrive — Cumulative Layout Shift (CLS) penalty + a failing eval check.

Atomicsite's image upload pipeline records dimensions at upload time and fills width/height attributes automatically. The block-level guardrail will refuse image blocks without dimensions.`,
	},
	{
		Category: "ai-search",
		Title:    "llms.txt — what to include",
		Content: `Atomicsite generates /llms.txt automatically with: business name, founder, key pages, services, target audience, technical stack, and E-E-A-T signals (Experience, Expertise, Authoritativeness, Trust).

Treat llms.txt like a README for AI crawlers. Update it when:
- You add a major service / product.
- You publish a new pillar article worth citing.
- Your founder bio or credentials change.

Brief, honest, fact-dense. No marketing fluff — LLMs prefer specifics.`,
	},
}

// SeedReferenceKnowledgebase seeds the brightinteraction.com-derived
// reference KB on every new site. Replaces the old 5-entry default seed.
func SeedReferenceKnowledgebase(ctx context.Context, q *store.Queries, siteID string) error {
	for i, e := range referenceKnowledgebase {
		if err := q.CreateKnowledgebaseEntry(ctx, store.CreateKnowledgebaseEntryParams{
			ID:        newID(),
			SiteID:    siteID,
			Category:  e.Category,
			Title:     e.Title,
			Content:   e.Content,
			SortOrder: int64(i),
		}); err != nil {
			return err
		}
	}
	return nil
}

// KBEntry is one knowledgebase row to seed.
type KBEntry struct {
	Category string
	Title    string
	Content  string
}

// ReferenceEntries returns the static list of reference KB entries (read-only).
// Tests use this; runtime callers use SeedReferenceKnowledgebase.
func ReferenceEntries() []KBEntry {
	out := make([]KBEntry, len(referenceKnowledgebase))
	copy(out, referenceKnowledgebase)
	return out
}
