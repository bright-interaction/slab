# Typography scale

The defaults across Atomic Site sites: Inter (sans), Geist (display), JetBrains Mono (mono). Operators swap fonts via the branding admin or a Figma token import; the agent never edits font-family directly. The agent's job is rhythm: scale, weight, line-height, tracking.

## The scale

A 1.250 modular scale from a 1rem (16px) base produces clean, legible body and reasonable display sizes. The agent uses these tokens:

```
--fs-xs   : 0.75rem   /* captions, badges */
--fs-sm   : 0.875rem  /* metadata, secondary labels */
--fs-base : 1rem      /* body */
--fs-lg   : 1.125rem  /* lead body, larger UI labels */
--fs-xl   : 1.25rem   /* small headings, callouts */
--fs-2xl  : 1.5rem    /* h3 default */
--fs-3xl  : 2rem      /* h2 default */
--fs-4xl  : 2.5rem    /* h1 default on content pages */
--fs-5xl  : 3.5rem    /* hero h1 */
--fs-6xl  : 5rem      /* oversized display, sparingly */
```

These are emitted as part of the global stylesheet when typography is enabled. Reference them in per-block CSS instead of hard-coding pixel values.

## Line height

Headings get tight line-height to feel decisive. Body gets generous line-height to feel readable. The defaults:

```
--lh-tight  : 1.1   /* h1, h2, hero copy */
--lh-snug   : 1.3   /* h3, large UI labels */
--lh-base   : 1.5   /* body */
--lh-relaxed: 1.7   /* long-form prose, blog body */
```

## Weight

```
--fw-regular: 400
--fw-medium : 500
--fw-semibold: 600
--fw-bold   : 700
```

Inter ships variable weights; pick from the scale above. Avoid 300 (too thin on coloured backgrounds) and 800+ (rarely earns its visual weight). Display fonts (Geist) shine at 600 to 700; body sits at 400 with 600 for emphasis.

## Tracking

Headings tighten slightly; body stays neutral. The defaults:

```
--ls-tight  : -0.02em   /* h1, hero */
--ls-snug   : -0.01em   /* h2, h3 */
--ls-normal : 0         /* body */
--ls-wide   : 0.05em    /* small caps, eyebrows */
```

## Loading and FOIT

Fonts load via `@font-face` with `font-display: swap`, served from the operator's woff2 uploads (no Google Fonts; no third-party origin). The body falls back to `system-ui, -apple-system, "Segoe UI", Roboto, sans-serif` until Inter resolves. Heading falls back to the same stack, weight inherited.

For LCP-critical heading typography, preload the woff2 (the builder emits `<link rel="preload" as="font">` automatically when the font is registered). Do not propose preloading every weight; one or two per face is the budget.

## Vertical rhythm

Pair the scale with consistent vertical spacing. Headings get `margin-block-start: 2em; margin-block-end: 0.5em` relative to their own font-size. Body paragraphs get `margin-block: 1em`. This produces a visual cadence that survives content edits.

## When to break the scale

Break it when the design needs a single oversized statement: a homepage hero with a 7rem headline, a quote section with massive type. Break it deliberately, in one place, with a comment in the block CSS noting the break. Two breaks per page tells you the scale is wrong; tune the base.

## What the eval grades

Eval does not directly score typography. But mis-scaled typography produces:

- Poor accessibility scores when `font-size` falls below 14px on body text
- Poor readability when line-height is too tight (Flesch-Kincaid grading)
- Poor LCP when the hero font is heavy and not preloaded

So the rhythm matters even though it is not graded explicitly.
