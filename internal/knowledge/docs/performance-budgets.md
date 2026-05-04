# Performance budgets

Atomic Site builds static sites; the agent's job is to keep them static and small. Every byte the page does not ship is a byte that does not have to be parsed, executed, or paid for. The eval engine measures hard caps; the discipline below keeps sites well inside them.

## The hard caps

`internal/eval/performance.go` enforces:

- HTML page size under 200KB rendered
- Inline `<style>` blocks total under 50KB
- All `<script>` tags in `<head>` must be `async`, `defer`, or `type="module"`
- No render-blocking external CSS or JS

Violations lower the eval grade and ship as warnings on every build.

## LCP image

The largest contentful paint image (usually the hero) gets:

- `loading="eager"` (not lazy)
- `fetchpriority="high"`
- A `<link rel="preload" as="image">` in `<head>`
- Width and height attributes to prevent CLS
- A WebP or AVIF variant served via `<picture>` with the original as fallback

The image pipeline auto-generates the variants; the agent does not manually re-encode. The block renderer for the hero block sets the priority hints automatically when the block has an image_id.

## Below-the-fold images

Every other image gets `loading="lazy"`. The block renderers do this by default for `image_id` fields except in hero contexts. When using `raw_astro` to ship an image, the agent adds `loading="lazy"` explicitly.

## Fonts

Two woff2 files preloaded, the rest lazy. The branding admin tracks which fonts are registered; the builder emits preloads for the heading and body weights. The agent does not author preloads; it picks the right weights in the typography settings.

`font-display: swap` on every `@font-face`. Atomic Site does this automatically; do not override to `optional` or `block` without a reason.

## Scripts

Every external script ships through `register_allowed_script`, which routes it through `<script defer integrity="...">`. Inline scripts are admin-only (in `raw_astro`) and add to the inline-CSS budget conceptually because they parse on the main thread.

Heavy SDKs (Cal.com, Stripe Elements, chat widgets, video players) lazy-load on user gesture or `IntersectionObserver`. Do not put them in `<head>` to "be safe"; they tank LCP.

## Critical CSS

The builder inlines critical above-the-fold CSS (~20KB ceiling) and externalises the rest. The agent does not pick what is critical; the builder does, by analysing the rendered page. What the agent controls is total size: do not write 100KB of CSS, do not duplicate styles across blocks, use the global CSS classes when a pattern repeats.

## Total page weight

Targets per page kind:

- Marketing landing: under 500KB transferred (HTML + CSS + JS + fonts + LCP image)
- Documentation: under 300KB (no hero image, less JS)
- Blog post: under 600KB (more images allowed)

These are targets, not hard caps. The eval's hard caps are smaller per-asset. Hit those first.

## Third-party budget

Budget for third-party origins per page:

- Hero / above-the-fold: zero
- Mid-page (logo carousel from a partner CDN, embedded video poster): two
- Below-the-fold (chat widget, analytics ping, embedded calendar): three more

Every third-party origin is a separate DNS resolve, TCP handshake, TLS negotiation. The CSP whitelist makes them legal; performance still suffers.

## Core Web Vitals

LCP, INP, CLS thresholds for "Good":

- LCP under 2.5s
- INP under 200ms
- CLS under 0.1

Atomic Site's static output and image pipeline make these achievable by default. The agent kills them by:

- Using a heavy hero font without preloading: bad LCP
- Adding a 5MB unoptimised image: bad LCP
- Animating layout properties: bad CLS
- Loading a heavy SDK on click without an interaction-ready loading state: bad INP

## What you must not do

- Do not paste base64 image data into HTML. Upload to media library.
- Do not import a 100KB icon font when 4 SVG icons would do.
- Do not load a charting library to render two numbers; use HTML and CSS.
- Do not enable a third-party tag manager unless the operator explicitly requested it. They are CSP nightmares and performance black holes.
