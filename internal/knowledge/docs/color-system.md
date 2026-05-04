# Color system

Premium-feeling sites have calibrated color, not enthusiastic color. The agent's job is to use the operator's chosen primary as a single reference point and derive everything else with discipline.

## The starting point

Branding gives the agent six seed colors via the eight CSS variables in `atomicsite://knowledge/css-variable-system`. The most consequential is `--color-primary`. Treat it as the brand voice, used sparingly: hero CTA, link hover, one or two accent moments per page. A page where every other element is `--color-primary` reads as enthusiastic and amateur.

## Neutral foundation

Most surface area on a premium site is neutrals: white, near-white, light gray, mid-gray, dark gray, near-black. The variables `--color-bg`, `--color-surface`, `--color-text`, `--color-muted`, `--color-border` carry the neutral system. Build pages on neutrals, accent with primary.

For light-mode neutrals, a clean ladder:

```
--color-bg      : #ffffff   /* canvas */
--color-surface : #f8f9fb   /* card surface, subtle elevation */
--color-border  : #e5e7eb   /* hairlines */
--color-muted   : #6b7280   /* secondary text, captions */
--color-text    : #111827   /* primary text */
```

For dark-mode, invert the relationships, not the values. Dark mode is not light mode with hue rotation; it is its own composition. See `atomicsite://knowledge/dark-mode`.

## Contrast ratios

The accessibility eval (`internal/eval/accessibility.go`) checks WCAG AA contrast: 4.5:1 for body text, 3:1 for large text and UI components. Plug the operator's primary into a contrast checker against the chosen text color before committing. If the primary fails against `--color-bg`, the agent should:

1. Propose a slightly darker primary for text-on-background uses
2. Keep the original primary for non-text accents (icon strokes, borders)
3. Not silently weaken the contrast guard

## Two accents are usually enough

Primary plus one secondary accent covers most marketing sites. A third accent (`--color-accent`) earns its place when the design needs a clear hierarchy of three CTAs (hero CTA primary, secondary action secondary, tertiary action accent). More than three competing accent colors and the page reads as a fairground.

## Calibrated, not saturated

Saturation reads as cheap. Premium palettes lean toward slightly desaturated, slightly cool primaries with high luminance contrast. When the operator's primary is a saturated red or green, the agent can:

- Use the saturated primary only at small surface areas (icons, badges, links)
- Derive a desaturated companion for larger surfaces (button backgrounds at 80% saturation, card borders at 60%)

These derivations live as block-local variables, not new global slots.

## Semantic color

Buttons need pressed, hover, focus, disabled states. The agent uses opacity or HSL adjustment, not new variables. Examples:

```
.btn:hover  { background: color-mix(in srgb, var(--color-primary) 90%, black); }
.btn:active { background: color-mix(in srgb, var(--color-primary) 80%, black); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
```

`color-mix()` is supported in every browser the eval engine targets.

## Status colors

When a site needs success / warning / danger / info colors, derive them, do not invent. Defaults:

```
--color-success : #10b981
--color-warning : #f59e0b
--color-danger  : #ef4444
--color-info    : #3b82f6
```

Override per-site only when the brand explicitly requires it. Most marketing sites use the defaults.

## Figma import path

When the operator uses `import_figma_tokens`, the imported palette overwrites the default slots. The agent should diff before applying: if Figma has 14 colors and Atomic Site has 6 slots, propose a mapping rather than dropping eight values. The Figma handler exposes the raw payload via `atomicsite://figma/imports/<id>`.

## What you must not do

- Do not paste hex codes into per-block CSS. Use variables.
- Do not introduce new global color slots without coordinating with the branding admin.
- Do not mix two operators' palettes in a single block (e.g., a Stripe-blue button on a Linear-purple site).
- Do not assume the operator's primary works on every surface. Test against `--color-bg` and `--color-text` before applying.
