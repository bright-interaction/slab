-- name: ListActiveApps :many
SELECT * FROM apps WHERE is_active = 1 ORDER BY category ASC, name ASC;

-- name: ListAppsByCategory :many
SELECT * FROM apps WHERE is_active = 1 AND category = ? ORDER BY name ASC;

-- name: GetAppByID :one
SELECT * FROM apps WHERE id = ?;

-- name: GetAppBySlug :one
SELECT * FROM apps WHERE slug = ?;

-- name: UpsertApp :exec
INSERT INTO apps (id, slug, name, description, category, publisher, icon_url, mcp_url, docs_url, version, credentials_schema_json, is_curated, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    slug = excluded.slug,
    name = excluded.name,
    description = excluded.description,
    category = excluded.category,
    publisher = excluded.publisher,
    icon_url = excluded.icon_url,
    mcp_url = excluded.mcp_url,
    docs_url = excluded.docs_url,
    version = excluded.version,
    credentials_schema_json = excluded.credentials_schema_json,
    is_curated = excluded.is_curated,
    is_active = excluded.is_active,
    updated_at = datetime('now');

-- name: DeactivateApp :exec
UPDATE apps SET is_active = 0, updated_at = datetime('now') WHERE id = ?;

-- name: ListSiteAppInstalls :many
SELECT i.*, a.slug as app_slug, a.name as app_name, a.description as app_description,
       a.category as app_category, a.icon_url as app_icon_url, a.publisher as app_publisher,
       a.version as app_version
FROM site_app_installs i
JOIN apps a ON a.id = i.app_id
WHERE i.site_id = ?
ORDER BY a.category ASC, a.name ASC;

-- name: GetSiteAppInstall :one
SELECT * FROM site_app_installs WHERE site_id = ? AND app_id = ?;

-- name: UpsertSiteAppInstall :exec
INSERT INTO site_app_installs (site_id, app_id, status, credentials_json, installed_by, notes)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(site_id, app_id) DO UPDATE SET
    status = excluded.status,
    credentials_json = excluded.credentials_json,
    installed_by = excluded.installed_by,
    notes = excluded.notes,
    updated_at = datetime('now');

-- name: DeleteSiteAppInstall :exec
DELETE FROM site_app_installs WHERE site_id = ? AND app_id = ?;

-- name: BumpSiteAppInstallLastUsed :exec
UPDATE site_app_installs SET last_used_at = datetime('now'), updated_at = datetime('now')
WHERE site_id = ? AND app_id = ?;

-- name: SetSiteAppInstallStatus :exec
UPDATE site_app_installs SET status = ?, updated_at = datetime('now')
WHERE site_id = ? AND app_id = ?;
