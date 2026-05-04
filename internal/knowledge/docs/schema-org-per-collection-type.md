# Schema.org per Collection type

When a Collection has `settings.schema_org_type` set, the builder emits a `<script type="application/ld+json">` on every item detail page. The contents are derived from the item's `data` fields plus the Collection schema, so naming your fields the schema.org way gets you valid structured data with zero extra work.

This doc lists every supported type, the canonical field names that auto-map, and what the eval engine checks for each.

## How the mapping works

The JSON-LD builder reads the item's `data_json`, looks up the Collection's `schema_org_type`, and emits a JSON object with `@context: "https://schema.org"` and `@type` set to that value. Every item already gets `name` (from `item.title`) and `url` (the canonical absolute URL, computed from `site.Domain` plus the item path). The type-specific mapping then layers on top.

When a field maps to a property the type expects, the builder includes it. When a field has no mapping, it is silently dropped from the JSON-LD body but still rendered into the visible page. The eval check `Collection JSON-LD present` fails when the script tag is missing or `@type` is empty; it does not validate property completeness, so an Article with no author still passes structurally.

## Supported types and field-name conventions

The table below shows which item field names auto-map for each type. Field names are case-sensitive and snake_case. When two names appear together (e.g. `headline | title`), the first match wins.

### Article, BlogPosting, NewsArticle

- `headline | title | name` → `headline`
- `description | summary | lead` → `description`
- `author | author_name | byline` → `author` (wrapped as `{@type: "Person", name}`)
- `date_published | datePublished | published_at` → `datePublished`
- `date_modified | dateModified | updated_at` → `dateModified`
- `image | image_id | cover | featured_image` → `image` (resolved via media library; falls back to direct URL if the field is a `http(s)` string)
- The site name auto-fills `publisher` as `{@type: "Organization", name: site.Name}`.

Use this for blog posts, case studies, news, long-form articles. The eval engine surfaces a warning if an Article-typed Collection has no `headline` or `description` because both are practically required for AI-search citations.

### Product

- `description | summary` → `description`
- `brand` → `brand` (wrapped as `{@type: "Brand", name}`)
- `image | image_id | main_image` → `image`
- `sku` → `sku`
- `price` → `offers.price` (any numeric field)
- `currency | price_currency` → `offers.priceCurrency` (defaults to `EUR` when absent)

Auto-emits `offers.availability: "https://schema.org/InStock"`. Out-of-stock is a Sprint 5 concern (commerce). Until then, hide unavailable products at the item level.

### Person

- `job_title | jobTitle | title | role` → `jobTitle`
- `email` → `email`
- `image | image_id | photo | headshot` → `image`
- `url | website | linkedin` → `sameAs` (as a single-element array)

Use for team members, speakers, authors. Don't reuse it for Article author bylines; the Article mapping handles that internally.

### Event

- `start_date | startDate | starts_at` → `startDate`
- `end_date | endDate | ends_at` → `endDate`
- `location | venue` → `location` (wrapped as `{@type: "Place", name}`)
- `image | image_id | cover` → `image`

Both start and end should be RFC 3339 datetimes for Google Rich Results to accept the markup. Use `datetime` field type in your schema; the validator enforces the format.

### JobPosting

- `title | job_title` → `title`
- `description | summary` → `description`
- `date_posted | datePosted | published_at` → `datePosted`
- `location | city` → `jobLocation` (wrapped as `{@type: "Place", address}`)
- The site name auto-fills `hiringOrganization` as `{@type: "Organization", name: site.Name}`.

Note that JobPosting is one of the few schema.org types Google polices aggressively. Stale or expired postings hurt indexing; use `condition: posting_active == "true"` per item to hide expired rows from rendering rather than leaving them published.

### Recipe

- `image | image_id | photo` → `image`
- `ingredients` → `recipeIngredient` (must be an array)
- `instructions` → `recipeInstructions` (must be an array)

Use field type `json` if you want structured ingredient or instruction objects beyond strings. The builder passes the array through verbatim.

### FAQPage

- `question` → `mainEntity[0].name`
- `answer` → `mainEntity[0].acceptedAnswer.text`

Each item becomes one Q&A pair. To render multiple Q&As as one structured FAQ, group them on a single page using a `collection_list` block; the JSON-LD per-item still emits a single Q&A, but Google indexes them collectively if the page is the FAQ index.

### Review

- `reviewed_item | item_name | subject` → `itemReviewed` (wrapped as `{@type: "Thing", name}`)
- `rating | score` → `reviewRating.ratingValue`
- `best_rating | max_rating` → `reviewRating.bestRating` (defaults to 5)
- `author | reviewer` → `author` (wrapped as `{@type: "Person", name}`)
- `review_body | body | content` → `reviewBody`

Reviews are typically attached to a parent (Product, Place). Sprint 4 emits the Review independently; the parent linkage is a Sprint 5 concern when reference fields land.

## What the eval engine validates

Three checks fire on Collection-rendered pages. They show up under the `seo` category, section `collections`, alongside the per-page on-page checks.

1. **Collection JSON-LD present**. Every item detail page must have a `<script type="application/ld+json">` with non-empty `@type`. Severity: warning. Recommendation when failing: set `settings.schema_org_type` on the Collection.
2. **Collection hreflang alternates**. When the same item slug exists in two or more locales, every page must emit `<link rel="alternate" hreflang>` for each variant. Severity: warning. Recommendation when failing: confirm `seo.hreflang_strategy = path` and that the locale items share the same slug.
3. **Collection internal links**. Every Collection index page must link to at least one item. Severity: info. This catches "I created a Collection but published zero items" cases that would otherwise ship a dead link.

All three short-circuit when the site has no Collection-rendered pages, so single-page-builder sites never see a fail.

## Common mistakes

- **Picking `Article` for everything**. Article is for editorial content. Products are Product, team members are Person, events are Event. The wrong type means Google does not show rich results for that page even if the markup is valid.
- **Using a `text` field for dates**. Dates must be ISO-8601 (date) or RFC 3339 (datetime). The validator rejects malformed dates; the JSON-LD builder still passes them through unchanged, which means malformed dates land in your structured data and Google complains.
- **Hard-coding image URLs in `text` fields instead of `image` fields**. The mapping looks at `image_id`-shaped strings (24-char hex) and falls back to direct URLs. Putting the URL in a generic text field means the JSON-LD emits no `image` property at all.
- **Putting markup in fields the JSON-LD reads**. `description` is plain prose. If the field is `richtext`, the renderer escapes the HTML before embedding in the script tag, but the surrounding markup tokens leak into the description Google indexes. Keep `description` simple text; use `richtext` for the rendered body field, not the meta description equivalent.
