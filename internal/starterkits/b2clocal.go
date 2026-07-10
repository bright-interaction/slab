package starterkits

import (
	"context"
	"fmt"

	"github.com/bright-interaction/slab/internal/store"
)

func init() {
	Default.Register(&b2cLocalKit{})
}

// b2cLocalKit is for B2C local businesses: restaurants, salons, dentists,
// trades. Optimises for local SEO (LocalBusiness schema with hours + geo)
// and quick "book / call / find" actions above the fold.
type b2cLocalKit struct{}

func (k *b2cLocalKit) ID() string   { return "b2c-local" }
func (k *b2cLocalKit) Name() string { return "B2C Local Business" }
func (k *b2cLocalKit) Description() string {
	return "Local-business site (restaurants, salons, trades, clinics). Hours, location, booking, reviews. LocalBusiness schema with geo + opening hours."
}
func (k *b2cLocalKit) TargetSiteTypes() []string { return []string{"b2c"} }

func (k *b2cLocalKit) Apply(ctx context.Context, q *store.Queries, siteID string) error {
	if err := applyBranding(ctx, q, siteID, BrandPulse); err != nil {
		return fmt.Errorf("b2c-local branding: %w", err)
	}
	if err := k.applyComponents(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2c-local components: %w", err)
	}
	if err := k.applyCSS(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2c-local css: %w", err)
	}
	if err := k.applyKnowledgebase(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2c-local kb: %w", err)
	}
	if err := k.applyGuardrails(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2c-local guardrails: %w", err)
	}
	if err := k.applyPages(ctx, q, siteID); err != nil {
		return fmt.Errorf("b2c-local pages: %w", err)
	}
	return nil
}

func (k *b2cLocalKit) applyComponents(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []componentDef{
		{
			Name:     "hero-local",
			Category: "hero",
			Template: `---
const { heading, lede, primaryLabel, primaryHref, phone, address } = Astro.props;
---
<section class="hero-local">
  <div class="container">
    <h1>{heading}</h1>
    <p class="lede">{lede}</p>
    <p class="hero-actions">
      <a class="btn btn-primary" href={primaryHref}>{primaryLabel}</a>
      {phone && <a class="btn btn-link" href={` + "`tel:${phone}`" + `}>Call {phone}</a>}
    </p>
    {address && <p class="hero-address">{address}</p>}
  </div>
</section>`,
			Props:      `{"type":"object","required":["heading","lede","primaryLabel","primaryHref"],"properties":{"heading":{"type":"string"},"lede":{"type":"string"},"primaryLabel":{"type":"string"},"primaryHref":{"type":"string"},"phone":{"type":"string"},"address":{"type":"string"}}}`,
			CSSClasses: `["hero-local","hero-actions","hero-address"]`,
			Usage:      "Local-business hero with primary action (book / order / visit) plus tap-to-call and address.",
		},
		{
			Name:     "hours-card",
			Category: "feature",
			Template: `---
const { heading, days = [] } = Astro.props;
---
<section class="hours-card">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <dl class="hours-list">
      {days.map((d) => (
        <Fragment>
          <dt>{d.day}</dt>
          <dd>{d.hours}</dd>
        </Fragment>
      ))}
    </dl>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"days":{"type":"array","items":{"type":"object","required":["day","hours"],"properties":{"day":{"type":"string"},"hours":{"type":"string"}}}}}}`,
			CSSClasses: `["hours-card","hours-list"]`,
			Usage:      "Opening hours definition list. Renders as <dl> for accessibility + LocalBusiness openingHoursSpecification schema.",
		},
		{
			Name:     "review-grid",
			Category: "feature",
			Template: `---
const { heading, reviews = [] } = Astro.props;
---
<section class="review-grid-section">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <ul class="review-grid">
      {reviews.map((r) => (
        <li class="review-card">
          <p class="rating" aria-label={` + "`${r.rating} out of 5 stars`" + `}>{'★'.repeat(r.rating)}{'☆'.repeat(5 - r.rating)}</p>
          <blockquote>{r.quote}</blockquote>
          <p class="reviewer">{r.author}</p>
        </li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"reviews":{"type":"array","items":{"type":"object","required":["author","quote","rating"],"properties":{"author":{"type":"string"},"quote":{"type":"string"},"rating":{"type":"number"}}}}}}`,
			CSSClasses: `["review-grid-section","review-grid","review-card","rating","reviewer"]`,
			Usage:      "Customer review cards with star rating and accessible aria-label. Pulls AggregateRating into schema when wired.",
		},
	}
	return seedComponents(ctx, q, siteID, defs)
}

func (k *b2cLocalKit) applyCSS(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []cssDef{
		{
			Name:     ".hero-local",
			Category: "hero",
			CSS:      ".hero-local { padding: 5rem 0 4rem; text-align: center; } .hero-local h1 { font-size: clamp(2rem, 5vw, 3.5rem); margin-bottom: 1rem; } .hero-local .lede { font-size: 1.125rem; max-width: 50ch; margin-inline: auto; color: var(--muted, #475569); } .hero-actions { display: flex; flex-wrap: wrap; gap: 0.75rem; justify-content: center; margin-top: 1.5rem; } .hero-address { margin-top: 1rem; font-size: 0.95rem; color: var(--muted); }",
			Usage:    "Centered hero suited to small local pages.",
		},
		{
			Name:     ".hours-list",
			Category: "feature",
			CSS:      ".hours-list { display: grid; grid-template-columns: max-content 1fr; gap: 0.5rem 2rem; max-width: 30rem; } .hours-list dt { font-weight: 600; } .hours-list dd { margin: 0; }",
			Usage:    "Two-column day/hours dl.",
		},
		{
			Name:     ".review-grid",
			Category: "feature",
			CSS:      ".review-grid { display: grid; gap: 1rem; grid-template-columns: 1fr; list-style: none; padding: 0; } @media (min-width: 768px) { .review-grid { grid-template-columns: repeat(3, 1fr); } } .review-card { padding: 1.5rem; background: var(--bg-elevated, #f8fafc); border-radius: 0.75rem; } .review-card blockquote { margin: 0.5rem 0; font-style: italic; } .review-card .rating { color: #f59e0b; font-size: 1rem; }",
			Usage:    "Three-column review cards with star rating colour.",
		},
	}
	return seedCSS(ctx, q, siteID, defs)
}

func (k *b2cLocalKit) applyKnowledgebase(ctx context.Context, q *store.Queries, siteID string) error {
	entries := []kbDef{
		{Category: "schema", Title: "LocalBusiness schema essentials", Content: "B2C local sites need LocalBusiness JSON-LD with: name, address (PostalAddress), telephone, geo (latitude, longitude), openingHoursSpecification (per-day), priceRange ('$', '$$', '$$$'), and aggregateRating when reviews are public. Slab emits this when site_profile.address + new settings local.opening_hours and local.geo are filled."},
		{Category: "local-seo", Title: "Google Business Profile parity", Content: "Site NAP (Name, Address, Phone) must EXACTLY match Google Business Profile. Mismatches sink local rankings. Same applies to Apple Business Connect, Bing Places. Slab's site profile is the single source: copy it verbatim into other directories."},
		{Category: "conversion", Title: "Above-the-fold actions", Content: "Local visitors want one of: book a table / appointment, call, see hours, see directions. Hero must surface these in <2 taps. Tap-to-call (tel: links) and tap-to-map (maps: or geo: links) on mobile."},
		{Category: "content", Title: "Reviews + UGC", Content: "Pull verified reviews from Google / Tripadvisor / Trustpilot via API where allowed. Display 3-6 on home, full grid on /reviews. Always include reviewer name (first name + initial OK), date, and star rating in schema. Never fabricate reviews: Google penalises and many jurisdictions criminalise."},
	}
	return seedKB(ctx, q, siteID, entries, 300)
}

func (k *b2cLocalKit) applyGuardrails(ctx context.Context, q *store.Queries, siteID string) error {
	rules := []guardrailDef{
		{RuleType: "allow_block_type", Target: "*", Value: "hero,text,image,cta,feature_grid,gallery,review,hours,faq,contact_form", Severity: "error"},
		{RuleType: "forbid_pattern", Target: "block.data.text", Value: "lorem ipsum", Severity: "error"},
		{RuleType: "require_block", Target: "page.layout=landing", Value: "[\"hero\"]", Severity: "warning"},
	}
	return seedGuardrails(ctx, q, siteID, rules)
}

func (k *b2cLocalKit) applyPages(ctx context.Context, q *store.Queries, siteID string) error {
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
			Eyebrow:        "STORGATAN 1 - STOCKHOLM",
			Headline:       "Booked in 30 seconds. In and out before lunch.",
			Subheading:     "Walk-in friendly, online booking that actually works, fair prices on the menu. Open six days a week and a real human picks up when you call.",
			CTAText:        "Book online",
			CTAUrl:         "/book",
			SecondaryLabel: "Call +46 76 297 80 35",
			SecondaryUrl:   "tel:+46762978035",
			HeroGraphic:    "pulse",
		}),
		LogoStripBlock("Featured in", []LogoItem{
			{Label: "DN Stockholm"}, {Label: "Aftonbladet"}, {Label: "City Guide"},
			{Label: "Time Out"}, {Label: "Visit Stockholm"}, {Label: "The Local"},
		}),
		StatGridBlock("Numbers from the last year", []StatItem{
			{Value: "4.8", Label: "Google rating", Context: "1,247 verified reviews"},
			{Value: "12 min", Label: "median walk-in wait", Context: "Excluding peak Friday evening"},
			{Value: "11", Label: "years on this street", Context: "Same family, same recipe"},
		}),
		{BlockType: "feature_grid", Data: `{"component":"hours-card","heading":"Opening hours","days":[{"day":"Mon-Fri","hours":"08:00 - 17:00"},{"day":"Saturday","hours":"10:00 - 16:00"},{"day":"Sunday","hours":"Closed"}]}`},
		{BlockType: "feature_grid", Data: `{"component":"review-grid","heading":"Recent reviews","reviews":[{"author":"Anna L.","quote":"Booked online, in and out in 30 minutes. Staff remembered my name on the second visit.","rating":5},{"author":"Erik B.","quote":"Best in town for what they do, and the only place I trust on Storgatan for the price.","rating":5},{"author":"Sofia K.","quote":"Friendly staff, fair prices, and the booking confirmation actually arrived. Will be back.","rating":4}]}`},
		AccordionFAQBlock("Quick answers", []FAQItem{
			{Question: "Do you take walk-ins or do I need to book?", Answer: "Both. Booking online is the safer bet, especially between 12:00 and 14:00 on weekdays. Walk-ins are welcome and the median wait is around 12 minutes; we'll text you when your table is ready so you can grab a coffee next door."},
			{Question: "Do you have gluten-free or vegan options?", Answer: "Yes. Three vegan dishes and at least one gluten-free option on the lunch menu, more in the evening. Tell us about allergies when you book and we'll flag the kitchen. Cross-contamination notes are on the printed menu."},
			{Question: "Where can I park?", Answer: "Street parking on Storgatan and Drottninggatan (paid until 18:00 on weekdays). The closest garage is Q-Park Klara, two minutes walk. Bike racks are right outside the door."},
			{Question: "Can you cater a private event?", Answer: "Yes, for groups of 12 to 40. We close the back room for private events Tuesday to Thursday evenings. Email events@example.com with a date and headcount and we'll send a fixed-price menu within one business day."},
			{Question: "Are kids welcome?", Answer: "Always. High chairs available, kids menu on request, changing table in the accessible toilet. We pace kid-friendly meals quickly so the small ones don't lose patience."},
		}),
		{BlockType: "cta", Data: `{"heading":"Ready to visit?","text":"Tap below to book a slot, or just walk in. The kitchen closes 45 minutes before listed hours so the team can clean up properly.","cta_text":"Book online","cta_url":"/book","variant":"primary"}`},
	}
	return seedBlocks(ctx, q, homeID, homeBlocks)
}
