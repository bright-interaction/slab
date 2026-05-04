# Collection design patterns

Custom Collections are atomicsite's CMS layer. A Collection is a user-defined content type (case studies, products, team members, FAQ entries, events, jobs, recipes, reviews) with a schema. Items conform to the schema and either render as standalone pages or feed into a `collection_list` block on a page.

This doc tells you when to reach for a Collection, how to design its field schema, and how to wire its items so the rest of the platform (Astro renderer, eval engine, hreflang, personalization) does the heavy lifting for you.

## When to use a Collection vs a Component vs a global block

Three primitives, three different jobs. Picking wrong wastes the schema you put around the right one.

- **Component**: a reusable Astro template. Used when you need the same visual shape (a CTA card, a stat tile, a quote block) at multiple places, with different content but identical markup. Components have `props_schema` declaring what data goes in. Pick this when you have one shape and many one-off contents.
- **Global block**: site-wide blocks like header, footer, cookie banner. Inserted on every page automatically. Pick this when the same instance (header X, footer Y) appears on every page; never for content that differs per page.
- **Collection**: a content type with many instances over time. Each instance has the same fields but different values, and each one matters as a unit (you'd want to link to it, share it, search for it). Pick this when you'd add new entries to it over the lifetime of the site.

The rule of thumb: if you'll add a new instance of this thing every week or month, it's a Collection. If you'll set it up once and forget it, it's a Component or a global block.

## Designing the field schema

A Collection's schema is an array of `FieldDef` objects: `{name, type, label, required, options?, max_length?, ref_collection?}`. The 16 supported types map to a real concept the renderer knows how to display:

- `text`: short single-line. Max 5 KB.
- `textarea`: multi-line plain text. Max 50 KB.
- `richtext`: longer formatted body, served as escaped HTML. Max 500 KB.
- `number`: int or float; the renderer outputs the raw value.
- `boolean`: true/false; renders as "yes" / "no" by default.
- `date`: ISO-8601 date (YYYY-MM-DD); rendered inside `<time datetime>`.
- `datetime`: RFC 3339 datetime; same `<time>` treatment.
- `url`: absolute URL with scheme + host.
- `email`: validated against `mail.ParseAddress`.
- `image`: media id reference; the renderer emits a proper `<picture>` with WebP source + width / height.
- `gallery`: array of media ids; rendered as a row of `<picture>` elements.
- `select`: single value from a fixed `options` list.
- `multiselect`: array of values from `options`.
- `reference`: foreign key to another Collection's item id (used by Memberships in Sprint 5).
- `color`: `#RRGGBB` hex.
- `json`: free-form blob for shapes the renderer should not care about.

Field-design rules of thumb:

1. Make required fields actually required. The validator rejects items missing required fields; if a field is "nice to have", mark it optional and the renderer will skip it gracefully.
2. Pick the narrowest type that fits. `select` beats `text` whenever you can enumerate the values, because the eval engine and the agent both reason about enums.
3. Reuse common field names. The JSON-LD generator looks for `headline`, `author`, `datePublished` for Article; `name`, `description`, `price`, `brand` for Product; `name`, `jobTitle`, `email` for Person. Naming your fields the schema.org way costs nothing and gets you free structured data.
4. Cap text lengths intentionally. `text` is for a tagline, not a paragraph. `textarea` is for a paragraph or two. `richtext` is for a body. The platform enforces these caps so a misbehaving import cannot blow up `data_json`.

## Slug strategy

Each Collection has a slug (kebab-case, max 64 chars). Each item also has a slug, unique within its Collection-locale tuple.

The renderer routes items as `/{collection.slug}/{item.slug}/`. With `additional_langs` configured, locale variants land at `/{locale}/{collection.slug}/{item.slug}/`. The same item slug may appear in multiple locales: the platform treats `/case-studies/acme` (English) and `/sv/case-studies/akme` (Swedish) as alternates and emits hreflang automatically, but only if both items exist with the right `locale` field.

Slug rules:

1. Stable. Once you publish an item, do not change its slug. Search engines and link shares will break. If you must rename, set up a redirect.
2. Descriptive. Slug is part of the URL and most people read it. `case-studies/acme-corp-2024-revenue-doubling` is fine; `case-studies/cs-001` is not.
3. Unique per locale, not per Collection. A Swedish translation can keep the same slug as the English original, or use a localised slug. The platform handles both.

## Locale variant model

A Collection item has a `locale` field. Empty string means "uses the site default lang" (`sites.lang`). Non-empty must match an entry in `settings.general.additional_langs` for hreflang to work end-to-end.

The hreflang machinery is the same one that drives pages: per-item alternates emit only when the item slug exists in multiple locales for the same Collection. Single-locale items get no hreflang noise. Sites with `seo.hreflang_strategy = "off"` get nothing regardless.

Authoring tip: when you add a Swedish translation of an English item, keep the same slug if the canonical name is the same brand or proper noun. Translate the slug only when the title is meaningfully different in the new language.

## Personalization at item granularity

Each item's `data` map can carry a `condition` field (string DSL). The renderer wraps the rendered card or detail body in `<div data-asp-when="..." hidden>`, and the visitor-hydration script unhides matching nodes at runtime. This is the same DSL used on regular blocks: `industry == "finance"`, `lead_score > 100`, `trial_days_left present`.

Use cases that work cleanly:

- Show industry-specific case studies when the visitor's CRM record indicates they work in that industry.
- Hide expired job postings without un-publishing the row (set `condition` to `posting_active == "true"`).
- Personalise product recommendations per traffic source.

Cases that do not work:

- Per-visitor item ordering (the build emits a fixed order).
- Member-gated content (Sprint 5 introduces a `requires_member_role` setting on the Collection; until then, treat membership as out of scope).
- Real-time pricing (the build is static; use a regular block with client-side fetch if you need live data).

## Static filter on `collection_list` blocks

The `collection_list` block accepts a `filter` field that runs at build time. It supports the two simplest forms of the personalization DSL: `field == "value"` and `field present`. Anything more complex (boolean operators, comparisons) is evaluated at runtime via the same hydration script that handles `data-asp-when` on individual items.

Static filter is for "show only featured items" or "show only the European product line". Dynamic filter (per-visitor industry, etc.) is for the cards themselves via item-level `condition`.

## Authoring with the agent

The MCP tool `bulk_import_collection_items` accepts a JSON array of items. Validate before sending: every `data` object must satisfy the Collection schema (required fields present, type-correct values, options match). The server validates again on receipt and rolls back the entire batch on any single failure, so a malformed item never half-imports.

A productive agent loop for filling a fresh Collection:

1. Call `create_collection` with name, slug, schema, and `settings.schema_org_type` if you want auto JSON-LD.
2. Generate the items as a JSON array (the agent's job, often from a brief or external source).
3. Call `bulk_import_collection_items` with the array. Inspect the response for the count and ids.
4. Trigger a build. The renderer emits index + per-item pages and JSON-LD if `render_as_pages` is true.
5. Read `/api/sites/{id}/evaluations/latest` to confirm the new Collection-aware checks pass.

If validation fails on import, the error tells you exactly which item index and which field; fix the row and resend the whole array.

## Anti-patterns

- Do not use a Collection for one-off pages. A static page is what `pages` is for.
- Do not nest content via `reference` to fake a one-to-many relationship the schema does not support. The renderer follows references shallowly; deep joins are out of scope.
- Do not bury layout choices inside `data_json`. The Collection schema describes content, not visuals; the `card_template` and `detail_page_template` settings control layout.
- Do not duplicate Collections per locale. Use `locale` per item, not "case-studies-en" and "case-studies-sv".
