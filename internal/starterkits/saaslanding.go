package starterkits

import (
	"context"
	"fmt"

	"github.com/brightinteraction/atomicsite/internal/store"
)

func init() {
	Default.Register(&saasLandingKit{})
}

type saasLandingKit struct{}

func (k *saasLandingKit) ID() string          { return "saas-landing" }
func (k *saasLandingKit) Name() string        { return "SaaS Landing Page" }
func (k *saasLandingKit) Description() string {
	return "Modern English SaaS marketing site. Hero, features, pricing, testimonials, FAQ, CTA, docs hub."
}
func (k *saasLandingKit) TargetSiteTypes() []string { return []string{"b2b", "b2c"} }

func (k *saasLandingKit) Apply(ctx context.Context, q *store.Queries, siteID string) error {
	if err := k.applyComponents(ctx, q, siteID); err != nil {
		return fmt.Errorf("saas-landing components: %w", err)
	}
	if err := k.applyCSS(ctx, q, siteID); err != nil {
		return fmt.Errorf("saas-landing css: %w", err)
	}
	if err := k.applyKnowledgebase(ctx, q, siteID); err != nil {
		return fmt.Errorf("saas-landing kb: %w", err)
	}
	if err := k.applyGuardrails(ctx, q, siteID); err != nil {
		return fmt.Errorf("saas-landing guardrails: %w", err)
	}
	if err := k.applyPages(ctx, q, siteID); err != nil {
		return fmt.Errorf("saas-landing pages: %w", err)
	}
	return nil
}

func (k *saasLandingKit) applyComponents(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []componentDef{
		{
			Name:     "hero-saas",
			Category: "hero",
			Template: `---
const { eyebrow, heading, subheading, ctaLabel, ctaHref, secondaryLabel, secondaryHref } = Astro.props;
---
<section class="hero-saas">
  <div class="container">
    {eyebrow && <p class="eyebrow">{eyebrow}</p>}
    <h1>{heading}</h1>
    <p class="subheading">{subheading}</p>
    <p class="actions">
      <a class="btn gradient-cta" href={ctaHref}>{ctaLabel}</a>
      {secondaryHref && <a class="btn btn-ghost" href={secondaryHref}>{secondaryLabel}</a>}
    </p>
  </div>
</section>`,
			Props:      `{"type":"object","required":["heading","subheading","ctaLabel","ctaHref"],"properties":{"eyebrow":{"type":"string"},"heading":{"type":"string"},"subheading":{"type":"string"},"ctaLabel":{"type":"string"},"ctaHref":{"type":"string"},"secondaryLabel":{"type":"string"},"secondaryHref":{"type":"string"}}}`,
			CSSClasses: `["hero-saas","gradient-cta"]`,
			Usage:      "SaaS hero with eyebrow, heading, subheading, primary and secondary CTA.",
		},
		{
			Name:     "feature-grid",
			Category: "feature",
			Template: `---
const { heading, features = [] } = Astro.props;
---
<section class="features">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <ul class="feature-grid">
      {features.map((f) => (
        <li class="feature-tile">
          {f.icon && <span class="icon">{f.icon}</span>}
          <h3>{f.title}</h3>
          <p>{f.body}</p>
        </li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"features":{"type":"array","items":{"type":"object","required":["title","body"],"properties":{"title":{"type":"string"},"body":{"type":"string"},"icon":{"type":"string"}}}}}}`,
			CSSClasses: `["feature-tile"]`,
			Usage:      "Feature grid with icon, title, body.",
		},
		{
			Name:     "pricing-table",
			Category: "pricing",
			Template: `---
const { heading, tiers = [] } = Astro.props;
---
<section class="pricing">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <ul class="pricing-table">
      {tiers.map((t) => (
        <li class={t.featured ? "pricing-tier featured" : "pricing-tier"}>
          <h3>{t.name}</h3>
          <p class="price">{t.price}</p>
          <ul class="features">
            {(t.features || []).map((f) => <li>{f}</li>)}
          </ul>
          <a class="btn gradient-cta" href={t.ctaHref}>{t.ctaLabel}</a>
        </li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"tiers":{"type":"array","items":{"type":"object","required":["name","price","ctaLabel","ctaHref"],"properties":{"name":{"type":"string"},"price":{"type":"string"},"features":{"type":"array","items":{"type":"string"}},"ctaLabel":{"type":"string"},"ctaHref":{"type":"string"},"featured":{"type":"boolean"}}}}}}`,
			CSSClasses: `["pricing-tier"]`,
			Usage:      "Three-tier pricing table with optional featured tier.",
		},
		{
			Name:     "testimonial-carousel",
			Category: "social_proof",
			Template: `---
const { items = [] } = Astro.props;
---
<section class="testimonials">
  <div class="container">
    <ul class="testimonial-carousel">
      {items.map((t) => (
        <li class="testimonial">
          <blockquote>{t.quote}</blockquote>
          <p class="cite">{t.author}, {t.role}</p>
        </li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"items":{"type":"array","items":{"type":"object","required":["quote","author"],"properties":{"quote":{"type":"string"},"author":{"type":"string"},"role":{"type":"string"}}}}}}`,
			CSSClasses: `["testimonial-carousel"]`,
			Usage:      "Customer testimonial carousel with quote, author and role.",
		},
		{
			Name:     "faq-accordion",
			Category: "faq",
			Template: `---
const { heading, items = [] } = Astro.props;
---
<section class="faq">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    {items.map((it) => (
      <details class="faq-item">
        <summary>{it.q}</summary>
        <p>{it.a}</p>
      </details>
    ))}
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"items":{"type":"array","items":{"type":"object","required":["q","a"],"properties":{"q":{"type":"string"},"a":{"type":"string"}}}}}}`,
			CSSClasses: `["faq-item"]`,
			Usage:      "Native disclosure FAQ accordion.",
		},
		{
			Name:     "metric-card",
			Category: "metric",
			Template: `---
const { value, label, delta } = Astro.props;
---
<div class="metric-card">
  <p class="value">{value}</p>
  <p class="label">{label}</p>
  {delta && <p class="delta">{delta}</p>}
</div>`,
			Props:      `{"type":"object","required":["value","label"],"properties":{"value":{"type":"string"},"label":{"type":"string"},"delta":{"type":"string"}}}`,
			CSSClasses: `["metric-card"]`,
			Usage:      "Single metric card for hero or features section.",
		},
	}
	return seedComponents(ctx, q, siteID, defs)
}

func (k *saasLandingKit) applyCSS(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []cssDef{
		{
			Name:     ".gradient-cta",
			Category: "button",
			CSS:      ".gradient-cta { background: linear-gradient(90deg, var(--accent, #6366f1), var(--accent-2, #8b5cf6)); color: #fff; padding: 0.75rem 1.5rem; border-radius: 0.5rem; font-weight: 600; }",
			Usage:    "Gradient primary CTA button.",
		},
		{
			Name:     ".metric-card",
			Category: "card",
			CSS:      ".metric-card { display: flex; flex-direction: column; gap: 0.25rem; padding: 1rem; border-radius: 0.75rem; background: var(--bg-soft, #f8fafc); } .metric-card .value { font-size: 2.25rem; font-weight: 700; line-height: 1; } .metric-card .label { color: var(--muted, #64748b); }",
			Usage:    "Metric card with value, label, optional delta.",
		},
		{
			Name:     ".feature-tile",
			Category: "card",
			CSS:      ".feature-tile { padding: 1.5rem; border: 1px solid var(--border, #e2e8f0); border-radius: 0.75rem; } .feature-tile .icon { display: inline-flex; width: 2.5rem; height: 2.5rem; align-items: center; justify-content: center; border-radius: 0.5rem; background: var(--accent-soft, #eef2ff); margin-bottom: 0.75rem; }",
			Usage:    "Feature tile with icon, title, body.",
		},
	}
	return seedCSS(ctx, q, siteID, defs)
}

func (k *saasLandingKit) applyKnowledgebase(ctx context.Context, q *store.Queries, siteID string) error {
	entries := []kbDef{
		{Category: "brand", Title: "Brand voice", Content: "Concise, technical, benefit-led. Lead with the outcome, follow with the mechanism. Avoid filler. No marketing jargon. No 'revolutionary'."},
		{Category: "brand", Title: "Target audience", Content: "Developers and small product teams shipping fast. They want proof, not promises. Code examples and pricing transparency win."},
		{Category: "technical", Title: "Pricing page rules", Content: "Always show three tiers. Always include a free or starter tier. The middle tier is the recommended one and gets the featured visual treatment. Show the price prominently."},
		{Category: "technical", Title: "Docs structure", Content: "Docs live under /docs as a silo hub with intro, quickstart, reference and guides. Every doc page has a working code example near the top."},
	}
	return seedKB(ctx, q, siteID, entries, 100)
}

func (k *saasLandingKit) applyGuardrails(ctx context.Context, q *store.Queries, siteID string) error {
	rules := []guardrailDef{
		{RuleType: "allow_block_type", Target: "*", Value: "hero,text,image,cta,feature_grid,pricing_table,testimonial,faq,metric_card", Severity: "error"},
		{RuleType: "forbid_pattern", Target: "block.data.text", Value: "lorem ipsum", Severity: "error"},
		{RuleType: "require_block", Target: "page.layout=landing", Value: "cta", Severity: "warning"},
	}
	return seedGuardrails(ctx, q, siteID, rules)
}

func (k *saasLandingKit) applyPages(ctx context.Context, q *store.Queries, siteID string) error {
	// Home page
	homeID := newKitID()
	if err := q.CreatePage(ctx, store.CreatePageParams{
		ID:        homeID,
		SiteID:    siteID,
		Title:     "Home",
		Slug:      "/",
		Layout:    "landing",
		SortOrder: 0,
		ShowInNav: 1,
	}); err != nil {
		return err
	}
	homeBlocks := []blockDef{
		{BlockType: "hero", Data: `{"component":"hero-saas","eyebrow":"New","heading":"Ship features faster","subheading":"The platform that gets out of your way. Real APIs, predictable pricing, no fake demos.","ctaLabel":"Start free","ctaHref":"/pricing","secondaryLabel":"Read the docs","secondaryHref":"/docs"}`},
		{BlockType: "feature_grid", Data: `{"component":"feature-grid","heading":"What you get","features":[{"title":"Typed SDKs","body":"First-class TypeScript and Go clients with full IntelliSense.","icon":"sdk"},{"title":"Predictable pricing","body":"Per-seat, no per-event surprises.","icon":"price"},{"title":"Open spec","body":"OpenAPI 3.1 schema published. Generate your own client.","icon":"spec"}]}`},
		{BlockType: "feature_grid", Data: `{"component":"pricing-table","heading":"Pricing","tiers":[{"name":"Starter","price":"$0","features":["1 project","Community support"],"ctaLabel":"Start","ctaHref":"/signup"},{"name":"Team","price":"$29 / seat","features":["Unlimited projects","Slack support","SSO"],"ctaLabel":"Start trial","ctaHref":"/signup?plan=team","featured":true},{"name":"Enterprise","price":"Talk to us","features":["SLA","SOC 2 report","Dedicated CSM"],"ctaLabel":"Contact sales","ctaHref":"/contact"}]}`},
		{BlockType: "feature_grid", Data: `{"component":"testimonial-carousel","items":[{"quote":"We replaced three vendors with this in a week.","author":"Sara P.","role":"CTO at Acme"},{"quote":"Pricing finally makes sense.","author":"Jonas K.","role":"Engineering Manager"}]}`},
		{BlockType: "feature_grid", Data: `{"component":"faq-accordion","heading":"Frequently asked","items":[{"q":"Do you have a free tier?","a":"Yes. The Starter tier is free forever for one project."},{"q":"Where is data hosted?","a":"EU regions by default. US optional on request."}]}`},
		{BlockType: "cta", Data: `{"component":"hero-saas","eyebrow":"Ready?","heading":"Start in two minutes","subheading":"No credit card. No demo call required.","ctaLabel":"Get started","ctaHref":"/signup"}`},
	}
	if err := seedBlocks(ctx, q, homeID, homeBlocks); err != nil {
		return err
	}

	// Other pages
	others := []struct {
		title  string
		slug   string
		layout string
		nav    int64
		hero   string
		body   string
	}{
		{"Docs", "/docs", "default", 1, "Documentation", "Quickstart, reference, and guides."},
		{"Pricing", "/pricing", "default", 1, "Pricing", "Simple, predictable pricing."},
		{"About", "/about", "default", 1, "About us", "Why we built this and where we are headed."},
	}
	for i, p := range others {
		pageID := newKitID()
		if err := q.CreatePage(ctx, store.CreatePageParams{
			ID:        pageID,
			SiteID:    siteID,
			Title:     p.title,
			Slug:      p.slug,
			Layout:    p.layout,
			SortOrder: int64(10 + i),
			ShowInNav: p.nav,
		}); err != nil {
			return err
		}
		blocks := []blockDef{
			{BlockType: "hero", Data: fmt.Sprintf(`{"component":"hero-saas","heading":%q,"subheading":%q,"ctaLabel":"Get started","ctaHref":"/signup"}`, p.hero, p.body)},
		}
		if p.slug == "/pricing" {
			blocks = append(blocks, blockDef{BlockType: "feature_grid", Data: `{"component":"pricing-table","tiers":[{"name":"Starter","price":"$0","features":["1 project","Community support"],"ctaLabel":"Start","ctaHref":"/signup"},{"name":"Team","price":"$29 / seat","features":["Unlimited projects","Slack support","SSO"],"ctaLabel":"Start trial","ctaHref":"/signup?plan=team","featured":true},{"name":"Enterprise","price":"Talk to us","features":["SLA","SOC 2 report","Dedicated CSM"],"ctaLabel":"Contact sales","ctaHref":"/contact"}]}`})
		}
		if err := seedBlocks(ctx, q, pageID, blocks); err != nil {
			return err
		}
	}

	return nil
}
