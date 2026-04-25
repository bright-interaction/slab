-- AtomicSite database schema (SQLite)

CREATE TABLE IF NOT EXISTS users (
    id             TEXT PRIMARY KEY,
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    name           TEXT NOT NULL DEFAULT '',
    role           TEXT NOT NULL DEFAULT 'admin',
    token_version  INTEGER NOT NULL DEFAULT 1,
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS sites (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    slug             TEXT NOT NULL UNIQUE,
    domain           TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'draft',
    -- Branding
    primary_color    TEXT NOT NULL DEFAULT '#D4AF37',
    secondary_color  TEXT NOT NULL DEFAULT '#935FA7',
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
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, slug)
);
CREATE INDEX IF NOT EXISTS idx_pages_site ON pages(site_id);

CREATE TABLE IF NOT EXISTS blocks (
    id               TEXT PRIMARY KEY,
    page_id          TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    block_type       TEXT NOT NULL,
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
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_media_site ON media(site_id);

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

-- Silos/sections within a site architecture
CREATE TABLE IF NOT EXISTS site_silos (
    id          TEXT PRIMARY KEY,
    site_id     TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug_prefix TEXT NOT NULL,
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

-- Allowlisted external scripts (feeds CSP generation + guardrails)
CREATE TABLE IF NOT EXISTS allowed_scripts (
    id         TEXT PRIMARY KEY,
    site_id    TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    domain     TEXT NOT NULL,
    purpose    TEXT NOT NULL DEFAULT '',
    is_active  INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(site_id, domain)
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

-- Individual visit events parsed from Nginx JSON access logs
CREATE TABLE IF NOT EXISTS visit_events (
    id           TEXT PRIMARY KEY,
    site_id      TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    fingerprint  TEXT NOT NULL,
    session_id   TEXT NOT NULL DEFAULT '',
    path         TEXT NOT NULL,
    referer      TEXT NOT NULL DEFAULT '',
    status       INTEGER NOT NULL DEFAULT 200,
    ms           INTEGER NOT NULL DEFAULT 0,
    ts           TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_visit_events_site_ts ON visit_events(site_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_visit_events_fingerprint ON visit_events(site_id, fingerprint);
CREATE INDEX IF NOT EXISTS idx_visit_events_session ON visit_events(site_id, session_id);

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
    UNIQUE(site_id, fingerprint)
);
CREATE INDEX IF NOT EXISTS idx_visit_sessions_site_seen ON visit_sessions(site_id, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_visit_sessions_identified ON visit_sessions(site_id, identified_at) WHERE identified_at != '';
