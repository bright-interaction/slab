package critique

// Inspiration is one curated reference the agent gets back from
// create_block / lint_block (gap 5 of 6, 2026-05-21). Each entry
// points at a real file in the bundled MIT-licensed design corpus
// the agent can read via the design corpus tools, plus a one-line
// "why this is on the list" note so the agent can pick fast without
// reading all three.
type Inspiration struct {
	Repo     string `json:"repo"`
	Path     string `json:"path"`
	Pattern  string `json:"pattern"`
	WhyRead  string `json:"why"`
	HowToUse string `json:"how_to_use"`
}

// InspirationsFor returns the curated 2-3 references that match a
// given block_type. The list is intentionally short: agents skim,
// they don't study, and three actually-relevant references beats ten
// "maybe relevant" ones. Keep entries pointed at well-designed,
// well-named, agency-grade examples, not toy components.
//
// Empty result is fine: not every block_type needs an external
// reference (text, image, code_block, embed are self-explanatory).
func InspirationsFor(blockType string) []Inspiration {
	switch blockType {
	case "hero":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Hero.astro",
				Pattern:  "centred hero with image right",
				WhyRead:  "The image-right variant of AstroWind's Hero ships the typography hierarchy + button spacing atomicsite's renderer mirrors. Read it to see how to balance an editorial headline with a CTA pair without crowding the eyebrow.",
				HowToUse: "Mirror the eyebrow > headline > subheading > primary CTA + secondary link order. Keep the headline <=12 words like the playbook's headline_length rule.",
			},
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Hero2.astro",
				Pattern:  "split hero with image left",
				WhyRead:  "Hero2's split layout is the right reference for any hero with a real product / dashboard / chart on one side. Atomicsite's split_hero block_type maps 1:1.",
				HowToUse: "Use block_type=split_hero when the image carries actual information; use block_type=hero (centred) when the message stands alone.",
			},
			{
				Repo:     "starlight",
				Path:     "design-references/starlight/packages/starlight/components/Hero.astro",
				Pattern:  "documentation hero with action set",
				WhyRead:  "Starlight's docs hero is the cleanest example of a hero that supports 2-3 CTAs without visual noise. Useful when the page sells multiple actions (audit + book call + read docs).",
				HowToUse: "Limit secondary CTAs to one. Two = analysis paralysis. Three = unsalvageable.",
			},
		}
	case "split_hero":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Hero2.astro",
				Pattern:  "split hero, image left, content right",
				WhyRead:  "AstroWind Hero2 is the canonical split. atomicsite's renderer ships the same grid breakpoints.",
				HowToUse: "Image carries the actual proposition (dashboard, chart, product), not decoration. Decorative image = use centred hero instead.",
			},
			{
				Repo:     "shadcn-ui",
				Path:     "design-references/shadcn-ui/apps/www/components/site-header.tsx",
				Pattern:  "tight typography pairing for tech audiences",
				WhyRead:  "shadcn's headline pairing (Geist Sans + Geist Mono) is the de-facto B2B tech standard. atomicsite's --font-heading / --font-mono variables are tuned for this look.",
				HowToUse: "If branding.font_heading is Geist or Inter and the audience is technical, this is your typography reference.",
			},
		}
	case "pricing":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Pricing.astro",
				Pattern:  "3-tier pricing card grid with featured middle tier",
				WhyRead:  "The middle-featured pattern (lift + accent border + 'Most popular' tag) is the highest-converting B2B pricing layout. Atomicsite's pricing block_type maps directly.",
				HowToUse: "Set the middle tier's featured=true so the renderer lifts it. Use 3 tiers (not 4, not 5); 4+ collapses choice on mobile.",
			},
			{
				Repo:     "shadcn-svelte",
				Path:     "design-references/shadcn-svelte/sites/docs/src/lib/registry/default/example/cards/pricing.svelte",
				Pattern:  "minimal pricing card",
				WhyRead:  "shadcn's pricing card is the reference for a less-busy, more luxury feel. Use when the brand is high-touch.",
				HowToUse: "Drop the feature list bullets if you want this vibe; keep just price + name + CTA.",
			},
		}
	case "accordion_faq":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/FAQs.astro",
				Pattern:  "two-column FAQ grid (no accordion)",
				WhyRead:  "Cleaner than an accordion when there are <=6 items. Avoids the click-to-reveal friction.",
				HowToUse: "If you have <=6 FAQ items, consider rendering as a two-column grid via block_type=text + style overrides instead of accordion_faq.",
			},
			{
				Repo:     "starlight",
				Path:     "design-references/starlight/packages/starlight/components/Tabs.astro",
				Pattern:  "tabs as alternative to accordion",
				WhyRead:  "When questions group into 2-3 audiences (developers / ops / legal), tabs read cleaner than a flat accordion.",
				HowToUse: "Atomicsite has no native tabs block_type yet; use a custom block_type=custom with tabs markup if the audience-grouping is needed.",
			},
		}
	case "stat_grid":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Stats.astro",
				Pattern:  "4-column number grid with label + context",
				WhyRead:  "AstroWind's Stats is the reference for above-the-fold trust signals. Atomicsite's stat_grid block_type maps 1:1.",
				HowToUse: "3-4 numbers max. Each carries: value (big), label (one line), context (one line under). Numbers must be real (the slop_number lint catches 99.99%, 1M+, etc).",
			},
		}
	case "feature_grid":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Features.astro",
				Pattern:  "3-up feature grid with icon + title + body",
				WhyRead:  "The canonical 3-column feature pattern. Atomicsite's feature_grid block_type renders the same shape.",
				HowToUse: "Stick to 3-4 items. 5+ stacks on mobile into an unreadable list; use replacement_grid or two separate feature_grid blocks instead.",
			},
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Features3.astro",
				Pattern:  "alternating-side feature rows",
				WhyRead:  "Features3 alternates image-left / image-right per row. Use when each feature deserves a screenshot.",
				HowToUse: "Better than feature_grid when feature count > 3 AND each feature has its own visual. Maps to block_type=about_split repeated with image_position alternating.",
			},
		}
	case "process_steps":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Steps.astro",
				Pattern:  "numbered 4-step vertical timeline",
				WhyRead:  "Atomicsite's process_steps block uses the same numbered timeline pattern. Read for spacing + numeral typography.",
				HowToUse: "Always exactly 4 steps. Number them 01-04 (mono font carries the eyebrow weight). Step 4 should resolve the journey, not introduce a new branch.",
			},
		}
	case "cta":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/CallToAction.astro",
				Pattern:  "centred CTA banner with single button",
				WhyRead:  "The CallToAction widget is the reference for closing-section CTAs. Atomicsite's cta block matches.",
				HowToUse: "ONE button only. Two = the visitor reads neither. Variant: variant=secondary for a softer mid-page banner; variant=primary for the final close at the end of the page.",
			},
		}
	case "testimonial", "quote":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Testimonials.astro",
				Pattern:  "3-up testimonial card grid",
				WhyRead:  "The grid pattern when you have 3+ testimonials. Below 3, use a single block_type=quote.",
				HowToUse: "Always attribute (name + role + company). Anonymous testimonials read as fake. Slop_company lint catches Acme / Initech / etc.",
			},
		}
	case "about_split":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Content.astro",
				Pattern:  "image + bio split with stats inline",
				WhyRead:  "Content widget shows how to pair a founder photo with copy + inline stats without the bio reading like a press release.",
				HowToUse: "Keep the photo to one side (image_position=right is default). 3 stats max. CTA at the bottom links to the long-form bio page.",
			},
		}
	case "logo_strip", "logo_carousel":
		return []Inspiration{
			{
				Repo:     "astrowind",
				Path:     "astrowind/src/components/widgets/Brands.astro",
				Pattern:  "horizontal logo strip with subtle desaturation",
				WhyRead:  "The Brands widget is the cleanest 'trusted by' bar reference. Real logos > placeholder names.",
				HowToUse: "Only ship logos for companies you've actually worked with. Empty logo_strip > fake logos; the slop_company lint catches placeholders.",
			},
		}
	}
	return nil
}
