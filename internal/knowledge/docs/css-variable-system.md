# The CSS variable system

Atomic Site sites are styled with vanilla CSS plus a small set of custom properties. There is no Tailwind, no CSS-in-JS, no utility framework. The whole visual system fits in `internal/builder/css.go` and the per-block CSS the renderers emit. Once you know the variables, you can style anything without touching the global stylesheet.

## The six color slots

Every site has the same palette shape, derived from `branding.primary_color` and friends:

```
--color-primary    /* hero CTA, link hover, accent on stats */
--color-secondary  /* secondary buttons, supporting accents */
--color-accent     /* tertiary, highlights, badges */
--color-text       /* body copy */
--color-bg         /* page background */
--color-surface    /* card / surface fills, derived from bg */
--color-border     /* hairlines and dividers */
--color-muted      /* secondary text, captions, helper labels */
```

Use these instead of hex codes. The branding admin and Figma token import both write to these variables, so any site-wide color change works without rebuilding pages.

## Layout variables

```
--container-narrow   /* 64rem */
--container-default  /* 72rem */
--container-wide     /* 80rem */
--container-fluid    /* 100% */
```

`general.container_width` selects which one applies to the body. Inside a block, you can override with a local variable on the block's wrapper element.

## Breakpoint variables

```
--bp-mobile   /* 640px default; configurable via general.mobile_breakpoint, range 320 to 960 */
--bp-tablet   /* 1024px default; configurable via general.tablet_breakpoint, range 640 to 1280 */
```

Reference them in media queries as `@media (min-width: var(--bp-tablet))`. Hard-coded breakpoints drift; the variables stay in sync with what the wider site is doing.

## Motion tokens

```
--motion-fast    /* 150ms; use for hover/focus state changes */
--motion-normal  /* 200ms; default transition duration */
```

Combined with hardware-accelerated properties only (`transform`, `opacity`), this keeps motion smooth on low-end devices. See `atomicsite://knowledge/motion-curves` for the full discipline.

## How to write a block style

Inside a `raw_astro` block, scope styles with `<style>` and reference the variables:

```astro
<section class="testimonial-strip">
  <p>{quote}</p>
  <p class="author">{author}</p>
</section>

<style>
  .testimonial-strip {
    background: var(--color-surface);
    color: var(--color-text);
    border-left: 4px solid var(--color-primary);
    padding: 2rem;
    border-radius: 0.5rem;
  }
  .testimonial-strip .author {
    color: var(--color-muted);
    font-size: 0.875rem;
    margin-top: 1rem;
  }
  @media (min-width: 1024px) {
    .testimonial-strip {
      padding: 3rem 4rem;
    }
  }
</style>
```

Astro scopes the `<style>` block to the component, so class names will not collide with the rest of the site.

## When to add a global class

If a pattern repeats across blocks, add it to the site's CSS Classes list (`upsert_css_class` tool). The class is emitted into `global.css` and is available to every page. Use this for buttons, badges, and other primitives the design system references many times. Do not duplicate the same `<style>` block across `raw_astro` blocks.

## What you must not do

Do not hard-code hex codes or rgb() values. Every color reference goes through a variable so the branding system keeps working.

Do not write `@import` rules that pull external stylesheets into a page. The CSP blocks them and the eval engine flags them.

Do not name custom variables `--color-anything-else`. Stick to per-block-local variables (`--testimonial-padding`) so the global namespace is owned by the system.

Do not paste pre-Tailwind utility classes (`flex flex-col gap-4`). They will not work; the built site has no utility CSS bundle.

## How dark mode hooks in

When a site enables dark mode, the same variables flip values under `[data-theme="dark"]`. Use the variables, not hard-coded values, and your block adapts for free. The full pattern is in `atomicsite://knowledge/dark-mode`.
