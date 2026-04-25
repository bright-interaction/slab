package starterkits

import (
	"context"
	"fmt"

	"github.com/bright-interaction/slab/internal/store"
)

func init() {
	Default.Register(&portfolioKit{})
}

type portfolioKit struct{}

func (k *portfolioKit) ID() string          { return "portfolio" }
func (k *portfolioKit) Name() string        { return "Portfolio (One-Pager)" }
func (k *portfolioKit) Description() string {
	return "Personal portfolio one-pager. Hero, project grid, about block, contact."
}
func (k *portfolioKit) TargetSiteTypes() []string { return []string{"personal", "one-pager"} }

func (k *portfolioKit) Apply(ctx context.Context, q *store.Queries, siteID string) error {
	if err := k.applyComponents(ctx, q, siteID); err != nil {
		return fmt.Errorf("portfolio components: %w", err)
	}
	if err := k.applyCSS(ctx, q, siteID); err != nil {
		return fmt.Errorf("portfolio css: %w", err)
	}
	if err := k.applyKnowledgebase(ctx, q, siteID); err != nil {
		return fmt.Errorf("portfolio kb: %w", err)
	}
	if err := k.applyGuardrails(ctx, q, siteID); err != nil {
		return fmt.Errorf("portfolio guardrails: %w", err)
	}
	if err := k.applyPages(ctx, q, siteID); err != nil {
		return fmt.Errorf("portfolio pages: %w", err)
	}
	return nil
}

func (k *portfolioKit) applyComponents(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []componentDef{
		{
			Name:     "hero-portrait",
			Category: "hero",
			Template: `---
const { name, role, bio, avatar } = Astro.props;
---
<section class="hero-portrait scroll-snap">
  <div class="container">
    {avatar && <img src={avatar} alt={name} class="avatar" />}
    <h1>{name}</h1>
    <p class="role">{role}</p>
    <p class="bio">{bio}</p>
  </div>
</section>`,
			Props:      `{"type":"object","required":["name","role"],"properties":{"name":{"type":"string"},"role":{"type":"string"},"bio":{"type":"string"},"avatar":{"type":"string"}}}`,
			CSSClasses: `["hero-portrait","scroll-snap"]`,
			Usage:      "Centered portrait hero with avatar, name, role and one-line bio.",
		},
		{
			Name:     "project-grid",
			Category: "feature",
			Template: `---
const { heading, projects = [] } = Astro.props;
---
<section class="projects scroll-snap">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <ul class="project-grid">
      {projects.map((p) => (
        <li class="project-card">
          {p.image && <img src={p.image} alt={p.title} />}
          <h3>{p.title}</h3>
          <p class="role">{p.role}</p>
          <a href={p.href}>View case</a>
        </li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"projects":{"type":"array","items":{"type":"object","required":["title","href"],"properties":{"title":{"type":"string"},"role":{"type":"string"},"href":{"type":"string"},"image":{"type":"string"}}}}}}`,
			CSSClasses: `["project-grid","project-card"]`,
			Usage:      "Project grid with image cards, title, role, link.",
		},
		{
			Name:     "about-block",
			Category: "about",
			Template: `---
const { heading, body, skills = [] } = Astro.props;
---
<section class="about scroll-snap">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <div class="body">{body}</div>
    {skills.length > 0 && (
      <ul class="skills">
        {skills.map((s) => <li>{s}</li>)}
      </ul>
    )}
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"body":{"type":"string"},"skills":{"type":"array","items":{"type":"string"}}}}`,
			CSSClasses: `["about"]`,
			Usage:      "About block with long text and skills list.",
		},
		{
			Name:     "contact-block",
			Category: "contact",
			Template: `---
const { email, socials = [] } = Astro.props;
---
<section class="contact scroll-snap">
  <div class="container">
    <h2>Get in touch</h2>
    {email && <p><a href={"mailto:" + email}>{email}</a></p>}
    <ul class="socials">
      {socials.map((s) => (
        <li><a href={s.href}>{s.label}</a></li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"email":{"type":"string"},"socials":{"type":"array","items":{"type":"object","required":["label","href"],"properties":{"label":{"type":"string"},"href":{"type":"string"}}}}}}`,
			CSSClasses: `["contact"]`,
			Usage:      "Contact block with email link and social links.",
		},
	}
	return seedComponents(ctx, q, siteID, defs)
}

func (k *portfolioKit) applyCSS(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []cssDef{
		{
			Name:     ".scroll-snap",
			Category: "layout",
			CSS:      ".scroll-snap { scroll-snap-align: start; min-height: 100vh; display: flex; align-items: center; }",
			Usage:    "One-pager scroll-snap section.",
		},
		{
			Name:     ".project-card",
			Category: "card",
			CSS:      ".project-card { display: flex; flex-direction: column; gap: 0.5rem; border-radius: 0.75rem; overflow: hidden; background: var(--bg-soft, #f8fafc); padding: 1rem; transition: transform 0.2s ease; } .project-card:hover { transform: translateY(-2px); } .project-card img { aspect-ratio: 16/9; object-fit: cover; border-radius: 0.5rem; }",
			Usage:    "Portfolio project card with image.",
		},
	}
	return seedCSS(ctx, q, siteID, defs)
}

func (k *portfolioKit) applyKnowledgebase(ctx context.Context, q *store.Queries, siteID string) error {
	entries := []kbDef{
		{Category: "brand", Title: "Brand voice", Content: "Personal, conversational, work-focused. Write in first person. Show the work, talk about the role you played, what shipped and what you learned. Avoid abstract claims."},
		{Category: "brand", Title: "What goes on the page", Content: "Hero with name and role, three to six recent projects, a short about section with skills, and a contact block. Nothing else. The page is a one-pager."},
	}
	return seedKB(ctx, q, siteID, entries, 100)
}

func (k *portfolioKit) applyGuardrails(ctx context.Context, q *store.Queries, siteID string) error {
	rules := []guardrailDef{
		{RuleType: "allow_block_type", Target: "*", Value: "hero,text,image,cta,feature_grid", Severity: "error"},
		{RuleType: "forbid_pattern", Target: "block.data.text", Value: "lorem ipsum", Severity: "error"},
	}
	return seedGuardrails(ctx, q, siteID, rules)
}

func (k *portfolioKit) applyPages(ctx context.Context, q *store.Queries, siteID string) error {
	homeID := newKitID()
	if err := q.CreatePage(ctx, store.CreatePageParams{
		ID:        homeID,
		SiteID:    siteID,
		Title:     "Home",
		Slug:      "/",
		Layout:    "one-pager",
		SortOrder: 0,
		ShowInNav: 1,
	}); err != nil {
		return err
	}
	blocks := []blockDef{
		{BlockType: "hero", Data: `{"component":"hero-portrait","name":"Your Name","role":"Designer and Developer","bio":"I build calm, useful things on the web.","avatar":""}`},
		{BlockType: "feature_grid", Data: `{"component":"project-grid","heading":"Selected work","projects":[{"title":"Project One","role":"Lead designer","href":"#","image":""},{"title":"Project Two","role":"Frontend engineer","href":"#","image":""},{"title":"Project Three","role":"Independent","href":"#","image":""}]}`},
		{BlockType: "text", Data: `{"component":"about-block","heading":"About","body":"I have spent the last decade shipping product across consumer and B2B. I like clear typography, fast pages and shipping things people actually use.","skills":["Product design","Frontend engineering","Design systems","Accessibility"]}`},
		{BlockType: "cta", Data: `{"component":"contact-block","email":"hello@example.com","socials":[{"label":"GitHub","href":"https://github.com/"},{"label":"LinkedIn","href":"https://linkedin.com/"}]}`},
	}
	return seedBlocks(ctx, q, homeID, blocks)
}
