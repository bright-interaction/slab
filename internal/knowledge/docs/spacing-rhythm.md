# Spacing rhythm

Premium-feeling sites have predictable rhythm. The agent uses a 4/8 base scale, applied consistently, broken only with intent. Spacing is the single most reliable signal of taste.

## The scale

```
--space-1 : 0.25rem  /* 4px;  micro */
--space-2 : 0.5rem   /* 8px;  small */
--space-3 : 0.75rem  /* 12px; small-medium */
--space-4 : 1rem     /* 16px; base */
--space-6 : 1.5rem   /* 24px; medium */
--space-8 : 2rem     /* 32px; large */
--space-12: 3rem     /* 48px; section padding */
--space-16: 4rem     /* 64px; major section breaks */
--space-24: 6rem     /* 96px; hero whitespace */
--space-32: 8rem     /* 128px; oversized whitespace, sparingly */
```

Stick to the scale. Reaching for 18px or 28px or 100px tells you something is wrong with the rhythm, not that the scale is wrong.

## Vertical rhythm between blocks

Each block has padding-block (top + bottom). The default cadence:

- Hero: `padding-block: var(--space-16)` (4rem). Breathes.
- Content blocks (text, feature_grid, stat_grid): `padding-block: var(--space-12)` (3rem)
- Compact blocks (logo_strip, cta): `padding-block: var(--space-8)` (2rem)

Adjacent blocks of the same kind get the same cadence; transitions between kinds preserve it. The result is an even visual heartbeat down the page.

## Horizontal rhythm inside a block

Container padding-inline scales with viewport: `var(--space-4)` mobile, `var(--space-8)` tablet, `var(--space-12)` desktop. Inside the container, grid gaps use `var(--space-6)` to `var(--space-8)` for content cards, smaller for tight clusters.

## When to break the grid

Break it for impact. A hero with asymmetric whitespace (image dominant left, text condensed right with negative offset) creates tension that draws the eye. Break it once per page, not three times. Two intentional breaks plus a default cadence reads as designed; three breaks plus a default cadence reads as broken.

## Type and space coupling

Section headings get more space above than below: `margin-block-start: var(--space-12); margin-block-end: var(--space-4)`. The asymmetry signals "new section" without a divider line.

Body paragraphs get `margin-block: var(--space-4)` (1em). Inline lists tighten to `margin-block: var(--space-2)`.

## Mobile

The whole scale halves at small viewports through media queries on the block wrappers. A `padding-block: var(--space-16)` on desktop becomes `var(--space-8)` on mobile. The cadence preserves but the surface adapts.

## What you must not do

- Do not nudge spacing by single pixels to "fix" alignment. Realign to the scale.
- Do not use `margin: auto` on every block; rely on container padding and grid gap.
- Do not paste `padding: 23px 17px`. The scale is there for a reason.
- Do not let a block grow flush against the next; preserve the cadence even when the design feels uniform.

## What the eval grades

Spacing is not directly graded by eval, but inconsistent spacing produces:

- Poor LCP measurements when above-the-fold blocks shift on hydration
- Poor accessibility scores when interactive elements are too tightly clustered (touch-target rule: 44x44px)

The agent treats the scale as a non-negotiable; the eval scores follow.
