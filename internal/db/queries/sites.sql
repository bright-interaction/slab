-- name: ListSites :many
SELECT * FROM sites ORDER BY updated_at DESC;

-- name: GetSiteByID :one
SELECT * FROM sites WHERE id = ?;

-- name: GetSiteBySlug :one
SELECT * FROM sites WHERE slug = ?;

-- name: CreateSite :exec
INSERT INTO sites (id, name, slug, domain, primary_color, secondary_color, bg_color, text_color, font_heading, font_body, lang)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

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
    font_heading = ?,
    font_body = ?,
    meta_title = ?,
    meta_description = ?,
    og_image_id = ?,
    favicon_id = ?,
    ga4_id = ?,
    umami_id = ?,
    umami_url = ?,
    cookiebit_domain = ?,
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
