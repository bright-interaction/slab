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
    cookiebit_domain TEXT NOT NULL DEFAULT '',
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
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_deployments_site ON deployments(site_id);

CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
