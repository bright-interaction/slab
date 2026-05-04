# Nav UX

Navigation is the most-touched component on most marketing sites. It earns careful design and almost no clever-ness. The agent's job is to keep it predictable, fast, and accessible across viewports.

## Header structure

Three slots from left to right on desktop:

```
[Logo]  [Primary nav]                 [Secondary CTA]
```

Logo links home. Primary nav holds three to six top-level items. Secondary CTA is the conversion ask (Start trial, Book demo, Sign in). The eval engine flags headers with more than seven primary nav items as cluttered.

## Mobile breakpoint

The mobile breakpoint variable `--bp-mobile` (default 640px) defines where the layout collapses. Below it, primary nav becomes a hamburger that toggles a full-screen menu. Above it, primary nav lays out horizontally.

The hamburger button has:

- `aria-expanded="true|false"` reflecting state
- `aria-controls="primary-nav"` pointing at the menu container
- A real `<button>`, not a div with click handler

## Sticky on scroll

Common pattern: header is `position: sticky; inset-block-start: 0` so it stays visible as the user scrolls. When sticky:

- Background gets a subtle backdrop-filter or solid color (so content does not bleed through)
- Border or shadow appears below to detach it from the page
- The transition uses `--motion-fast` and respects `prefers-reduced-motion`

```css
.header {
  position: sticky;
  inset-block-start: 0;
  z-index: 50;
  background: color-mix(in srgb, var(--color-bg) 92%, transparent);
  backdrop-filter: blur(12px);
  border-block-end: 1px solid var(--color-border);
}
```

When the operator wants the header to hide on scroll-down and show on scroll-up, that is a JS pattern. Use sparingly; many users find it disorienting.

## Active state

The current page's nav item shows an active state: bolder weight, primary-color underline, or a subtle background. The pattern is consistent across breakpoints. Do not use color alone; pair with a non-color signal (weight, underline) so colorblind users have access.

## In-page anchors

When the page has anchor sections (pricing, FAQ), nav links target them with `#anchor-id`. The agent ensures:

- Each section has a stable `id`
- Anchored sections get `scroll-margin-block-start: 6rem` (or the height of the sticky header) so the heading is not hidden under it
- The browser-native smooth-scroll honors `prefers-reduced-motion`

## Footer

Multi-column on desktop, single-column stack on mobile. Typical columns: Product / Company / Resources / Legal. Each column has a heading and a short link list. The footer is the safety net for SEO internal linking; aim for 12 to 24 internal links across columns, all with descriptive anchor text.

Legal column always has: Privacy Policy, Terms, Cookie Settings (opens the consent preferences modal), and the operator's company name + registration number. The agent reads `atomicsite://site/profile` to populate these.

## Breadcrumbs

Pages deeper than the homepage emit a `<nav aria-label="breadcrumb">` with the path. The builder generates this from the page slug; the agent does not hand-write it.

## Search

Most marketing sites do not need search. When they do, route to a static search index (Astro's content collections) rather than a third-party widget. A search input that returns nothing useful is worse than no search.

## What you must not do

- Do not put a CTA in the primary nav slot that competes with the secondary CTA
- Do not nest dropdown menus more than two levels (root -> category -> page)
- Do not animate the hamburger icon to a complex morph; a simple cross is enough
- Do not autoplay focus into the search input on page load. It steals focus from screen readers.

## What the eval grades

`internal/eval/accessibility.go` and `internal/eval/seo.go` between them check: nav landmarks present, aria-current on active links, descriptive anchor text, breadcrumbs schema-marked, footer has Privacy + Terms links. The discipline above passes all of these by default.
