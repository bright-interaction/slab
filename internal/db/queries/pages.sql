-- name: ListPagesBySite :many
SELECT * FROM pages WHERE site_id = ? ORDER BY sort_order ASC;

-- name: ListPublishedPagesBySite :many
SELECT * FROM pages WHERE site_id = ? AND status = 'published' ORDER BY sort_order ASC;

-- name: GetPageByID :one
SELECT * FROM pages WHERE id = ?;

-- name: GetPageBySiteAndSlug :one
SELECT * FROM pages WHERE site_id = ? AND slug = ?;

-- name: CreatePage :exec
INSERT INTO pages (id, site_id, title, slug, layout, sort_order, show_in_nav)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePage :exec
UPDATE pages SET
    title = ?,
    slug = ?,
    status = ?,
    meta_title = ?,
    meta_description = ?,
    og_image_id = ?,
    layout = ?,
    sort_order = ?,
    show_in_nav = ?,
    nav_label = ?,
    no_index = ?,
    canonical_url = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: UpdatePageOrder :exec
UPDATE pages SET sort_order = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeletePage :exec
DELETE FROM pages WHERE id = ?;

-- name: CountPagesBySite :one
SELECT COUNT(*) FROM pages WHERE site_id = ?;
