# Astro conventions in Atomic Site

Atomic Site emits Astro 4 sites. The builder owns the layout shell, the head element, security headers, hreflang alternates, and the block render loop. Authors (and you, the agent) plug in pages and blocks; the framework handles the rest.

## What the builder writes

`internal/builder/layouts.go` produces `src/layouts/Layout.astro`. It exports a TypeScript interface for `title`, `description`, `ogImage`, `robots`, `lang`, `alternates`, `hideGlobalBlocks`. Every page imports it. You do not modify Layout.astro through MCP.

`internal/builder/pages.go::RenderPages` walks every published page from the database, applies the meta-title and meta-description templates from settings.seo, computes hreflang alternates, then renders each block in `sort_order`. Output goes to `src/pages/<slug>.astro` (with nested folders for slugs containing `/`).

`internal/builder/css.go` emits `src/styles/global.css` with custom properties for color, container, and breakpoints. There is no Tailwind in the built site. Style your output with the CSS variables described in `atomicsite://knowledge/css-variable-system`, or scope per-block CSS inside `<style>` tags inside `raw_astro` blocks when you need something the variable system does not cover.

## What you write through MCP

Pages and blocks. Every page is one row. Every block is one row keyed by `block_type` and a `data_json` payload that conforms to the block's schema (see `atomicsite://knowledge/block-renderer-patterns`). Do not author `.astro` files through the filesystem. The build is deterministic from the database and a manual file edit is overwritten on the next `trigger_build`.

When a block type does not fit, use `raw_astro`. It accepts arbitrary Astro source and is admin-only because it bypasses the schema-validated render path.

## Hydration directives

The builder defaults to zero-JS pages. When a block needs interactivity, use Astro's island directives:

- `client:load` for above-the-fold interactive components (rare; costs LCP)
- `client:idle` for nice-to-have components that can wait
- `client:visible` for below-the-fold interactivity (the default for islands)
- `client:media="(min-width: 1024px)"` to gate by viewport

Heavy SDKs like Cal.com, Stripe Elements, or chat widgets should be hydrated `client:visible` or behind a user gesture, not `client:load`.

## Server vs static

Atomic Site builds static output (`output: "static"` in `astro.config.mjs`). There is no SSR. If you need server-rendered behaviour, use a small inline `<script>` for client-side enhancement or wire it through a third-party service (allowed_scripts handles the CSP whitelist).

## File routing

`src/pages/index.astro` is the homepage. `src/pages/about.astro` becomes `/about/`. Nested slugs become folders: a page with slug `blog/post-name` becomes `src/pages/blog/post-name.astro` and serves at `/blog/post-name/`. The page-slug validator (`internal/handlers/pages.go::validatePageSlug`) rejects path traversal, control chars, and labels longer than 80 characters per segment.

## Trailing slashes

Atomic Site canonicalises every URL with a trailing slash. The builder emits hreflang alternates with trailing slashes (`/en/` not `/en`) so the canonical and the alternates match. This is not optional; a self-referencing hreflang without the trailing slash fails the SEO eval.

## What "deploy" means

`trigger_build` produces `dist/` from `src/`. The deploy step copies `dist/` to the configured target (Caddy in cloud mode, an arbitrary host in single-domain mode). Static output, no runtime. If your block needs a database, that database lives outside the built site and you call into it from a `<script>` tag or via a CSP-whitelisted iframe.
