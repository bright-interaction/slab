-- name: ListCollectionsBySite :many
SELECT * FROM collections WHERE site_id = ? ORDER BY sort_order, name;

-- name: GetCollectionByID :one
SELECT * FROM collections WHERE id = ?;

-- name: GetCollectionBySiteAndSlug :one
SELECT * FROM collections WHERE site_id = ? AND slug = ?;

-- name: CreateCollection :exec
INSERT INTO collections (id, site_id, name, slug, schema_json, settings_json, sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateCollection :exec
UPDATE collections
SET name = ?,
    slug = ?,
    schema_json = ?,
    settings_json = ?,
    sort_order = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE id = ?;

-- name: ListItemsByCollection :many
SELECT * FROM collection_items
WHERE collection_id = ?
ORDER BY sort_order, published_at DESC, created_at DESC;

-- name: ListPublishedItemsByCollection :many
SELECT * FROM collection_items
WHERE collection_id = ? AND status = 'published'
ORDER BY sort_order, published_at DESC, created_at DESC;

-- name: ListPublishedItemsByCollectionAndLocale :many
SELECT * FROM collection_items
WHERE collection_id = ? AND status = 'published' AND locale = ?
ORDER BY sort_order, published_at DESC, created_at DESC;

-- name: ListAllPublishedItemsBySite :many
SELECT * FROM collection_items
WHERE site_id = ? AND status = 'published'
ORDER BY collection_id, sort_order, published_at DESC, created_at DESC;

-- name: GetItemByID :one
SELECT * FROM collection_items WHERE id = ?;

-- name: GetItemByCollectionLocaleSlug :one
SELECT * FROM collection_items
WHERE collection_id = ? AND locale = ? AND slug = ?;

-- name: CreateItem :exec
INSERT INTO collection_items
    (id, collection_id, site_id, slug, title, data_json, locale, status, published_at, sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpsertItem :one
-- Sprint 4 (2026-05-06): re-import upsert. Conflict on the
-- (collection_id, locale, slug) compound natural key.
INSERT INTO collection_items
    (id, collection_id, site_id, slug, title, data_json, locale, status, published_at, sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(collection_id, locale, slug) DO UPDATE SET
    title        = excluded.title,
    data_json    = excluded.data_json,
    status       = excluded.status,
    published_at = excluded.published_at,
    sort_order   = excluded.sort_order,
    updated_at   = datetime('now')
RETURNING *;

-- name: UpdateItem :exec
UPDATE collection_items
SET slug = ?,
    title = ?,
    data_json = ?,
    locale = ?,
    status = ?,
    published_at = ?,
    sort_order = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteItem :exec
DELETE FROM collection_items WHERE id = ?;

-- name: CountItemsByCollection :one
SELECT COUNT(*) AS total FROM collection_items WHERE collection_id = ?;

-- name: ListLocalesByCollection :many
SELECT DISTINCT locale FROM collection_items WHERE collection_id = ? ORDER BY locale;
