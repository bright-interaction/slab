-- name: ListSites :many
SELECT * FROM sites ORDER BY updated_at DESC;

-- name: GetSiteByID :one
SELECT * FROM sites WHERE id = ?;

-- name: GetSiteBySlug :one
SELECT * FROM sites WHERE slug = ?;

-- name: CreateSite :exec
INSERT INTO sites (id, workspace_id, name, slug, domain, primary_color, secondary_color, bg_color, text_color, font_heading, font_body, lang)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateSite :exec
UPDATE sites SET
    name = ?,
    slug = ?,
    domain = ?,
    status = ?,
    primary_color = ?,
    secondary_color = ?,
    bg_color = ?,
    text_color = ?,
    surface_color = ?,
    border_color = ?,
    muted_color = ?,
    accent_color = ?,
    on_primary_color = ?,
    font_heading = ?,
    font_body = ?,
    meta_title = ?,
    meta_description = ?,
    og_image_id = ?,
    favicon_id = ?,
    ga4_id = ?,
    umami_id = ?,
    umami_url = ?,
    cookieproof_domain = ?,
    lang = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateSiteBuildStatus :exec
UPDATE sites SET
    last_build_at = ?,
    last_build_status = ?,
    last_build_error = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateSiteDeployAt :exec
UPDATE sites SET last_deploy_at = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteSite :exec
DELETE FROM sites WHERE id = ?;

-- name: GetSiteQuota :one
SELECT storage_quota_bytes, build_minutes_quota, quota_overage_blocked FROM sites WHERE id = ?;

-- name: UpdateSiteQuota :exec
UPDATE sites
SET storage_quota_bytes = ?, build_minutes_quota = ?, quota_overage_blocked = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: SumStorageBytesBySite :one
SELECT COALESCE(SUM(file_size), 0) AS bytes FROM media WHERE site_id = ?;

-- name: SumBuildMinutesBySiteSinceCutoff :one
SELECT COALESCE(SUM(duration_ms), 0) AS duration_ms_total
FROM deployments
WHERE site_id = ? AND created_at >= ?;

-- name: ListSitesForQuotaAudit :many
SELECT id, name, slug, storage_quota_bytes, build_minutes_quota, quota_overage_blocked
FROM sites
ORDER BY updated_at DESC;
