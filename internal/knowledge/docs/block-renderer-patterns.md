# Block renderer patterns

Atomic Site composes pages from blocks. There are 19 registered block types, each with a schema describing its fields. Every page is a sorted sequence of blocks. The agent's primary job is choosing the right blocks, in the right order, with the right data.

## The 19 block types

Hero family: `hero`, `split_hero`, `header` (global), `footer` (global).
Content: `text`, `quote`, `code_block`, `accordion_faq`, `process_steps`, `about_split`.
Social proof: `logo_strip`, `logo_carousel`, `stat_grid`, `replacement_grid`.
Conversion: `feature_grid`, `pricing`, `cta`, `form`.
Embed: `image`, `embed`, `custom`, `raw_astro`.

The full schema lives in `internal/blocks/registry.go`. Always pull `atomicsite://site/context` first so the block schemas the agent has match the running server's registry version.

## Schema kinds

Each field on a block has one of ten kinds:

- `text`: single-line string
- `textarea`: multi-line plain string
- `richtext`: markdown-with-some-HTML, rendered through the safe pipeline
- `url`: validated URL
- `image_id`: references the media library; the renderer resolves the URL + alt text + dimensions
- `select`: one value from `Options`
- `bool`: true or false
- `number`: numeric, with optional min/max
- `array`: repeated `ItemSchema` (e.g., pricing tiers, FAQ Q&A)
- `object`: nested fields under one key

When the agent calls `create_block` or `update_block`, the `data_json` payload must conform to the schema. Validation rejects unknown keys, missing required fields, and wrong kinds.

## Marketing page composition

The canonical sequence for a SaaS landing page (from `EvalPlaybook`):

1. `split_hero` or `hero`: single H1 carrying the page intent
2. `logo_carousel` or `logo_strip`: social proof above the fold of the second viewport
3. `stat_grid`: credibility (numbers customers care about)
4. `replacement_grid`: what we replace, how we are different
5. `feature_grid`: three to six benefits, not features
6. `text` or `about_split`: narrative breath in the middle
7. `pricing`: when relevant; one tier emphasised
8. `accordion_faq`: objection handling
9. `cta`: final ask, single action

Two H1s on a page fail eval. The hero owns the H1; subsequent blocks render H2 and below.

## When to use raw_astro

`raw_astro` accepts arbitrary Astro source. It bypasses the schema, so every other validation does not apply, and it is admin-only. Use it when:

- You need a one-off layout no registered block covers (an animated gradient wall behind a hero)
- You are integrating a third-party component that needs Astro's frontmatter (a Cal.com inline embed with custom config)
- A block's design needs a CSS pattern the renderer does not expose

When you reach for `raw_astro`, ask whether a new registered block would serve future sites better. Custom components (`upsert_css_class` plus a registered block schema) are reusable; `raw_astro` is not.

## Sort order

Blocks render in `sort_order` ascending. When inserting in the middle, gap the orders by 10 (10, 20, 30) so future inserts do not require renumbering. The bulk-save endpoint normalises orders on its own when needed.

## Visibility

`is_visible: false` keeps the block in the database but excludes it from the build. Useful for staging blocks for review without exposing them. Hidden blocks still count against drafts of analytics queries and personalization rules; do not use visibility as a delete substitute.

## Global blocks

`header` and `footer` are global. One per site, applied to every page unless `page.hide_global_blocks = true`. Edit them via the global-block endpoints, not as page blocks. Most landing pages should keep the global header and footer; one-off conversion pages (post-click landers) often hide them.

## How the renderer reads data

`internal/builder/pages.go::renderBlock` switches on `block_type`, applies the schema-validated data to the right Astro template, and emits the section into the page. The same render path runs in production and in the live preview iframe (see `RenderBlockDraft`). What you see in the editor is what tenants see, byte for byte.
