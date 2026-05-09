-- name: ListPagesBySite :many
SELECT * FROM pages WHERE site_id = ? ORDER BY sort_order ASC;

-- name: ListPublishedPagesBySite :many
SELECT * FROM pages WHERE site_id = ? AND status = 'published' ORDER BY sort_order ASC;

-- name: GetPageByID :one
SELECT * FROM pages WHERE id = ?;

-- name: GetPageBySiteAndSlug :one
-- Tolerates slug forms with and without a leading slash. The MCP layer
-- normalizes incoming slugs to a leading-slash form, but the migration
-- porter (and agent.UpdatePage when callers pass new_slug raw) stores
-- slugs without the slash. This OR keeps both reachable.
SELECT * FROM pages
WHERE site_id = sqlc.arg(site_id)
  AND (slug = sqlc.arg(slug) OR slug = LTRIM(sqlc.arg(slug), '/'));

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
    hide_global_blocks = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: UpdatePageOrder :exec
UPDATE pages SET sort_order = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeletePage :exec
DELETE FROM pages WHERE id = ?;

-- name: CountPagesBySite :one
SELECT COUNT(*) FROM pages WHERE site_id = ?;

-- name: ListExistingPageSlugsBySite :many
-- Sprint 4 (2026-05-06): used by the migration upsert path so the
-- plan-quota gate can subtract pages whose slug already exists from
-- the projected addition. Returns slugs without leading slash.
SELECT slug FROM pages WHERE site_id = ?;

-- name: UpsertPage :one
-- Sprint 4 (2026-05-06): re-import upsert mode. INSERTs fresh, UPDATEs
-- on (site_id, slug) conflict. RETURNING * lets the porter detect the
-- outcome by comparing returned row.ID to the candidate ID it passed.
INSERT INTO pages
    (id, site_id, title, slug, status, meta_title, meta_description,
     og_image_id, layout, sort_order, show_in_nav, no_index,
     canonical_url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(site_id, slug) DO UPDATE SET
    title            = excluded.title,
    status           = excluded.status,
    meta_title       = excluded.meta_title,
    meta_description = excluded.meta_description,
    og_image_id      = excluded.og_image_id,
    layout           = excluded.layout,
    sort_order       = excluded.sort_order,
    no_index         = excluded.no_index,
    canonical_url    = excluded.canonical_url,
    updated_at       = datetime('now')
RETURNING *;
