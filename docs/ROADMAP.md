# Atomic Site → WordPress / Webflow Replacement Roadmap

Last updated: 2026-05-22

## Goal

Position Atomic Site as the credible WordPress / Webflow replacement for serious EU-sovereign buyers. The wedge stays unchanged ("the agent ships the whole site without an agency"); this roadmap closes the table-stakes gaps that block real RFPs.

## Wedge re-frame (not negotiable)

- Production output stays pure Astro SSG. Zero JS framework at runtime. Zero third-party CDN. Self-hosted woff2 only.
- The agent loop is the killer feature, not a visual canvas editor. We do NOT chase Webflow's click-and-drag designer.
- Plugin ecosystem path is MCP-as-Apps, not a WordPress-style PHP plugin host.

## Out of scope

- Webflow-style visual canvas editor (defeats SSG moat).
- Self-hosted PHP plugin runtime (wrong shape for agent-first product).
- Real-time multi-cursor collab (Webflow has it sort of; not a deal-breaker for the EU SMB segment).
- Comments / threaded discussions (most modern sites use Discord/Disqus, low ROI here).

## Sequencing - 5 sprints

Each sprint ships a complete, deployable, used-end-to-end feature. No half-finished implementations. Definition of done per sprint is at the bottom.

### Sprint 1 - Version history + restore (foundation)

**Why first**: every other sprint adds editing surface area. Operators edit more aggressively when they trust undo. Webflow has it native; WordPress has revisions on every post. AS feels like a prototype without it.

**Scope (v1: pages + blocks only)**:
- `entity_revisions` table: id, site_id, entity_type, entity_id, version_number, snapshot_json, change_summary, created_by, created_at. Index on (site_id, entity_type, entity_id, version_number DESC).
- Retention: 50 revisions per entity per site, prune older on insert.
- Backend: `internal/revisions/recorder.go` with `Record(ctx, store, params)` writing a snapshot of the pre-update row. Hooks: `PageHandler.Update`, `BlockHandler.Update`, `BlockHandler.Delete`.
- REST: `GET /sites/{id}/{entity_type}/{entity_id}/revisions`, `GET .../revisions/{version}`, `POST .../revisions/{version}/restore`. SiteAccessMiddleware.
- MCP tools: `list_revisions(entity_type, entity_id)`, `get_revision(entity_type, entity_id, version)`, `restore_revision(entity_type, entity_id, version)`. RequiresWrite on restore.
- Frontend: `api/revisions.ts`, `components/revisions/RevisionDrawer.svelte` slide-in panel with timestamp + author + restore button. Hook into pages list + block editor.
- Restore semantics: NOT destructive. Restoring v3 of a page creates v4 with v3's snapshot applied. History is append-only.
- Tests: Go handler tests (list, get, restore, cross-tenant), MCP wire tests, Playwright e2e (edit → drawer shows revision → restore → revert verified).

**Files**: ~14 new, ~3 modified.

### Sprint 2 - E-commerce v1 (vertical-opening)

**Why second**: the single biggest commercial gap. Mollie code is already half-wired (`internal/cloud/mollie/checkout.go`). Half of agency RFPs include WooCommerce; AS is cut out of B2C / retail / course sellers / SaaS-with-self-serve without checkout.

**Scope (v1: physical or digital products, single currency per site, no shipping zones complexity)**:
- Schema: `products`, `product_variants`, `orders`, `order_items`, `inventory`, `discount_codes`. site_id FK + ON DELETE CASCADE everywhere.
- Mollie finish: webhook idempotency hardening (existing `internal/cloud/mollie/`), per-tenant API key in site_settings (encrypted via existing Shield), order state machine (pending → paid → fulfilled → refunded).
- Cart: client-side state in localStorage + server-validated checkout payload (price + inventory re-check at checkout, not trusted from cart).
- Public blocks: `product_grid`, `product_detail`, `cart_drawer`, `checkout_form`. All Astro SSG with hydrated TS islands for cart/checkout state. Zero React/Vue.
- Admin UI: Products tab + Orders tab + Discount codes tab in `/sites/[siteID]/store/*` routes.
- Agent tools: `list_products`, `create_product`, `update_product`, `list_orders`, `update_order_status`, `set_inventory`, `create_discount_code`. RequiresWrite on mutators.
- Tax: VAT-inclusive pricing per EU norms; per-country VAT rates table; OSS reporting export.
- Retention + GDPR: orders retained per `data_retention_settings`; PII fields tokenized via Shield.
- Tests: full handler + MCP wire test coverage; Mollie webhook fixtures; Playwright e2e through a real checkout to Mollie test mode.

**Files**: ~35 new, ~5 modified.

**Carve-out for Sprint 2.5 if scope blows up**: shipping zones, multi-currency, subscriptions. Single-currency physical/digital products + one-time Mollie payment is the v1 ship.

### Sprint 3 - Multilingual v1 (EU table-stakes)

**Why third**: Sweden / EU market. Webflow Localization shipped 2024. AS already has builder plumbing (`internal/builder/i18n.go`) but no operator UX or agent flow.

**Scope**:
- Schema: `page_locales(page_id, locale, slug, title, meta_title, meta_description, status, updated_at)`, `block_locales(block_id, locale, data_json, updated_at)`, `collection_item_locales(...)`, `site_locales(site_id, locale, is_default, sort_order)`.
- Routing: per-locale URL prefix (`/en/`, `/sv/`) or subdomain (operator choice in settings). Default locale = no prefix.
- Hreflang: auto-generated `<link rel="alternate" hreflang="...">` on every page based on `page_locales` row presence; sitemap.xml per-locale entries.
- Admin UI: locale switcher in the page editor header; "Translate to X" CTA that opens an agent clarification with the source text pre-filled.
- Agent tool: `translate_entity(entity_type, entity_id, target_locale, preserve_voice=true)` calling a translation pass through the existing model surface; returns suggested copy for human review.
- Public site locale switcher block.
- SEO: per-locale meta + canonical + hreflang fully wired; tested via existing migration verify-live path.
- Tests: handler + builder + hreflang generation tests; Playwright e2e through a 3-locale site.

**Files**: ~25 new, ~10 modified (existing builder + page handler).

### Sprint 4 - MCP-as-Apps platform (the ecosystem bet)

**Why fourth**: this is where AS wins, not catches up. Agent-key + MCP is already in product; productize it as a discoverable, installable apps surface so third parties ship MCP integrations and any AS site can wire them in with a tool call.

**Scope**:
- `apps` table: app_id, name, description, mcp_url, icon_url, category, publisher, version. site_installs join table.
- Discovery page: `/apps` (cross-site) listing curated MCP servers (Stripe, HubSpot, Calendly, Brevo, Algolia, OpenAI, Slack, Discord).
- Per-site Apps tab in `/sites/[siteID]/apps`: list installed, browse marketplace, install + revoke.
- Install flow: site picks an app → enters credentials → AS stores tokens via Shield → exposes them to the agent under that app's MCP namespace.
- Agent context: `list_installed_apps` MCP tool returns each install's available tools so the agent can plan multi-app flows.
- Publisher console: simple form to submit an app (initially curated whitelist; later open).
- Docs page: "Build an Atomic Site App" with examples (Astro starter MCP server template).
- Tests: install/revoke handler tests; agent tool wire test verifying namespacing.

**Files**: ~20 new, ~3 modified.

### Sprint 5 - Quick wins batch (polish + parity tail)

Bundle of small features that don't justify standalone sprints but matter for parity. Each is 0.5-1 day.

- **On-site search**: Pagefind generated at build time + `/search` page + `SiteSearchBlock`.
- **Image focal-point**: per-Medium `focal_x`, `focal_y` field + UI to pick + variants use `object-position` per focal point.
- **Conditional form logic**: `forms.fields_json` gains `visible_if: { field, equals }` schema; runtime hides/shows fields client-side.
- **Multi-step forms**: `forms.fields_json` supports a `step: number` grouping; renders a multi-step component.
- **CWV ingestion**: collect web-vitals from production via a beacon endpoint; show last-7-day p75 LCP/INP/CLS in the site dashboard.
- **Newsletter integration**: Brevo + Mailerlite tool that captures form submissions to a list (via Sprint 4's apps platform if shipped).

**Files**: ~30 new across the 6 items.

## Definition of done - per sprint

A sprint is DONE only when ALL of these are true:

1. `bunx tsc --noEmit` zero errors.
2. `bun run build` clean.
3. `go build ./...` clean.
4. `go test ./... -count=1` clean.
5. Em-dash sweep on every new file passes (no `-` characters).
6. Playwright e2e spec covers the happy path + at least one edge case.
7. Pushed via `git psync`. the deploy pipeline deploy verified live: container restart timestamp newer than push timestamp.
8. Live smoke: log in to production, exercise the feature end-to-end, verify the data round-trip survives a page refresh.
9. Hive entity entry refined into a long-form retrospective (no `auto-stub` lines left for the sprint).
10. Graphify rebuilt (automatic via psync post-push).

## Estimated calendar

Assuming focused execution and no surprises:

- Sprint 1 (version history): 1 session.
- Sprint 2 (e-commerce): 2-3 sessions.
- Sprint 3 (multilingual): 2 sessions.
- Sprint 4 (apps platform): 1-2 sessions.
- Sprint 5 (quick wins): 1 session.

Total: 7-9 sessions across 1-2 weeks of focused work.

## Sequencing rationale (TL;DR)

- Version history first because it underpins trust for the next four sprints.
- E-commerce next because the half-wired Mollie code is rotting and the commercial unlock is largest.
- Multilingual third because EU market expects it and we already have builder plumbing.
- Apps platform fourth because it's the moat play and requires the earlier sprints to feel productized.
- Quick wins last because they're polish, not blockers, and let us bundle small items efficiently.
