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

-- name: SumStorageBytesByWorkspace :one
-- Workspace-level storage rollup. Joins media -> sites by workspace_id
-- so a single SUM covers every site in the workspace. Used by the
-- plan-ladder enforcement (solo plan = 1 GB workspace-wide regardless
-- of how many sites exist or how their per-site caps are configured).
SELECT COALESCE(SUM(m.file_size), 0) AS bytes
FROM media m
JOIN sites s ON s.id = m.site_id
WHERE s.workspace_id = ?;

-- name: SumBuildMinutesByWorkspaceSinceCutoff :one
-- Workspace-level rolling build-minutes rollup. Same shape as the
-- per-site query but joined through sites so the cutoff applies
-- to every deployment under the workspace.
SELECT COALESCE(SUM(d.duration_ms), 0) AS duration_ms_total
FROM deployments d
JOIN sites s ON s.id = d.site_id
WHERE s.workspace_id = ? AND d.created_at >= ?;

-- name: GetWorkspaceLastActivity :one
-- Returns the most recent activity timestamp across the workspace:
-- the workspace itself, any of its sites' updated_at / last_build_at /
-- last_deploy_at, and any recent deployment created_at. Used by the
-- lifecycle sweep to decide if a workspace has been idle long enough
-- to pause / delete. Empty workspace returns the workspace's own
-- updated_at.
SELECT COALESCE(MAX(activity), '') AS last_activity FROM (
    SELECT w.updated_at AS activity FROM workspaces w WHERE w.id = ?1
    UNION ALL
    SELECT s.updated_at AS activity FROM sites s WHERE s.workspace_id = ?1
    UNION ALL
    SELECT s.last_build_at AS activity FROM sites s WHERE s.workspace_id = ?1 AND s.last_build_at != ''
    UNION ALL
    SELECT s.last_deploy_at AS activity FROM sites s WHERE s.workspace_id = ?1 AND s.last_deploy_at != ''
    UNION ALL
    SELECT d.created_at AS activity FROM deployments d
        JOIN sites s ON s.id = d.site_id
        WHERE s.workspace_id = ?1
);

-- name: UpdateWorkspaceStatus :exec
UPDATE workspaces SET status = ?, updated_at = datetime('now') WHERE id = ?;

-- name: ListWorkspacesForLifecycleSweep :many
-- Returns every workspace except those already in the deleted state
-- (terminal). The sweep uses each row's id to query
-- GetWorkspaceLastActivity and decide whether to pause / delete.
SELECT id, name, slug, plan, status, created_at, updated_at
FROM workspaces
WHERE status != 'deleted'
ORDER BY id;

-- name: ListSitesForQuotaAudit :many
SELECT id, name, slug, storage_quota_bytes, build_minutes_quota, quota_overage_blocked
FROM sites
ORDER BY updated_at DESC;
