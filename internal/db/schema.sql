-- AtomicSite database schema (SQLite)

CREATE TABLE IF NOT EXISTS users (
    id             TEXT PRIMARY KEY,
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    name           TEXT NOT NULL DEFAULT '',
    role           TEXT NOT NULL DEFAULT 'admin',
    token_version  INTEGER NOT NULL DEFAULT 1,
    -- TOTP MFA (Sprint 3, 2026-05-04). totp_secret is the base32-
    -- encoded 20-byte random shared with the authenticator app;
    -- empty means not enrolled. totp_enrolled_at is set when the
    -- user verifies their first code (locks in enrollment).
    -- totp_recovery_json is a JSON array of bcrypt-hashed single-
    -- use recovery codes shown ONCE at enrollment.
    totp_secret         TEXT NOT NULL DEFAULT '',
    totp_enrolled_at    TEXT NOT NULL DEFAULT '',
    totp_recovery_json  TEXT NOT NULL DEFAULT '[]',
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- One-time signup tokens minted by an admin so a teammate can self-onboard.
-- Tier-1 invite (no email sending): admin copies the URL, shares it offline,
-- invitee opens /signup/{token} and sets a password to activate the account.
CREATE TABLE IF NOT EXISTS invites (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'editor',
    token       TEXT NOT NULL UNIQUE,
    created_by  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at  TEXT NOT NULL,
    used_at     TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_invites_token ON invites(token);
CREATE INDEX IF NOT EXISTS idx_invites_email ON invites(email);

-- site_members links users to sites for multi-tenant authorization. Without
-- this table every authenticated user could read, update, delete any site
-- by enumerating site IDs (audit finding C1, fixed 2026-05-01). The
-- RequireSiteAccess middleware checks membership on every /api/sites/{id}/*
-- route. Users with role='admin' bypass the check (preserves the legacy
-- single-workspace-admin behaviour).
--
-- Roles inside a site: owner can do anything (including delete the site +
-- manage members), editor can change content but not the membership list.
-- The seed admin path auto-grants ownership on every existing site at the
-- migration boundary so single-admin deployments keep working without a
-- manual data migration.
CREATE TABLE IF NOT EXISTS site_members (
    site_id    TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'editor',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (site_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_site_members_user ON site_members(user_id);
CREATE INDEX IF NOT EXISTS idx_site_members_site ON site_members(site_id);

CREATE TABLE IF NOT EXISTS sites (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    slug             TEXT NOT NULL UNIQUE,
    domain           TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'draft',
    -- Branding
    primary_color    TEXT NOT NULL DEFAULT '#D4AF37',
    secondary_color  TEXT NOT NULL DEFAULT '#935FA7',
    -- Phase 12.9: extra color slots so the schema covers what the
    -- preview already implies. Surface = card panels (lighter than bg).
    -- Border = hairlines around cards / dividers. Muted = de-emphasised
    -- text (footer, captions). Accent = link / focus ring colour
    -- distinct from primary CTA. OnPrimary = text on primary buttons
    -- (white for dark primaries, black for light ones).
    surface_color    TEXT NOT NULL DEFAULT '#FFFFFF',
    border_color     TEXT NOT NULL DEFAULT '#E5E7EB',
    muted_color      TEXT NOT NULL DEFAULT '#6B7280',
    accent_color     TEXT NOT NULL DEFAULT '',
    on_primary_color TEXT NOT NULL DEFAULT '#FFFFFF',
    bg_color         TEXT NOT NULL DEFAULT '#FFFFFF',
    text_color       TEXT NOT NULL DEFAULT '#1A1A1A',
    font_heading     TEXT NOT NULL DEFAULT 'Inter',
    font_body        TEXT NOT NULL DEFAULT 'Inter',
    -- SEO defaults
    meta_title       TEXT NOT NULL DEFAULT '',
    meta_description TEXT NOT NULL DEFAULT '',
    og_image_id      TEXT NOT NULL DEFAULT '',
    favicon_id       TEXT NOT NULL DEFAULT '',
    -- Analytics
    ga4_id           TEXT NOT NULL DEFAULT '',
    umami_id         TEXT NOT NULL DEFAULT '',
    umami_url        TEXT NOT NULL DEFAULT '',
    cookieproof_domain TEXT NOT NULL DEFAULT '',
    -- Build config
    lang             TEXT NOT NULL DEFAULT 'en',
    last_build_at    TEXT NOT NULL DEFAULT '',
    last_build_status TEXT NOT NULL DEFAULT 'none',
    last_build_error TEXT NOT NULL DEFAULT '',
    last_deploy_at   TEXT NOT NULL DEFAULT '',
    -- Per-tenant quotas (Sprint 3, 2026-05-04). Defaults give every
    -- site 1 GiB of media storage and 600 build minutes / 30-day window
    -- without any operator action; admin can raise per-site via
    -- PATCH /api/admin/sites/{id}/quota. quota_overage_blocked = 1
    -- means QuotaMiddleware short-circuits 429 on overage; 0 = warn
    -- via X-Quota-Warning header but allow the request.
    storage_quota_bytes   INTEGER NOT NULL DEFAULT 1073741824,
    build_minutes_quota   INTEGER NOT NULL DEFAULT 600,
    quota_overage_blocked INTEGER NOT NULL DEFAULT 1,
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_sites_slug ON sites(slug);

CREATE TABLE IF NOT EXISTS pages (
    id               TEXT PRIMARY KEY,
    site_id          TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    title            TEXT NOT NULL,
    slug             TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'draft',
    meta_title       TEXT NOT NULL DEFAULT '',
    meta_description TEXT NOT NULL DEFAULT '',
    og_image_id      TEXT NOT NULL DEFAULT '',
    layout           TEXT NOT NULL DEFAULT 'default',
    sort_order       INTEGER NOT NULL DEFAULT 0,
    show_in_nav      INTEGER NOT NULL DEFAULT 1,
    nav_label        TEXT NOT NULL DEFAULT '',
    no_index         INTEGER NOT NULL DEFAULT 0,
    canonical_url    TEXT NOT NULL DEFAULT '',
    hide_global_blocks INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, slug)
);
CREATE INDEX IF NOT EXISTS idx_pages_site ON pages(site_id);

CREATE TABLE IF NOT EXISTS blocks (
    id               TEXT PRIMARY KEY,
    page_id          TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    block_type       TEXT NOT NULL,
    name             TEXT NOT NULL DEFAULT '',
    sort_order       INTEGER NOT NULL DEFAULT 0,
    data_json        TEXT NOT NULL DEFAULT '{}',
    style_json       TEXT NOT NULL DEFAULT '{}',
    is_visible       INTEGER NOT NULL DEFAULT 1,
    template_version INTEGER NOT NULL DEFAULT 1,
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_blocks_page ON blocks(page_id);
CREATE INDEX IF NOT EXISTS idx_blocks_page_order ON blocks(page_id, sort_order);

CREATE TABLE IF NOT EXISTS global_blocks (
    id               TEXT PRIMARY KEY,
    site_id          TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    slot             TEXT NOT NULL,
    block_type       TEXT NOT NULL,
    data_json        TEXT NOT NULL DEFAULT '{}',
    style_json       TEXT NOT NULL DEFAULT '{}',
    is_active        INTEGER NOT NULL DEFAULT 1,
    sort_order       INTEGER NOT NULL DEFAULT 0,
    template_version INTEGER NOT NULL DEFAULT 1,
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, slot, sort_order)
);
CREATE INDEX IF NOT EXISTS idx_global_blocks_site ON global_blocks(site_id);

CREATE TABLE IF NOT EXISTS media (
    id             TEXT PRIMARY KEY,
    site_id        TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    filename       TEXT NOT NULL,
    alt_text       TEXT NOT NULL DEFAULT '',
    mime_type      TEXT NOT NULL,
    file_size      INTEGER NOT NULL,
    width          INTEGER NOT NULL DEFAULT 0,
    height         INTEGER NOT NULL DEFAULT 0,
    blurhash       TEXT NOT NULL DEFAULT '',
    variants_json  TEXT NOT NULL DEFAULT '[]',
    original_path  TEXT NOT NULL,
    folder         TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_media_site ON media(site_id);
CREATE INDEX IF NOT EXISTS idx_media_site_folder ON media(site_id, folder);

CREATE TABLE IF NOT EXISTS media_folders (
    site_id    TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    is_system  INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (site_id, name)
);

CREATE TABLE IF NOT EXISTS deployments (
    id            TEXT PRIMARY KEY,
    site_id       TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'pending',
    build_log     TEXT NOT NULL DEFAULT '',
    deploy_target TEXT NOT NULL DEFAULT 'local',
    deploy_config TEXT NOT NULL DEFAULT '{}',
    pages_built   INTEGER NOT NULL DEFAULT 0,
    duration_ms   INTEGER NOT NULL DEFAULT 0,
    error         TEXT NOT NULL DEFAULT '',
    target_id     TEXT NOT NULL DEFAULT '',
    deploy_url    TEXT NOT NULL DEFAULT '',
    deployed_at   TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_deployments_site ON deployments(site_id);

CREATE TABLE IF NOT EXISTS deploy_targets (
  id          TEXT PRIMARY KEY,
  site_id     TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  kind        TEXT NOT NULL CHECK (kind IN ('dockyard','rsync','local')),
  config_json TEXT NOT NULL DEFAULT '{}',
  is_default  INTEGER NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(site_id, name)
);
CREATE INDEX IF NOT EXISTS idx_deploy_targets_site ON deploy_targets(site_id);

CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ============================================================
-- Atomicsite AI Website Builder: Agent API + Evaluation Engine
-- ============================================================

-- Business profile (auto-fills legal pages, schema, security.txt)
CREATE TABLE IF NOT EXISTS site_profiles (
    id                TEXT PRIMARY KEY,
    site_id           TEXT NOT NULL UNIQUE REFERENCES sites(id) ON DELETE CASCADE,
    business_name     TEXT NOT NULL DEFAULT '',
    registration_nr   TEXT NOT NULL DEFAULT '',
    country           TEXT NOT NULL DEFAULT 'SE',
    contact_email     TEXT NOT NULL DEFAULT '',
    contact_phone     TEXT NOT NULL DEFAULT '',
    privacy_email     TEXT NOT NULL DEFAULT '',
    security_email    TEXT NOT NULL DEFAULT '',
    address_line1     TEXT NOT NULL DEFAULT '',
    address_line2     TEXT NOT NULL DEFAULT '',
    city              TEXT NOT NULL DEFAULT '',
    postal_code       TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at        TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Site architecture (one-pager, soft-silo, hard-silo)
CREATE TABLE IF NOT EXISTS site_architecture (
    id              TEXT PRIMARY KEY,
    site_id         TEXT NOT NULL UNIQUE REFERENCES sites(id) ON DELETE CASCADE,
    structure_type  TEXT NOT NULL DEFAULT 'soft-silo',
    max_depth       INTEGER NOT NULL DEFAULT 3,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Silos/sections within a site architecture. silo_type allows hybrid
-- structures: 'inherit' uses site_architecture.structure_type, 'soft' allows
-- cross-silo links, 'hard' enforces strict topical isolation.
CREATE TABLE IF NOT EXISTS site_silos (
    id          TEXT PRIMARY KEY,
    site_id     TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug_prefix TEXT NOT NULL,
    silo_type   TEXT NOT NULL DEFAULT 'inherit' CHECK (silo_type IN ('inherit','soft','hard')),
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, slug_prefix)
);
CREATE INDEX IF NOT EXISTS idx_site_silos_site ON site_silos(site_id);

-- Agent API keys (scoped per site with capability declarations)
CREATE TABLE IF NOT EXISTS agent_keys (
    id           TEXT PRIMARY KEY,
    site_id      TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    key_hash     TEXT NOT NULL,
    capabilities TEXT NOT NULL DEFAULT '["read","write"]',
    is_active    INTEGER NOT NULL DEFAULT 1,
    last_used_at TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_agent_keys_site ON agent_keys(site_id);

-- Knowledgebase entries (per-site brand/voice/technical rules for AI context)
CREATE TABLE IF NOT EXISTS knowledgebase_entries (
    id         TEXT PRIMARY KEY,
    site_id    TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    category   TEXT NOT NULL,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    is_active  INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_knowledgebase_site ON knowledgebase_entries(site_id);

-- Guardrail rules (validated on every agent write operation)
CREATE TABLE IF NOT EXISTS guardrail_rules (
    id         TEXT PRIMARY KEY,
    site_id    TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    rule_type  TEXT NOT NULL,
    target     TEXT NOT NULL,
    value      TEXT NOT NULL,
    severity   TEXT NOT NULL DEFAULT 'error',
    is_active  INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_guardrail_rules_site ON guardrail_rules(site_id);

-- Reusable components with typed props (JSON Schema)
CREATE TABLE IF NOT EXISTS components (
    id           TEXT PRIMARY KEY,
    site_id      TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    category     TEXT NOT NULL DEFAULT 'section',
    template     TEXT NOT NULL,
    props_schema TEXT NOT NULL DEFAULT '{}',
    css_classes  TEXT NOT NULL DEFAULT '[]',
    usage_note   TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, name)
);
CREATE INDEX IF NOT EXISTS idx_components_site ON components(site_id);

-- Global CSS classes (first-class entities, not embedded in blocks)
CREATE TABLE IF NOT EXISTS css_classes (
    id         TEXT PRIMARY KEY,
    site_id    TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    category   TEXT NOT NULL DEFAULT 'utility',
    css        TEXT NOT NULL,
    usage_note TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, name)
);
CREATE INDEX IF NOT EXISTS idx_css_classes_site ON css_classes(site_id);

-- Per-site settings (security headers, robots, nginx, analytics, SEO)
CREATE TABLE IF NOT EXISTS site_settings (
    id         TEXT PRIMARY KEY,
    site_id    TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    category   TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, category, key)
);
CREATE INDEX IF NOT EXISTS idx_site_settings_site ON site_settings(site_id);

-- Allowlisted external domains (feeds CSP generation + guardrails). The kind
-- column routes the domain to the right CSP directive, so a single table
-- covers scripts (Stripe.js), iframes (cal.com), images (Cloudinary CDN),
-- media (Vimeo), and connect-only API hosts.
--
-- kind values:
--   script  -> script-src + connect-src (default; backwards-compatible)
--   frame   -> frame-src (iframe embeds like cal.com / YouTube / Stripe Checkout)
--   image   -> img-src
--   media   -> media-src
--   connect -> connect-src
--   all     -> every directive above
CREATE TABLE IF NOT EXISTS allowed_scripts (
    id         TEXT PRIMARY KEY,
    site_id    TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    domain     TEXT NOT NULL,
    purpose    TEXT NOT NULL DEFAULT '',
    kind       TEXT NOT NULL DEFAULT 'script',
    is_active  INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, domain, kind)
);
CREATE INDEX IF NOT EXISTS idx_allowed_scripts_site ON allowed_scripts(site_id);

-- 301/302 redirects (auto-created on slug changes, manual management)
CREATE TABLE IF NOT EXISTS redirects (
    id          TEXT PRIMARY KEY,
    site_id     TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    from_path   TEXT NOT NULL,
    to_path     TEXT NOT NULL,
    status_code INTEGER NOT NULL DEFAULT 301,
    is_auto     INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, from_path)
);
CREATE INDEX IF NOT EXISTS idx_redirects_site ON redirects(site_id);

-- Form definitions (contact, lead capture, newsletter)
CREATE TABLE IF NOT EXISTS forms (
    id            TEXT PRIMARY KEY,
    site_id       TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    fields_json   TEXT NOT NULL DEFAULT '[]',
    action        TEXT NOT NULL DEFAULT 'store',
    action_config TEXT NOT NULL DEFAULT '{}',
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_forms_site ON forms(site_id);

-- Form submissions
CREATE TABLE IF NOT EXISTS form_submissions (
    id         TEXT PRIMARY KEY,
    form_id    TEXT NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    site_id    TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    data_json  TEXT NOT NULL DEFAULT '{}',
    ip_hash    TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_form_submissions_form ON form_submissions(form_id);
CREATE INDEX IF NOT EXISTS idx_form_submissions_site ON form_submissions(site_id);

-- Build evaluation results (130+ checks from site-inspector)
CREATE TABLE IF NOT EXISTS evaluations (
    id          TEXT PRIMARY KEY,
    build_id    TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    site_id     TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    category    TEXT NOT NULL,
    score       INTEGER NOT NULL,
    max_score   INTEGER NOT NULL,
    grade       TEXT NOT NULL,
    checks_json TEXT NOT NULL DEFAULT '[]',
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_evaluations_build ON evaluations(build_id);
CREATE INDEX IF NOT EXISTS idx_evaluations_site ON evaluations(site_id);

-- ============================================================
-- Analytics moat: server-side visit tracking + consent stitching
-- ============================================================

-- Individual visit events parsed from Nginx JSON access logs.
-- Enrichment columns (browser/os/device/country/utm) are filled by the
-- analytics parser at ingest time; raw IP and full UA are NEVER stored.
CREATE TABLE IF NOT EXISTS visit_events (
    id           TEXT PRIMARY KEY,
    site_id      TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    fingerprint  TEXT NOT NULL,
    session_id   TEXT NOT NULL DEFAULT '',
    path         TEXT NOT NULL,
    referer      TEXT NOT NULL DEFAULT '',
    status       INTEGER NOT NULL DEFAULT 200,
    ms           INTEGER NOT NULL DEFAULT 0,
    ts           TEXT NOT NULL DEFAULT (datetime('now')),
    -- Enrichment (Phase 12.5)
    browser      TEXT NOT NULL DEFAULT '',  -- e.g. "Chrome", "Safari", "Firefox"
    os           TEXT NOT NULL DEFAULT '',  -- e.g. "macOS", "Windows", "iOS"
    device       TEXT NOT NULL DEFAULT '',  -- "desktop" | "mobile" | "tablet" | "bot"
    country      TEXT NOT NULL DEFAULT '',  -- ISO 3166-1 alpha-2, sourced from CF-IPCountry header
    lang         TEXT NOT NULL DEFAULT '',  -- primary language tag from Accept-Language ("en", "sv-SE")
    utm_source   TEXT NOT NULL DEFAULT '',
    utm_medium   TEXT NOT NULL DEFAULT '',
    utm_campaign TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_visit_events_site_ts ON visit_events(site_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_visit_events_fingerprint ON visit_events(site_id, fingerprint);
CREATE INDEX IF NOT EXISTS idx_visit_events_session ON visit_events(site_id, session_id);
CREATE INDEX IF NOT EXISTS idx_visit_events_country ON visit_events(site_id, country);
CREATE INDEX IF NOT EXISTS idx_visit_events_browser ON visit_events(site_id, browser);
CREATE INDEX IF NOT EXISTS idx_visit_events_utm ON visit_events(site_id, utm_source);

-- Aggregated visit sessions (one per site/fingerprint), stitched on consent
CREATE TABLE IF NOT EXISTS visit_sessions (
    id              TEXT PRIMARY KEY,
    site_id         TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    fingerprint     TEXT NOT NULL,
    visitor_id      TEXT NOT NULL DEFAULT '',
    email           TEXT NOT NULL DEFAULT '',
    consent_method  TEXT NOT NULL DEFAULT '',
    consent_categories_json TEXT NOT NULL DEFAULT '{}',
    started_at      TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen_at    TEXT NOT NULL DEFAULT (datetime('now')),
    page_count      INTEGER NOT NULL DEFAULT 0,
    identified_at   TEXT NOT NULL DEFAULT '',
    -- Bidirectional CRM personalization (Phase 18). The CRM pushes a JSON
    -- blob of opaque key/value pairs (lifecycle_stage, lead_score,
    -- last_topic, name, email, ...). identity_confirmed_at is stamped by
    -- the CRM whenever it has fresh confirmation that the fingerprint is
    -- the right person; the read endpoint scrubs identified-tier fields
    -- when this is older than analytics.identity_max_age_days. expires_at
    -- lets the CRM tell us "this lead_score is good for 24h".
    metadata_json           TEXT NOT NULL DEFAULT '{}',
    metadata_expires_at     TEXT NOT NULL DEFAULT '',
    identity_confirmed_at   TEXT NOT NULL DEFAULT '',
    UNIQUE(site_id, fingerprint)
);
CREATE INDEX IF NOT EXISTS idx_visit_sessions_site_seen ON visit_sessions(site_id, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_visit_sessions_identified ON visit_sessions(site_id, identified_at) WHERE identified_at != '';
CREATE INDEX IF NOT EXISTS idx_visit_sessions_identity_conf ON visit_sessions(site_id, identity_confirmed_at) WHERE identity_confirmed_at != '';

-- Client-side engagement beacon (Phase 12.6). The public site embeds a tiny
-- inline JS beacon (see internal/builder/layouts.go). On visibilitychange
-- to "hidden" or pagehide it sendBeacon's a payload to /t/engagement with
-- screen + viewport + prefers-color-scheme + time-on-page + max-scroll
-- depth. Consent-gated: only fires after CookieProof emits consent:init
-- with analytics=true. Server log tail can't see any of these so the
-- beacon is the only path. Fingerprint is computed server-side at the
-- handler from IP+UA+Accept-Language so it correlates with visit_events.
CREATE TABLE IF NOT EXISTS visit_engagement (
    id                      TEXT PRIMARY KEY,
    site_id                 TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    fingerprint             TEXT NOT NULL,
    path                    TEXT NOT NULL,
    ts                      TEXT NOT NULL DEFAULT (datetime('now')),
    screen_w                INTEGER NOT NULL DEFAULT 0,
    screen_h                INTEGER NOT NULL DEFAULT 0,
    viewport_w              INTEGER NOT NULL DEFAULT 0,
    viewport_h              INTEGER NOT NULL DEFAULT 0,
    prefers_dark            INTEGER NOT NULL DEFAULT 0,
    prefers_reduced_motion  INTEGER NOT NULL DEFAULT 0,
    time_on_page_ms         INTEGER NOT NULL DEFAULT 0,
    max_scroll_pct          INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_visit_engagement_site_ts ON visit_engagement(site_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_visit_engagement_path ON visit_engagement(site_id, path);
CREATE INDEX IF NOT EXISTS idx_visit_engagement_fingerprint ON visit_engagement(site_id, fingerprint);

-- Per-site uploaded fonts (Phase 12.8). Self-hosted woff2 only: best
-- compression, universal browser support since 2020. The file lands on
-- disk at {storage}/fonts/{site_id}/{id}.woff2; this row carries the
-- metadata the Astro Layout needs to emit @font-face and the dropdown
-- needs to render an option.
CREATE TABLE IF NOT EXISTS site_fonts (
    id            TEXT PRIMARY KEY,
    site_id       TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    family_name   TEXT NOT NULL,
    weight        INTEGER NOT NULL DEFAULT 400,
    style         TEXT NOT NULL DEFAULT 'normal',
    file_path     TEXT NOT NULL,
    file_size     INTEGER NOT NULL DEFAULT 0,
    original_name TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_site_fonts_site ON site_fonts(site_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_site_fonts_per_family ON site_fonts(site_id, family_name, weight, style);

-- GitHub design references (Phase 12.8). Site owners paste a public
-- repo URL; atomicsite pre-fetches a small bundle of representative
-- files (package.json, README.md, tailwind.config, app.css, plus 3-5
-- component files) and surfaces the bundle to AI agents via the
-- /api/agent/context endpoint as "design vocabulary the user wants
-- atomicsite to feel like." Read-only pattern reference, not code copy.
CREATE TABLE IF NOT EXISTS design_references (
    id            TEXT PRIMARY KEY,
    site_id       TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    url           TEXT NOT NULL,
    label         TEXT NOT NULL DEFAULT '',
    ref_type      TEXT NOT NULL DEFAULT 'design-system',
    fetched_json  TEXT NOT NULL DEFAULT '{}',
    fetched_at    TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, url)
);
CREATE INDEX IF NOT EXISTS idx_design_references_site ON design_references(site_id);

-- consent_records is the GDPR proof-of-consent log. One row per consent
-- decision (accept/reject/custom/GPC/DNS/do-not-sell), keyed by site_id.
-- Stores enough to prove "this visitor said X at time T from page P" without
-- retaining identifying data: ip is hashed with a daily-rotating salt
-- (consent_salts table) so audit lookups within a day work, but the IP
-- itself never lands on disk.
--
-- This table is the system of record for tenants of Atomic Site after the
-- CookieProof fold-in (2026-04-30). Before that, proofs were posted to
-- consent.example.com; now they live here, same-origin.
CREATE TABLE IF NOT EXISTS consent_records (
    id                TEXT PRIMARY KEY,
    site_id           TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    session_id        TEXT NOT NULL DEFAULT '',
    fingerprint       TEXT NOT NULL DEFAULT '',
    domain            TEXT NOT NULL,
    page_url          TEXT NOT NULL DEFAULT '',
    referrer          TEXT NOT NULL DEFAULT '',
    user_agent        TEXT NOT NULL DEFAULT '',
    ip_hash           TEXT NOT NULL,
    consent_method    TEXT NOT NULL,
    consent_version   INTEGER NOT NULL DEFAULT 1,
    categories_json   TEXT NOT NULL DEFAULT '{}',
    gpc_active        INTEGER NOT NULL DEFAULT 0,
    created_at        INTEGER NOT NULL,
    created_at_iso    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_consent_records_site_created ON consent_records(site_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_consent_records_site_method ON consent_records(site_id, consent_method);
CREATE INDEX IF NOT EXISTS idx_consent_records_session ON consent_records(session_id);
CREATE INDEX IF NOT EXISTS idx_consent_records_site_fingerprint ON consent_records(site_id, fingerprint);

-- password_resets holds short-lived single-use tokens for the
-- forgot-password flow. The user requests a reset by email; we
-- mint a 32-byte hex token, store its sha256 hash, and either
-- email a reset link or surface it via slog (when no mail sender
-- is configured) so the operator can deliver it manually.
--
-- Why store the hash and not the raw token: a DB read leak (or a
-- backup file in the wrong hands) would otherwise expose every
-- live reset link.
--
-- Token lifetime is 30 minutes , long enough for a real user to
-- find the email and click, short enough that a leaked link
-- doesn't outlive its delivery channel. Single-use is enforced
-- by setting used_at on first redemption; subsequent reads of
-- the same token are rejected.
--
-- token_version on the user is bumped on successful reset, which
-- invalidates every active session (including the attacker's, if
-- they had one) per the existing JWT-with-version contract.
CREATE TABLE IF NOT EXISTS password_resets (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TEXT NOT NULL,
    used_at     TEXT NOT NULL DEFAULT '',
    requester_ip TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_password_resets_user ON password_resets(user_id);
CREATE INDEX IF NOT EXISTS idx_password_resets_token_hash ON password_resets(token_hash);

-- schema_versions tracks which point-in-time migrations have been
-- applied to this DB. Each row records when applySchema completed
-- a migration step. Lets the operator reason about whether a hot-
-- restored backup matches the running binary's expected schema
-- without diffing the schema.sql by hand.
CREATE TABLE IF NOT EXISTS schema_versions (
    version    INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now')),
    note       TEXT NOT NULL DEFAULT ''
);

-- audit_log records who-did-what-when for every destructive action
-- across the platform. The handler layer calls a single AuditLog()
-- helper after the underlying mutation succeeds, so the row is the
-- "we did this" entry, not the "we tried this" entry.
--
-- Why a single global table (vs per-resource tables): the consumers
-- (compliance reviews, incident root-cause analysis, GDPR data-export)
-- want a chronological feed across every resource type. JSONB diff in
-- changes_json keeps schema evolution cheap.
--
-- Indexes:
--   idx_audit_log_site_time      -> the per-site activity feed in admin
--   idx_audit_log_actor_time     -> "what has user X been doing"
--   idx_audit_log_resource       -> "what has been done to resource X"
--
-- Retention: not auto-purged. Default retention is forever; operators
-- who need a cap can wire a `general.audit_retention_days` setting
-- and add it to the retention manager (Phase N).
CREATE TABLE IF NOT EXISTS audit_log (
    id              TEXT PRIMARY KEY,
    actor_user_id   TEXT NOT NULL DEFAULT '',
    actor_role      TEXT NOT NULL DEFAULT '',
    actor_ip        TEXT NOT NULL DEFAULT '',
    site_id         TEXT NOT NULL DEFAULT '',
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     TEXT NOT NULL DEFAULT '',
    changes_json    TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_audit_log_site_time ON audit_log(site_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor_time ON audit_log(actor_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_resource ON audit_log(resource_type, resource_id);

-- consent_salts holds daily-rotated salts used to hash IPs in
-- consent_records.ip_hash. One row per UTC day. Old rows are pruned by the
-- retention job (default 30 days) so historical hashes can never be
-- correlated back to an IP.
CREATE TABLE IF NOT EXISTS consent_salts (
    day_utc    TEXT PRIMARY KEY,
    salt       TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- site_domains holds custom hostnames pointing at an atomicsite tenant.
-- One row per (siteID, hostname). Status drives the live-domain pipeline:
--
--   pending     - row created, no DNS check has succeeded yet.
--   verified    - HTTP-01-style verification at /.well-known/atomic-verify
--                 returned the expected token, proving the domain points
--                 at our edge.
--   cert_ready  - certbot HTTP-01 issued a Let's Encrypt cert and the
--                 nginx vhost has been written and reloaded.
--   live        - the most recent verification poll succeeded against
--                 the TLS endpoint. This is the steady-state status.
--   error       - last attempt failed; last_error carries the message
--                 surfaced in the admin UI.
--
-- The verify_token is a single-use random string the admin posts as the
-- value at /.well-known/atomic-verify/{token}. Cert path points at the
-- letsencrypt live dir; rotated by the host's certbot timer. The
-- canonical flag marks the host the build pipeline should treat as the
-- site's primary URL (canonical / OG / sitemap base) when site.domain
-- isn't set explicitly.
CREATE TABLE IF NOT EXISTS site_domains (
    id              TEXT PRIMARY KEY,
    site_id         TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    hostname        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    verify_token    TEXT NOT NULL,
    cert_path       TEXT NOT NULL DEFAULT '',
    last_check_at   TEXT NOT NULL DEFAULT '',
    last_error      TEXT NOT NULL DEFAULT '',
    is_canonical    INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_site_domains_hostname ON site_domains(hostname);
CREATE INDEX IF NOT EXISTS idx_site_domains_site ON site_domains(site_id);
CREATE INDEX IF NOT EXISTS idx_site_domains_status ON site_domains(status);
