# i18n authoring

Atomic Site supports multi-language sites with three strategies: path-based (`/en/about/`), subdomain-based (`en.example.com/about/`), or off (single language). Pick once per site through `general.hreflang_strategy`; the builder enforces the choice everywhere.

## Default language and additional languages

`general.default_lang` is the canonical language code (e.g., `sv` for a Swedish site). It is the language used at the root path, with no prefix.

`general.additional_langs` is a comma-separated list (e.g., `en,de,fr`). Each becomes a locale prefix in path mode (`/en/`, `/de/`) or a subdomain (`en.example.com`) depending on strategy.

## Path strategy (most common)

Default language at the root, additional languages at `/<lang>/`:

```
/                /* sv (default) */
/en/             /* English */
/de/             /* German */
```

Page slugs are language-agnostic in the database (`pages.slug = "about"`), and the builder generates the localised file path. Translations are separate page rows with the same slug and a `lang` column.

## Hreflang trailing-slash rule

Every locale URL emits with a trailing slash, including alternates and the canonical. A self-referencing hreflang of `/en` (no slash) when the canonical is `/en/` (slash) fails the SEO eval. The fix lives in `internal/builder/i18n.go::buildLocalePath`; the agent does not modify it but must keep authored canonical overrides consistent.

## Subdomain strategy

Each language lives on its own host:

```
example.com           /* sv */
en.example.com        /* English */
de.example.com        /* German */
```

DNS for each subdomain points at the same edge. The builder writes the right `link rel="alternate"` cross-references. Pick subdomain mode only when SEO requires distinct domains per language; path mode is simpler operationally.

## Off (single language)

`hreflang_strategy = off` skips alternates entirely. Use it for single-language sites; do not leave it set when adding a second language, because the builder will not emit alternates and Google will not learn about the translation.

## Default OG and meta templates

`settings.seo.meta_title_template` and `meta_description_template` accept tokens like `{{title}}`, `{{site_name}}`, `{{lang}}`. The template applies per page after the agent fills `meta_title` and `meta_description` per page. Localise the template per language by writing one row per `lang` in the `settings` table.

## When you create a page in a non-default language

`create_page` accepts an optional `lang` field. If the site has additional langs configured, the agent should:

1. Create the default-language page first
2. Create the translation with the same `slug` and the target `lang`
3. Verify both appear under `atomicsite://site/structure` before triggering build

The builder skips emitting alternates that point at unpublished pages, so a half-translated site degrades gracefully.

## What the eval checks

- Self-referencing hreflang per locale with trailing slash
- Every locale lists every other locale via `<link rel="alternate">`
- `x-default` alternate points at the default-language home
- HTML `lang` attribute matches the page's locale, not the site's default

If any of these fail, the SEO eval grades drop sharply. Fix at the settings level (strategy, default lang) before fixing per-page.
