package starterkits

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/bright-interaction/slab/internal/store"
)

func init() {
	Default.Register(&lawFirmKit{})
}

type lawFirmKit struct{}

func (k *lawFirmKit) ID() string          { return "law-firm" }
func (k *lawFirmKit) Name() string        { return "Law Firm (Swedish)" }
func (k *lawFirmKit) Description() string {
	return "Formal Swedish business site for advokatbyråer. GDPR-confident, services silo, contact-form first."
}
func (k *lawFirmKit) TargetSiteTypes() []string { return []string{"b2b"} }

func (k *lawFirmKit) Apply(ctx context.Context, q *store.Queries, siteID string) error {
	if err := k.applyComponents(ctx, q, siteID); err != nil {
		return fmt.Errorf("law-firm components: %w", err)
	}
	if err := k.applyCSS(ctx, q, siteID); err != nil {
		return fmt.Errorf("law-firm css: %w", err)
	}
	if err := k.applyKnowledgebase(ctx, q, siteID); err != nil {
		return fmt.Errorf("law-firm kb: %w", err)
	}
	if err := k.applyGuardrails(ctx, q, siteID); err != nil {
		return fmt.Errorf("law-firm guardrails: %w", err)
	}
	if err := k.applyPages(ctx, q, siteID); err != nil {
		return fmt.Errorf("law-firm pages: %w", err)
	}
	return nil
}

func (k *lawFirmKit) applyComponents(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []componentDef{
		{
			Name:     "hero-formal",
			Category: "hero",
			Template: `---
const { heading, lede, ctaLabel, ctaHref, secondaryLabel, secondaryHref } = Astro.props;
---
<section class="section-formal hero-formal">
  <div class="container">
    <h1>{heading}</h1>
    <p class="lede">{lede}</p>
    <p class="actions">
      <a class="btn btn-primary" href={ctaHref}>{ctaLabel}</a>
      {secondaryHref && <a class="btn btn-link" href={secondaryHref}>{secondaryLabel}</a>}
    </p>
  </div>
</section>`,
			Props:      `{"type":"object","required":["heading","lede","ctaLabel","ctaHref"],"properties":{"heading":{"type":"string"},"lede":{"type":"string"},"ctaLabel":{"type":"string"},"ctaHref":{"type":"string"},"secondaryLabel":{"type":"string"},"secondaryHref":{"type":"string"}}}`,
			CSSClasses: `["section-formal","hero-formal"]`,
			Usage:      "Formal hero with large headline, lede paragraph, primary CTA and optional secondary link.",
		},
		{
			Name:     "services-grid",
			Category: "feature",
			Template: `---
const { heading, services = [] } = Astro.props;
---
<section class="section-formal">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <ul class="services-grid">
      {services.map((s) => (
        <li class="service-card">
          <h3>{s.title}</h3>
          <p>{s.body}</p>
          <a href={s.href}>Läs mer</a>
        </li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"services":{"type":"array","items":{"type":"object","required":["title","body","href"],"properties":{"title":{"type":"string"},"body":{"type":"string"},"href":{"type":"string"}}}}}}`,
			CSSClasses: `["section-formal","services-grid"]`,
			Usage:      "Three-column services grid. Each item has title, body and link.",
		},
		{
			Name:     "contact-callout",
			Category: "cta",
			Template: `---
const { heading, gdprNotice, formEmbedUrl } = Astro.props;
---
<section class="section-formal callout-svar">
  <div class="container">
    <h2>{heading}</h2>
    {formEmbedUrl && <iframe src={formEmbedUrl} title="Kontaktformulär" loading="lazy" />}
    <p class="gdpr-notice">{gdprNotice}</p>
  </div>
</section>`,
			Props:      `{"type":"object","required":["heading","gdprNotice"],"properties":{"heading":{"type":"string"},"gdprNotice":{"type":"string"},"formEmbedUrl":{"type":"string"}}}`,
			CSSClasses: `["section-formal","callout-svar"]`,
			Usage:      "Contact callout with heading, embedded form and GDPR notice.",
		},
		{
			Name:     "team-grid",
			Category: "feature",
			Template: `---
const { heading, members = [] } = Astro.props;
---
<section class="section-formal">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <ul class="team-grid">
      {members.map((m) => (
        <li class="team-card">
          {m.avatar && <img src={m.avatar} alt={m.name} />}
          <h3>{m.name}</h3>
          <p class="role">{m.role}</p>
          <p class="bio">{m.bio}</p>
        </li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"members":{"type":"array","items":{"type":"object","required":["name","role"],"properties":{"name":{"type":"string"},"role":{"type":"string"},"bio":{"type":"string"},"avatar":{"type":"string"}}}}}}`,
			CSSClasses: `["team-grid","team-card"]`,
			Usage:      "Team member cards with avatar, name, role and bio.",
		},
	}
	return seedComponents(ctx, q, siteID, defs)
}

func (k *lawFirmKit) applyCSS(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []cssDef{
		{
			Name:     ".section-formal",
			Category: "layout",
			CSS:      ".section-formal { padding: 6rem 0; background: var(--bg, #ffffff); }",
			Usage:    "Standard formal section spacing for law firm pages.",
		},
		{
			Name:     ".hero-formal",
			Category: "hero",
			CSS:      ".hero-formal h1 { font-size: clamp(2.25rem, 4vw, 3.75rem); line-height: 1.1; letter-spacing: -0.01em; max-width: 36ch; } .hero-formal .lede { font-size: 1.25rem; max-width: 60ch; color: var(--muted, #475569); }",
			Usage:    "Typographic hero with large headline and supporting lede.",
		},
		{
			Name:     ".callout-svar",
			Category: "callout",
			CSS:      ".callout-svar { border-left: 3px solid var(--accent, #1e3a8a); padding-left: 1.5rem; }",
			Usage:    "Subtle accent border for GDPR-related callouts.",
		},
		{
			Name:     ".team-card",
			Category: "card",
			CSS:      ".team-card { display: flex; flex-direction: column; align-items: center; text-align: center; gap: 0.5rem; } .team-card img { width: 96px; height: 96px; border-radius: 50%; object-fit: cover; }",
			Usage:    "Centered avatar, name, role for team member cards.",
		},
	}
	return seedCSS(ctx, q, siteID, defs)
}

func (k *lawFirmKit) applyKnowledgebase(ctx context.Context, q *store.Queries, siteID string) error {
	entries := []kbDef{
		{Category: "brand", Title: "Brand voice", Content: "Formal Swedish, third-person, GDPR-confident. Never marketing-jargon. Use 'vi' to refer to the firm. Avoid superlatives and vague claims."},
		{Category: "brand", Title: "Tonality do/don't list", Content: "Do: 'vi', 'rådgivning', 'erfarenhet', 'hantera', 'följa GDPR'. Don't: 'kanonisk', 'best in class', 'AI-driven', 'leverage', 'synergy'."},
		{Category: "brand", Title: "Target audience", Content: "Small and mid-sized Swedish law firms (advokatbyråer) worried about GDPR vendor questionnaires, security incidents, and client data handling."},
		{Category: "technical", Title: "Kontaktuppgifter on every page", Content: "Every page must surface kontaktuppgifter (phone or email and reception address) either inline or via the footer global block."},
		{Category: "technical", Title: "Article and news templates", Content: "Articles use the 'analyse' page layout (long-form, with author and updated date). News items use the 'news' layout (short, dated, no byline)."},
	}
	return seedKB(ctx, q, siteID, entries, 100)
}

func (k *lawFirmKit) applyGuardrails(ctx context.Context, q *store.Queries, siteID string) error {
	rules := []guardrailDef{
		{RuleType: "allow_block_type", Target: "*", Value: "hero,text,image,cta,feature_grid,contact_form,team_grid", Severity: "error"},
		{RuleType: "forbid_pattern", Target: "block.data.text", Value: "lorem ipsum", Severity: "error"},
		{RuleType: "require_block", Target: "page.layout=landing", Value: "contact_form", Severity: "error"},
	}
	return seedGuardrails(ctx, q, siteID, rules)
}

func (k *lawFirmKit) applyPages(ctx context.Context, q *store.Queries, siteID string) error {
	// Home page
	homeID := newKitID()
	if err := q.CreatePage(ctx, store.CreatePageParams{
		ID:        homeID,
		SiteID:    siteID,
		Title:     "Hem",
		Slug:      "/",
		Layout:    "landing",
		SortOrder: 0,
		ShowInNav: 1,
	}); err != nil {
		return err
	}
	homeBlocks := []blockDef{
		{BlockType: "hero", Data: `{"component":"hero-formal","heading":"Juridisk rådgivning du kan lita på","lede":"Vi hjälper svenska företag att navigera GDPR, avtal och tvister med klarhet och erfarenhet.","ctaLabel":"Kontakta oss","ctaHref":"#kontakt","secondaryLabel":"Våra tjänster","secondaryHref":"/tjanster"}`},
		{BlockType: "text", Data: `{"text":"Vår byrå har lång erfarenhet av att stödja små och medelstora företag i Sverige. Vi kombinerar djup juridisk kompetens med praktisk affärsförståelse. När ni anlitar oss får ni en partner som tar ansvar genom hela processen, från första rådgivning till genomförande."}`},
		{BlockType: "feature_grid", Data: `{"component":"services-grid","heading":"Tjänster","services":[{"title":"GDPR-rådgivning","body":"Vi guidar er genom dataskyddsförordningens krav.","href":"/tjanster/gdpr-radgivning"},{"title":"Avtal","body":"Vi granskar och utformar avtal som håller.","href":"/tjanster/avtal"},{"title":"Tvistelösning","body":"Vi företräder er i förhandlingar och processer.","href":"/tjanster"}]}`},
		{BlockType: "feature_grid", Data: `{"component":"team-grid","heading":"Teamet","members":[{"name":"Anna Lindqvist","role":"Senior advokat","bio":"Specialist på GDPR och dataskydd."},{"name":"Erik Bergström","role":"Advokat","bio":"Avtalsrätt och bolagsrätt."},{"name":"Sofia Karlsson","role":"Biträdande jurist","bio":"Tvistelösning och processrätt."},{"name":"Johan Nilsson","role":"Advokat","bio":"Arbetsrätt och anställningsavtal."}]}`},
		{BlockType: "contact_form", Data: `{"component":"contact-callout","heading":"Kontakta oss","gdprNotice":"Vi behandlar dina personuppgifter enligt vår integritetspolicy. Inga uppgifter delas med tredje part utan ditt samtycke.","formEmbedUrl":""}`},
	}
	if err := seedBlocks(ctx, q, homeID, homeBlocks); err != nil {
		return err
	}

	// Services hub
	hubID := newKitID()
	if err := q.CreatePage(ctx, store.CreatePageParams{
		ID:        hubID,
		SiteID:    siteID,
		Title:     "Tjänster",
		Slug:      "/tjanster",
		Layout:    "default",
		SortOrder: 10,
		ShowInNav: 1,
	}); err != nil {
		return err
	}
	hubBlocks := []blockDef{
		{BlockType: "hero", Data: `{"component":"hero-formal","heading":"Våra tjänster","lede":"Vi erbjuder rådgivning inom GDPR, avtal och tvistelösning.","ctaLabel":"Kontakta oss","ctaHref":"/#kontakt"}`},
		{BlockType: "text", Data: `{"text":"Varje uppdrag börjar med ett samtal. Vi lyssnar på er situation, kartlägger riskerna och föreslår en konkret plan. Inga oklara timmar, inga överraskningar i fakturan."}`},
		{BlockType: "feature_grid", Data: `{"component":"services-grid","heading":"Områden","services":[{"title":"GDPR-rådgivning","body":"Dataskydd, DPIA, registerförteckning, leverantörsavtal.","href":"/tjanster/gdpr-radgivning"},{"title":"Avtal","body":"Granskning och utformning av kommersiella avtal.","href":"/tjanster/avtal"}]}`},
	}
	if err := seedBlocks(ctx, q, hubID, hubBlocks); err != nil {
		return err
	}

	// Service detail pages (empty placeholders)
	details := []struct {
		title string
		slug  string
	}{
		{"GDPR-rådgivning", "/tjanster/gdpr-radgivning"},
		{"Avtal", "/tjanster/avtal"},
	}
	for i, d := range details {
		pageID := newKitID()
		if err := q.CreatePage(ctx, store.CreatePageParams{
			ID:        pageID,
			SiteID:    siteID,
			Title:     d.title,
			Slug:      d.slug,
			Layout:    "default",
			SortOrder: int64(20 + i),
			ShowInNav: 0,
		}); err != nil {
			return err
		}
		if err := seedBlocks(ctx, q, pageID, []blockDef{
			{BlockType: "hero", Data: fmt.Sprintf(`{"component":"hero-formal","heading":%q,"lede":"Beskrivning kommer snart.","ctaLabel":"Kontakta oss","ctaHref":"/#kontakt"}`, d.title)},
		}); err != nil {
			return err
		}
	}

	return nil
}

// --- shared kit helpers ---

type componentDef struct {
	Name       string
	Category   string
	Template   string
	Props      string
	CSSClasses string
	Usage      string
}

type cssDef struct {
	Name     string
	Category string
	CSS      string
	Usage    string
}

type kbDef struct {
	Category string
	Title    string
	Content  string
}

type guardrailDef struct {
	RuleType string
	Target   string
	Value    string
	Severity string
}

type blockDef struct {
	BlockType string
	Data      string
}

func seedComponents(ctx context.Context, q *store.Queries, siteID string, defs []componentDef) error {
	for _, d := range defs {
		if _, err := q.GetComponentByName(ctx, store.GetComponentByNameParams{SiteID: siteID, Name: d.Name}); err == nil {
			continue
		}
		if err := q.CreateComponent(ctx, store.CreateComponentParams{
			ID:          newKitID(),
			SiteID:      siteID,
			Name:        d.Name,
			Category:    d.Category,
			Template:    d.Template,
			PropsSchema: d.Props,
			CssClasses:  d.CSSClasses,
			UsageNote:   d.Usage,
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedCSS(ctx context.Context, q *store.Queries, siteID string, defs []cssDef) error {
	for i, d := range defs {
		if _, err := q.GetCSSClassByName(ctx, store.GetCSSClassByNameParams{SiteID: siteID, Name: d.Name}); err == nil {
			continue
		}
		if err := q.CreateCSSClass(ctx, store.CreateCSSClassParams{
			ID:        newKitID(),
			SiteID:    siteID,
			Name:      d.Name,
			Category:  d.Category,
			Css:       d.CSS,
			UsageNote: d.Usage,
			SortOrder: int64(i),
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedKB(ctx context.Context, q *store.Queries, siteID string, entries []kbDef, baseSort int) error {
	for i, e := range entries {
		if err := q.CreateKnowledgebaseEntry(ctx, store.CreateKnowledgebaseEntryParams{
			ID:        newKitID(),
			SiteID:    siteID,
			Category:  e.Category,
			Title:     e.Title,
			Content:   e.Content,
			SortOrder: int64(baseSort + i),
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedGuardrails(ctx context.Context, q *store.Queries, siteID string, rules []guardrailDef) error {
	for _, r := range rules {
		if err := q.CreateGuardrail(ctx, store.CreateGuardrailParams{
			ID:       newKitID(),
			SiteID:   siteID,
			RuleType: r.RuleType,
			Target:   r.Target,
			Value:    r.Value,
			Severity: r.Severity,
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedBlocks(ctx context.Context, q *store.Queries, pageID string, blocks []blockDef) error {
	for i, b := range blocks {
		if err := q.CreateBlock(ctx, store.CreateBlockParams{
			ID:        newKitID(),
			PageID:    pageID,
			BlockType: b.BlockType,
			SortOrder: int64(i),
			DataJson:  b.Data,
			StyleJson: "{}",
			IsVisible: 1,
		}); err != nil {
			return err
		}
	}
	return nil
}

func newKitID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
