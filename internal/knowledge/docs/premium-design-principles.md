# Premium design principles

The thing that separates "this looks AI-generated" from "this looks like an agency designed it" is not motion or gradient or font choice. It is calibration. Premium-feeling sites feel deliberate at every scale: the macro layout, the meso block composition, the micro state transitions. The agent's job is to be deliberate.

## Asymmetry as composition

Symmetric layouts feel safe and read as filler. Premium sites use asymmetry intentionally:

- Hero with image dominant on one side, condensed text on the other
- Stat grid with one stat oversized and the rest in a small row
- Pricing with one tier visually elevated and the others quieter
- Testimonials where one quote takes the full width and others are compact

Asymmetry should serve the content hierarchy, not be decoration. When everything is "balanced" the eye has nowhere to land.

## Type-scale contrast

The strongest single signal of taste is the gap between hero typography and body typography. A hero at 80px next to body at 16px (5x ratio) feels confident. A hero at 40px next to body at 16px (2.5x ratio) feels timid. Push the hero. The page will breathe.

Use the modular scale from `atomicsite://knowledge/typography-scale`. Reach for the top of the scale on hero pages.

## Calibrated color

Premium palettes are slightly desaturated, slightly cool, with high luminance contrast between elements. Saturation reads as cheap; high contrast reads as deliberate. See `atomicsite://knowledge/color-system` for derivation patterns.

The accent color appears at one or two moments per page, not seven. The rest is neutral.

## Perpetual micro-motion

Every interactive element responds to hover and focus. The motion is small (1-2px translate), fast (under 200ms), and predictable. The user does not notice each individual response; the cumulative effect is "this site feels alive."

```css
.btn,
.card,
.nav-link,
.icon-btn {
  transition: transform var(--motion-fast) var(--ease-out),
              background var(--motion-fast) var(--ease-out);
}
.btn:hover  { transform: translateY(-1px); }
.card:hover { transform: translateY(-2px); }
```

See `atomicsite://knowledge/motion-curves`.

## Hardware-accelerated only

Premium feels smooth. Smooth means 60fps on a 4-year-old phone. The only way to get there is animating composited properties (`transform`, `opacity`) and never animating layout-affecting properties (`width`, `height`, `padding`, `top`).

## Whitespace as a feature

Most AI-generated layouts undershoot whitespace. The fix is generous block padding (`var(--space-12)` to `var(--space-16)` on marketing pages), generous container max-widths, and confident gaps between elements (`var(--space-6)` to `var(--space-8)` between cards).

The visual breath is what makes a section feel curated rather than crammed.

## One idea per block

Each block carries one message. The hero pitches the value prop. The feature grid lists three to six benefits. The testimonial block delivers one strong quote, not five medium ones. When a block tries to do three jobs, the agent splits it into three blocks.

## Editorial typography

Body copy on premium marketing pages reads like an editor touched it. Short paragraphs (1 to 4 sentences), pulled-out emphasis on the strongest line, generous line-height (1.7 on long-form prose, 1.5 on UI labels). The agent does not paste corporate-speak; it writes like the user.

## Content density per fold

A homepage's first viewport contains: a single H1, one short subhead, one CTA, one visual. That is it. If the second viewport's content shows above the fold on tall monitors, that is fine; the deliberate first viewport is what the eye absorbs.

Loading the first viewport with three CTAs, four trust badges, and a video tells the user "we did not pick what mattered."

## Respect the operator

Atomic Site sites belong to operators, not the agent. The agent honors the operator's existing brand: their logo, their primary color, their voice. Premium-feeling does not mean overriding the operator's identity. It means executing their identity at the highest standard.

When the operator supplies a Figma file, the agent imports tokens and works within them rather than proposing alternatives. When the operator supplies a brand voice in the knowledgebase, the agent matches it.

## Reference taste, do not imitate

Linear, Stripe, Vercel, Resend, Apple are common references. The agent reads the principles, not the surface. Stripe's trust signals are not "use Stripe's purple"; they are clean typography, calibrated color, generous whitespace, perpetual micro-motion. Apply those, not the colors.

The user's installed taste-skill, high-end-visual-design, brutalist-ui, and minimalist-ui playbooks are external references the agent can register via `add_design_reference` and read through `atomicsite://design-references`.

## What separates premium from generic

A non-exhaustive checklist:

- Hero has a clear single H1, no secondary headline competing for attention
- Buttons have explicit hover, focus, active, and disabled states
- Forms have visible labels, autocomplete attributes, and clear error messages
- Images have width and height attributes (no CLS)
- Fonts are preloaded, not lazy
- Color contrast meets WCAG AA on every text-on-background pair
- Spacing follows the scale; no 23px paddings
- Motion is fast, small, and consistent across components
- Dark mode is its own composition (when enabled)
- Footer has Privacy, Terms, Cookie Settings, registration number
- Site has Open Graph image, llms.txt, sitemap.xml, robots.txt
- Eval grades A or higher across security, SEO, accessibility, privacy, performance

When all of these pass, the site feels premium because it is premium.
