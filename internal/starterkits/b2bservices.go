package starterkits

import (
	"context"
	"fmt"

	"github.com/brightinteraction/atomicsite/internal/store"
)

func init() {
	Default.Register(&b2bServicesKit{})
}

// b2bServicesKit is the general-purpose B2B starter (consultancies, agencies,
// professional services, law firms, accountancies). Replaces the niche
// law-firm kit in the wizard's default list.
type b2bServicesKit struct{}

func (k *b2bServicesKit) ID() string   { return "b2b-services" }
func (k *b2bServicesKit) Name() string { return "B2B Services" }
func (k *b2bServicesKit) Description() string {
	return "Professional services site (consultancies, agencies, law / accounting / advisory). Hero, services grid, case studies, team, contact, FAQ."
}
func (k *b2bServicesKit) TargetSiteTypes() []string { return []string{"b2b"} }

func (k *b2bServicesKit) Apply(ctx context.Context, q *store.Queries, siteID string) error {
	if err := applyBranding(ctx, q, siteID, BrandAudit); err != nil {
		return fmt.Errorf("b2b-services branding: %w", err)
	}
	if err := k.applyComponents(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2b-services components: %w", err)
	}
	if err := k.applyCSS(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2b-services css: %w", err)
	}
	if err := k.applyKnowledgebase(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2b-services kb: %w", err)
	}
	if err := k.applyGuardrails(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2b-services guardrails: %w", err)
	}
	if err := k.applyPages(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2b-services pages: %w", err)
	}
	return nil
}

func (k *b2bServicesKit) applyComponents(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []componentDef{
		{
			Name:     "hero-services",
			Category: "hero",
			Template: `---
const { eyebrow, heading, lede, ctaLabel, ctaHref, secondaryLabel, secondaryHref } = Astro.props;
---
<section class="section-services hero-services">
  <div class="container">
    {eyebrow && <p class="eyebrow">{eyebrow}</p>}
    <h1>{heading}</h1>
    <p class="lede">{lede}</p>
    <p class="actions">
      <a class="btn btn-primary" href={ctaHref}>{ctaLabel}</a>
      {secondaryHref && <a class="btn btn-link" href={secondaryHref}>{secondaryLabel}</a>}
    </p>
  </div>
</section>`,
			Props:      `{"type":"object","required":["heading","lede","ctaLabel","ctaHref"],"properties":{"eyebrow":{"type":"string"},"heading":{"type":"string"},"lede":{"type":"string"},"ctaLabel":{"type":"string"},"ctaHref":{"type":"string"},"secondaryLabel":{"type":"string"},"secondaryHref":{"type":"string"}}}`,
			CSSClasses: `["section-services","hero-services","eyebrow"]`,
			Usage:      "Professional-services hero. Eyebrow + headline + lede + primary/secondary CTA.",
		},
		{
			Name:     "case-study-grid",
			Category: "feature",
			Template: `---
const { heading, items = [] } = Astro.props;
---
<section class="section-services">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <ul class="case-study-grid">
      {items.map((c) => (
        <li class="case-card">
          <p class="industry">{c.industry}</p>
          <h3>{c.title}</h3>
          <p>{c.summary}</p>
          {c.metric && <p class="metric">{c.metric}</p>}
          <a href={c.href}>Read the case study</a>
        </li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"items":{"type":"array","items":{"type":"object","required":["title","summary","href"],"properties":{"industry":{"type":"string"},"title":{"type":"string"},"summary":{"type":"string"},"metric":{"type":"string"},"href":{"type":"string"}}}}}}`,
			CSSClasses: `["case-study-grid","case-card"]`,
			Usage:      "Three-column case study grid. Industry + title + summary + result metric + read-more link.",
		},
		{
			Name:     "trust-strip",
			Category: "feature",
			Template: `---
const { heading, logos = [] } = Astro.props;
---
<section class="trust-strip">
  <div class="container">
    {heading && <p class="trust-label">{heading}</p>}
    <ul class="trust-logos">
      {logos.map((l) => (
        <li><img src={l.src} alt={l.alt} width={l.width || 120} height={l.height || 40} loading="lazy" /></li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"logos":{"type":"array","items":{"type":"object","required":["src","alt"],"properties":{"src":{"type":"string"},"alt":{"type":"string"},"width":{"type":"number"},"height":{"type":"number"}}}}}}`,
			CSSClasses: `["trust-strip","trust-logos","trust-label"]`,
			Usage:      "Logo strip with optional heading. Used for client / partner social proof.",
		},
	}
	return seedComponents(ctx, q, siteID, defs)
}

func (k *b2bServicesKit) applyCSS(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []cssDef{
		{
			Name:     ".section-services",
			Category: "layout",
			CSS:      ".section-services { padding: 5rem 0; background: var(--bg, #ffffff); } @media (min-width: 768px) { .section-services { padding: 7rem 0; } }",
			Usage:    "Standard B2B section spacing.",
		},
		{
			Name:     ".hero-services",
			Category: "hero",
			CSS:      ".hero-services h1 { font-size: clamp(2.25rem, 4vw, 3.75rem); line-height: 1.05; letter-spacing: -0.02em; max-width: 22ch; } .hero-services .lede { font-size: 1.25rem; max-width: 60ch; color: var(--muted, #475569); }",
			Usage:    "Editorial hero typography for service businesses.",
		},
		{
			Name:     ".case-study-grid",
			Category: "feature",
			CSS:      ".case-study-grid { display: grid; gap: 1.5rem; grid-template-columns: 1fr; } @media (min-width: 768px) { .case-study-grid { grid-template-columns: repeat(3, 1fr); } } .case-card { padding: 1.5rem; border: 1px solid var(--border, #e2e8f0); border-radius: 0.75rem; }",
			Usage:    "Case study card grid (1 column mobile, 3 column desktop).",
		},
		{
			Name:     ".trust-strip",
			Category: "feature",
			CSS:      ".trust-strip { padding: 2rem 0; border-block: 1px solid var(--border, #e2e8f0); } .trust-logos { display: flex; flex-wrap: wrap; gap: 2rem; align-items: center; justify-content: center; list-style: none; padding: 0; } .trust-logos img { opacity: 0.7; transition: opacity 0.2s; } .trust-logos img:hover { opacity: 1; }",
			Usage:    "Logo strip with hover affordance.",
		},
	}
	return seedCSS(ctx, q, siteID, defs)
}

func (k *b2bServicesKit) applyKnowledgebase(ctx context.Context, q *store.Queries, siteID string) error {
	entries := []kbDef{
		{Category: "brand", Title: "Voice for B2B services", Content: "Confident, specific, plain. Lead with the problem the buyer recognises (procurement questionnaires, missed deadlines, audit risk). Avoid abstract value words ('synergy', 'leverage', 'best in class'). Use we / us in first person."},
		{Category: "brand", Title: "Proof beats promises", Content: "B2B buyers verify before signing. Every claim should be backed by a metric, named client, or case study link. Testimonials with full name + role + company > anonymous quotes."},
		{Category: "structure", Title: "Page set", Content: "B2B services sites work with: home, services hub + per-service detail pages, case studies, about / team, insights / blog, contact. Keep URLs shallow (max 3 levels). Every page has one primary CTA — usually 'book a discovery call' or 'request an audit'."},
		{Category: "conversion", Title: "Lead magnet pattern", Content: "On every page, offer a tangible artefact in exchange for an email: industry report, audit checklist, ROI calculator. Atomicsite's form blocks integrate with BrightCRM webhook so submissions appear in the CRM with attribution."},
	}
	return seedKB(ctx, q, siteID, entries, 200)
}

func (k *b2bServicesKit) applyGuardrails(ctx context.Context, q *store.Queries, siteID string) error {
	rules := []guardrailDef{
		{RuleType: "allow_block_type", Target: "*", Value: "hero,text,image,cta,feature_grid,contact_form,team_grid,testimonial,faq,pricing", Severity: "error"},
		{RuleType: "forbid_pattern", Target: "block.data.text", Value: "lorem ipsum", Severity: "error"},
		{RuleType: "require_block", Target: "page.layout=landing", Value: "[\"hero\",\"cta\"]", Severity: "warning"},
	}
	return seedGuardrails(ctx, q, siteID, rules)
}

func (k *b2bServicesKit) applyPages(ctx context.Context, q *store.Queries, siteID string) error {
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
		HeroBlock(HeroPayload{
			Eyebrow:        "TRUSTED BY 40+ MID-MARKET TEAMS",
			Headline:       "Compliance work that survives the audit, not the slide deck.",
			Subheading:     "We scope, deliver, and document. SOC 2, GDPR, ISO 27001. Fixed-price projects with measurable outcomes. No retainers built around busy work.",
			CTAText:        "Book a discovery call",
			CTAUrl:         "/contact",
			SecondaryLabel: "How we work",
			SecondaryUrl:   "/process",
			HeroGraphic:    "audit-receipt",
			AuditScore:     "100",
			AuditScoreMax:  "100",
			AuditBaseline:  "62",
			AuditLabel:     "Live audit, this site",
		}),
		LogoStripBlock("Recent and active engagements", []LogoItem{
			{Label: "Nordhealth"}, {Label: "Northbeam"}, {Label: "Lattice"},
			{Label: "Pylon"}, {Label: "Mux"}, {Label: "Korex"},
		}),
		StatGridBlock("What 12 months of work looks like", []StatItem{
			{Value: "43", Label: "audits closed", Context: "SOC 2 Type I + II, ISO 27001, GDPR"},
			{Value: "6 wk", Label: "median time to readiness", Context: "From scope to internal audit"},
			{Value: "0", Label: "renewals lost on a non-conformity", Context: "Across 18 surveillance audits"},
		}),
		{BlockType: "feature_grid", Data: `{"component":"case-study-grid","heading":"Recent work, signed off","items":[{"industry":"FinTech","title":"Closed 11 SOC 2 control gaps in 6 weeks","summary":"Mapped scope, ran the gap analysis, drafted policies, walked the auditor through the evidence.","metric":"6 weeks","href":"/cases/fintech-soc2"},{"industry":"Healthcare","title":"GDPR vendor questionnaires that actually pass","summary":"Built a reusable answer library, audited 30 active vendors, replaced 4 non-compliant ones.","metric":"30 vendors audited","href":"/cases/healthcare-gdpr"},{"industry":"SaaS","title":"Cut compliance overhead by 60 percent","summary":"Replaced manual evidence collection with auto-pulled artefacts from the systems the team already runs.","metric":"60 percent time saved","href":"/cases/saas-automation"}]}`},
		{BlockType: "quote", Data: `{"quote":"They turned a six-month death march into a six-week project. The auditor signed off on the evidence the same day she opened the room.","attribution":"Director of Engineering, Series B FinTech"}`},
		AccordionFAQBlock("Questions we answer before the discovery call", []FAQItem{
			{Question: "What does a fixed-price project actually mean?", Answer: "We write a scope before you sign. It names the standard, the controls in scope, the deliverables, the timeline, and the exit criteria. The fee is the fee. Out-of-scope work needs a written change order. No hourly invoices, no 'we'll see what we find'."},
			{Question: "Do you bring the auditor or do we?", Answer: "You bring the auditor. We do not sell readiness AND audit; that's a conflict of interest most boutiques pretend doesn't exist. We've worked with Schellman, Prescient, A-LIGN, and the four big firms. We pre-meet the auditor with you so the engagement starts on the same evidence shape."},
			{Question: "Will the policies be ours or templated?", Answer: "Yours. We start from a clean template, then walk every paragraph against how your team actually operates. Generic policies fail surveillance audits because the evidence doesn't match. We refuse to ship policies you can't enforce."},
			{Question: "How are you different from a Vanta or Drata?", Answer: "We're the humans those tools tell you to hire. Vanta automates evidence collection; somebody still has to scope, draft, run the gap analysis, and prep the auditor. We do that part. We work alongside Vanta and Drata regularly."},
			{Question: "Can you sign an NDA before the discovery call?", Answer: "Yes. We countersign within one business day. We're the data-controllers for nothing on your side and never receive production data. The discovery call is scope-of-work only."},
		}),
		{BlockType: "cta", Data: `{"heading":"Ready to scope a project?","text":"30 minutes, no slides, no pitch. We listen, ask the questions that scope the work, and tell you whether we can help.","cta_text":"Book the call","cta_url":"/contact","variant":"primary"}`},
	}
	if err := seedBlocks(ctx, q, homeID, homeBlocks); err != nil {
		return err
	}

	// Services hub
	hubID := newKitID()
	if err := q.CreatePage(ctx, store.CreatePageParams{
		ID:        hubID,
		SiteID:    siteID,
		Title:     "Services",
		Slug:      "/services",
		Layout:    "default",
		SortOrder: 10,
		ShowInNav: 1,
	}); err != nil {
		return err
	}
	if err := seedBlocks(ctx, q, hubID, []blockDef{
		{BlockType: "hero", Data: `{"component":"hero-services","heading":"Services","lede":"Fixed-scope projects with measurable outcomes. No retainers built around busy work."}`},
	}); err != nil {
		return err
	}

	return nil
}
