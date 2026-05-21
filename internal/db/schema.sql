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
    -- GDPR Article 17 erasure cooling-off window (2026-05-09). Set
    -- when the user requests account deletion; the daily retention
    -- sweep hard-deletes rows older than the cooling-off window.
    -- Empty = not pending.
    deletion_requested_at TEXT NOT NULL DEFAULT '',
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

-- ============================================================
-- Workspaces (Phase 30, Cloud Tier MVP, 2026-05-05).
--
-- A workspace is the billing + ownership boundary that contains N sites.
-- One user can belong to N workspaces (agency tier). The plan column
-- gates feature access (custom_domain, max_sites, white_label, etc.);
-- plan='oss' means unlimited / no Stripe subscription, used by every
-- single-deploy OSS install where the boot bootstrap auto-creates one
-- workspace per user. Cloud signups create workspaces with plan='solo'
-- on a free trial.
--
-- region is reserved for the EU/US split (Phase 31). MVP ships EU-only;
-- 'eu' is the only valid value today.
--
-- stripe_customer_id and stripe_subscription_id are populated by the
-- billing webhook (Phase 30.2). status mirrors the subscription state
-- so cron jobs can suspend builds + serving without going through
-- Stripe on every request.
-- ============================================================
CREATE TABLE IF NOT EXISTS workspaces (
    id                      TEXT PRIMARY KEY,
    name                    TEXT NOT NULL,
    slug                    TEXT NOT NULL UNIQUE,
    plan                    TEXT NOT NULL DEFAULT 'oss',
    region                  TEXT NOT NULL DEFAULT 'eu',
    billing_email           TEXT NOT NULL DEFAULT '',
    stripe_customer_id      TEXT NOT NULL DEFAULT '',
    stripe_subscription_id  TEXT NOT NULL DEFAULT '',
    status                  TEXT NOT NULL DEFAULT 'active',
    trial_ends_at           TEXT NOT NULL DEFAULT '',
    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at              TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);
CREATE INDEX IF NOT EXISTS idx_workspaces_status ON workspaces(status);

-- Per-user role within a workspace. owner = full control + billing,
-- admin = manage sites + members but not billing, member = read + edit
-- assigned sites. The seed admin gets owner on the auto-bootstrap
-- workspace; cloud signups make the registering user the owner.
CREATE TABLE IF NOT EXISTS workspace_members (
    workspace_id  TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role          TEXT NOT NULL DEFAULT 'member',
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (workspace_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user ON workspace_members(user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace ON workspace_members(workspace_id);

-- Workspace-scoped invite tokens. Mirrors the existing invites table
-- shape (used for global admin invites pre-Phase-30) but adds
-- workspace_id so a member who joins via this flow lands inside the
-- right workspace + role. Cloud waitlist signups also mint these.
CREATE TABLE IF NOT EXISTS workspace_invites (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email         TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'member',
    token         TEXT NOT NULL UNIQUE,
    created_by    TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at    TEXT NOT NULL,
    used_at       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_token ON workspace_invites(token);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_workspace ON workspace_invites(workspace_id);

-- ============================================================
-- Billing (Phase 30.2, Cloud Tier MVP, 2026-05-06).
--
-- Provider-agnostic shape so the same tables work for Mollie today
-- and any future provider. Atomicsite ships with Mollie because the
-- EU-sovereign positioning rules out Stripe (US-incorporated, Cloud
-- Act exposure). The brightcrm payment_config pattern was the
-- reference for the provider-enum + opaque external-id design.
--
-- subscriptions.external_id is the Mollie subscription mol_xxxx ID;
-- external_customer_id is the Mollie customer cst_xxxx. status mirrors
-- the Mollie subscription state so daily cron + middleware can refuse
-- builds without a webhook round-trip.
-- ============================================================
CREATE TABLE IF NOT EXISTS subscriptions (
    id                    TEXT PRIMARY KEY,
    workspace_id          TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider              TEXT NOT NULL DEFAULT 'mollie',
    external_id           TEXT NOT NULL DEFAULT '',
    external_customer_id  TEXT NOT NULL DEFAULT '',
    plan                  TEXT NOT NULL,
    status                TEXT NOT NULL DEFAULT 'pending',
    amount_cents          INTEGER NOT NULL DEFAULT 0,
    currency              TEXT NOT NULL DEFAULT 'EUR',
    interval_unit         TEXT NOT NULL DEFAULT 'months',
    interval_count        INTEGER NOT NULL DEFAULT 1,
    current_period_end    TEXT NOT NULL DEFAULT '',
    cancel_at             TEXT NOT NULL DEFAULT '',
    metadata_json         TEXT NOT NULL DEFAULT '{}',
    created_at            TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at            TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_workspace ON subscriptions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_external ON subscriptions(external_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);

-- Webhook idempotency. Every inbound webhook records its provider
-- event id + payload before the side-effect runs. UNIQUE constraint
-- on (provider, external_event_id) makes duplicate deliveries no-op
-- without a per-handler check. 30-day retention sweep drops
-- processed entries via the existing retention.Manager hook.
CREATE TABLE IF NOT EXISTS billing_events (
    id                  TEXT PRIMARY KEY,
    workspace_id        TEXT NOT NULL DEFAULT '',
    provider            TEXT NOT NULL DEFAULT 'mollie',
    external_event_id   TEXT NOT NULL,
    event_type          TEXT NOT NULL DEFAULT '',
    payload_json        TEXT NOT NULL DEFAULT '{}',
    processed_at        TEXT NOT NULL DEFAULT '',
    error               TEXT NOT NULL DEFAULT '',
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(provider, external_event_id)
);
CREATE INDEX IF NOT EXISTS idx_billing_events_workspace ON billing_events(workspace_id);
CREATE INDEX IF NOT EXISTS idx_billing_events_processed ON billing_events(processed_at);

-- ============================================================
-- Waitlist (Phase 30.5, Cloud Tier MVP, 2026-05-06).
--
-- Public-facing capture for the invite-beta signup model. Visitors
-- on the pricing page submit { email, name, use_case, region } via
-- POST /api/waitlist. The owner reviews entries in /admin/waitlist
-- and clicks "Send invite" to mint a workspace_invite token + email
-- the recipient via MailSender.
--
-- Spam mitigation: rate-limited by IP at the handler layer (5/hour
-- per IP). The DB has a composite UNIQUE index on (email) so a
-- single email can only sit in the waitlist once.
-- ============================================================
CREATE TABLE IF NOT EXISTS waitlist (
    id              TEXT PRIMARY KEY,
    email           TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL DEFAULT '',
    use_case        TEXT NOT NULL DEFAULT '',
    region          TEXT NOT NULL DEFAULT 'eu',
    source_ip       TEXT NOT NULL DEFAULT '',
    user_agent      TEXT NOT NULL DEFAULT '',
    invited_at      TEXT NOT NULL DEFAULT '',
    invite_token    TEXT NOT NULL DEFAULT '',
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_waitlist_invited ON waitlist(invited_at);
CREATE INDEX IF NOT EXISTS idx_waitlist_created ON waitlist(created_at);

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
    -- workspace_id (Phase 30, Cloud Tier MVP, 2026-05-05). Every site
    -- belongs to exactly one workspace. OSS installs auto-bootstrap a
    -- single workspace at boot and migrate every existing site into it
    -- (cmd/server/main.go ensureDefaultWorkspace). Empty string is only
    -- valid mid-migration; the bootstrap fills it before serving any
    -- request. Foreign key intentionally omitted on the existing
    -- column (legacy DBs may have NULLs at migration time);
    -- workspace_access middleware verifies membership at every request.
    workspace_id     TEXT NOT NULL DEFAULT '',
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

-- Site migrations: parsed manifest from a source CMS (WordPress/Webflow/Ghost/
-- sitemap-crawl). Stored as JSON so the agent can review the proposed import
-- before any pages or redirects are written. status flow:
--   pending -> ready (after parse) -> applied (after migration_apply) | failed
CREATE TABLE IF NOT EXISTS migrations (
    id            TEXT PRIMARY KEY,
    site_id       TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    source_url    TEXT NOT NULL,
    source_type   TEXT NOT NULL,
    manifest_json TEXT NOT NULL DEFAULT '{}',
    status        TEXT NOT NULL DEFAULT 'pending',
    error         TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_migrations_site ON migrations(site_id);
CREATE INDEX IF NOT EXISTS idx_migrations_status ON migrations(status);

-- Migration verifications (Layer 5 / Sprint 2 of the migration system,
-- 2026-05-06). Per-source-URL post-launch crawl results - the operator
-- ran VerifyLive after flipping DNS and we recorded what each old URL
-- now resolves to on the new atomicsite. ok=1 means the chain ended at
-- 200; ok=0 means somewhere along the redirect chain we hit a 4xx/5xx
-- or final 200 was missing. The migration UI surfaces failures for
-- one-click manual redirect creation.
CREATE TABLE IF NOT EXISTS migration_verifications (
    id            TEXT PRIMARY KEY,
    site_id       TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    migration_id  TEXT NOT NULL,
    source_url    TEXT NOT NULL,
    status_code   INTEGER NOT NULL DEFAULT 0,
    final_url     TEXT NOT NULL DEFAULT '',
    hops          INTEGER NOT NULL DEFAULT 0,
    ok            INTEGER NOT NULL DEFAULT 0,
    error         TEXT NOT NULL DEFAULT '',
    checked_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_migration_verifications_site ON migration_verifications(site_id);
CREATE INDEX IF NOT EXISTS idx_migration_verifications_migration ON migration_verifications(migration_id);

-- Verify jobs (Sprint 4 of the migration system, 2026-05-06). Async
-- container that owns a verify-live crawl: a single row tracks total +
-- processed counts so the UI can render a progress bar while the worker
-- writes per-URL results into migration_verifications. Lifts the 1k URL
-- sync-request cap because the request returns 202 immediately and the
-- crawl runs in a background goroutine for as long as it takes.
-- status flow: queued -> running -> done | failed | cancelled.
-- Reaper marks any 'running' rows older than the boot time as 'failed'
-- on startup so a server crash can't leave a job stuck running forever.
CREATE TABLE IF NOT EXISTS verify_jobs (
    id              TEXT PRIMARY KEY,
    site_id         TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    migration_id    TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'queued',
    total_urls      INTEGER NOT NULL DEFAULT 0,
    processed_urls  INTEGER NOT NULL DEFAULT 0,
    ok_count        INTEGER NOT NULL DEFAULT 0,
    fail_count      INTEGER NOT NULL DEFAULT 0,
    deployed_domain TEXT NOT NULL DEFAULT '',
    error           TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    started_at      TEXT NOT NULL DEFAULT '',
    completed_at    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_verify_jobs_site ON verify_jobs(site_id);
CREATE INDEX IF NOT EXISTS idx_verify_jobs_migration ON verify_jobs(migration_id);
CREATE INDEX IF NOT EXISTS idx_verify_jobs_status ON verify_jobs(status);

-- Missing URLs: 404s captured in real-time by the analytics nginx-log
-- parser, aggregated per (site_id, path). One row per unique 404 path;
-- hit_count increments on each new request. The UI lists them top-N by
-- hit_count so the operator can convert ongoing organic 404 traffic
-- into recovered link juice via the one-click "create redirect" flow.
-- This is the highest-leverage feature in the migration system: no
-- other CMS does it, yet every CMS migration produces a long tail of
-- forgotten URLs that keep getting linked from the wider web.
CREATE TABLE IF NOT EXISTS missing_urls (
    id          TEXT PRIMARY KEY,
    site_id     TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    path        TEXT NOT NULL,
    referer     TEXT NOT NULL DEFAULT '',
    hit_count   INTEGER NOT NULL DEFAULT 1,
    first_seen  TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen   TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, path)
);
CREATE INDEX IF NOT EXISTS idx_missing_urls_site ON missing_urls(site_id);
CREATE INDEX IF NOT EXISTS idx_missing_urls_count ON missing_urls(site_id, hit_count DESC);

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

-- AI-native Custom Collections (Sprint 4, 2026-05-04). The CMS layer that
-- turns Atomic Site from "landing page builder" into a true content
-- platform. A Collection is a user-defined content type (case studies,
-- portfolio items, products, team members, FAQ entries, events, etc.)
-- with a schema_json field array. Items conform to the schema and can
-- render either as standalone pages (settings.render_as_pages = true)
-- or as data fed into a collection_list block on a page. JSON-LD per
-- item is auto-generated from settings.schema_org_type.
CREATE TABLE IF NOT EXISTS collections (
    id            TEXT PRIMARY KEY,
    site_id       TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL,
    -- schema_json: array of field definitions
    -- [{name, type, label, required, options?, max_length?, ref_collection?}]
    -- types: text, textarea, richtext, number, boolean, date, datetime,
    --        url, email, image, gallery, select, multiselect, reference,
    --        color, json
    schema_json   TEXT NOT NULL DEFAULT '[]',
    -- settings_json: {render_as_pages, schema_org_type, list_page_template,
    --                 detail_page_template, items_per_index_page,
    --                 requires_member_role}
    -- requires_member_role is a Sprint 5 hook left passthrough today.
    settings_json TEXT NOT NULL DEFAULT '{}',
    sort_order    INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, slug)
);
CREATE INDEX IF NOT EXISTS idx_collections_site ON collections(site_id);

CREATE TABLE IF NOT EXISTS collection_items (
    id            TEXT PRIMARY KEY,
    collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    site_id       TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    slug          TEXT NOT NULL,
    title         TEXT NOT NULL,
    data_json     TEXT NOT NULL DEFAULT '{}',
    locale        TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'draft',
    published_at  TEXT NOT NULL DEFAULT '',
    sort_order    INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(collection_id, locale, slug)
);
CREATE INDEX IF NOT EXISTS idx_collection_items_collection ON collection_items(collection_id);
CREATE INDEX IF NOT EXISTS idx_collection_items_site ON collection_items(site_id);
CREATE INDEX IF NOT EXISTS idx_collection_items_status ON collection_items(status);

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

-- Conversion goals + events (Phase 31.1, 2026-05-06). Each site defines
-- goals that count as conversions; visit/event/form-submit activity is
-- evaluated against active goals and a row is appended to
-- conversion_events on a match. Three match strategies cover the common
-- cases without leaving operators stuck: a URL-pattern glob for "land
-- on /thank-you/* counts", an event_name for explicit
-- atomic.track('signup') JS calls, and a form_submit match keyed on
-- form_id for native CMS forms. value_cents is optional monetary value
-- for downstream revenue summing (Phase 31.4 wires deal revenue back
-- in; goals carry intent-value in the meantime).
CREATE TABLE IF NOT EXISTS conversion_goals (
    id              TEXT PRIMARY KEY,
    site_id         TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    match_type      TEXT NOT NULL,
    match_value     TEXT NOT NULL,
    value_cents     INTEGER NOT NULL DEFAULT 0,
    value_currency  TEXT NOT NULL DEFAULT 'EUR',
    active          INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, slug)
);
CREATE INDEX IF NOT EXISTS idx_conversion_goals_site ON conversion_goals(site_id);
CREATE INDEX IF NOT EXISTS idx_conversion_goals_active ON conversion_goals(site_id, active);

CREATE TABLE IF NOT EXISTS conversion_events (
    id               TEXT PRIMARY KEY,
    site_id          TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    goal_id          TEXT NOT NULL REFERENCES conversion_goals(id) ON DELETE CASCADE,
    fingerprint      TEXT NOT NULL,
    session_id       TEXT NOT NULL DEFAULT '',
    path             TEXT NOT NULL DEFAULT '',
    ts               TEXT NOT NULL,
    value_cents      INTEGER NOT NULL DEFAULT 0,
    value_currency   TEXT NOT NULL DEFAULT 'EUR',
    properties_json  TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_conversion_events_site_ts ON conversion_events(site_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_conversion_events_goal_ts ON conversion_events(goal_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_conversion_events_fingerprint ON conversion_events(site_id, fingerprint);

-- ============================================================
-- Shield: LLM-boundary tokenization (2026-05-08).
--
-- Per-MCP-connection token vault that backs internal/shield. PII
-- fields exposed to the LLM are replaced with markers of the form
-- [shield:<kind>:tok_<hex>:<hint>]. The agent reasons over the
-- markers; tool calls referencing markers are resolved server-side
-- via this table. Tokens are session-scoped, TTL-expired, and
-- cleaned up by ON DELETE CASCADE when the session row is deleted.
--
-- ciphertext = base64(AES-256-GCM(value, ATOMICSITE_SHIELD_KEY)).
-- Rotating the key invalidates live sessions (forces re-shield on
-- the next MCP turn). hint is an optional whitelisted metadata
-- string the agent reads ("domain=lawfirm.se,len=22") that must
-- never leak the value.
-- ============================================================
CREATE TABLE IF NOT EXISTS shield_sessions (
    id          TEXT PRIMARY KEY,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen   TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_shield_sessions_expires ON shield_sessions(expires_at);

CREATE TABLE IF NOT EXISTS shield_tokens (
    token       TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES shield_sessions(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    hint        TEXT NOT NULL DEFAULT '',
    ciphertext  TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_shield_tokens_session ON shield_tokens(session_id);

-- Outbound webhook subscriptions (2026-05-09). Per-site delivery
-- targets that receive HMAC-signed POSTs for content events. Each
-- subscription has a target URL, a shared HMAC secret (kept
-- server-side; never returned in list/read responses after creation),
-- and an optional event-types filter. Empty filter = subscribe to
-- every event.
CREATE TABLE IF NOT EXISTS webhook_subscriptions (
    id                TEXT PRIMARY KEY,
    site_id           TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    target_url        TEXT NOT NULL,
    secret            TEXT NOT NULL,
    event_types_json  TEXT NOT NULL DEFAULT '[]',
    active            INTEGER NOT NULL DEFAULT 1,
    last_delivered_at TEXT NOT NULL DEFAULT '',
    last_error        TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at        TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_webhook_subs_site ON webhook_subscriptions(site_id);

-- Outbound webhook delivery attempts. One row per (subscription,
-- event) pair; rows in 'pending' or 'retrying' state are picked up
-- by the WebhookDeliveryManager goroutine on its 5-second tick. Hard
-- cap at 5 attempts before flipping to 'dropped'.
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id                   TEXT PRIMARY KEY,
    subscription_id      TEXT NOT NULL REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    site_id              TEXT NOT NULL,
    event_type           TEXT NOT NULL,
    payload_json         TEXT NOT NULL,
    status               TEXT NOT NULL DEFAULT 'pending',
    attempts             INTEGER NOT NULL DEFAULT 0,
    next_attempt_at      TEXT NOT NULL DEFAULT (datetime('now')),
    last_response_status INTEGER NOT NULL DEFAULT 0,
    last_error           TEXT NOT NULL DEFAULT '',
    created_at           TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at         TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_webhook_del_status ON webhook_deliveries(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_webhook_del_site ON webhook_deliveries(site_id, created_at DESC);

-- Global search index (2026-05-09). FTS5 virtual table covering the
-- operator-facing content surface: pages, collection_items,
-- knowledgebase entries, and the site shell (name + slug). Populated
-- by handler-side `Upsert` calls on every content mutation; the
-- /api/sites/{siteID}/search/reindex endpoint rebuilds wholesale
-- after bulk imports / migrations. Standalone (not external-content)
-- so a row deletion in the underlying table doesn't orphan FTS rows
-- without an explicit DELETE.
--
-- tokenize: unicode61 + remove_diacritics covers Swedish + most EU
-- languages (Atomic Site is EU-sovereign by design); operators
-- running CJK content can drop in their own tokenizer once SQLite
-- supports them via build flags.
CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
    site_id UNINDEXED,
    kind UNINDEXED,
    resource_id UNINDEXED,
    title,
    body,
    tokenize = 'unicode61 remove_diacritics 2'
);

-- clarifications back the request_clarification MCP tool. The agent
-- creates a row when it needs a human to choose between options
-- (which hero variant, formal vs casual voice, etc); the dashboard
-- inbox surfaces pending rows; the human resolves them, the agent
-- polls until status flips. Closes the loop where the agent has to
-- guess + ship wrong + correct after the fact, costing the one-shot
-- wow.
--
-- options_json is a JSON-encoded array of choices. resolution_text
-- is the human's free-text answer (use when the chosen option needs
-- elaboration, or when none of the options fit). resolution_option
-- is the index into options_json (negative one means no option
-- picked, free-text only). context is the surrounding info the
-- human needs to make a sensible choice (page slug, block id,
-- screenshot summary, etc).
--
-- Indexes: site+status powers the dashboard inbox filter;
-- requested_by powers the agent-side "what am I waiting on" view.
CREATE TABLE IF NOT EXISTS clarifications (
    id                  TEXT PRIMARY KEY,
    site_id             TEXT NOT NULL,
    requested_by        TEXT NOT NULL DEFAULT '',
    question            TEXT NOT NULL,
    context             TEXT NOT NULL DEFAULT '',
    options_json        TEXT NOT NULL DEFAULT '[]',
    status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','resolved','cancelled')),
    resolution_option   INTEGER NOT NULL DEFAULT -1,
    resolution_text     TEXT NOT NULL DEFAULT '',
    resolved_by         TEXT NOT NULL DEFAULT '',
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    resolved_at         TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_clarifications_site_status ON clarifications(site_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_clarifications_agent ON clarifications(requested_by, status, created_at DESC);
