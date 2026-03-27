-- name: ListMediaBySite :many
SELECT * FROM media WHERE site_id = ? ORDER BY created_at DESC;

-- name: GetMediaByID :one
SELECT * FROM media WHERE id = ?;

-- name: CreateMedia :exec
INSERT INTO media (id, site_id, filename, alt_text, mime_type, file_size, width, height, original_path)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateMedia :exec
UPDATE media SET
    alt_text = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateMediaVariants :exec
UPDATE media SET
    width = ?,
    height = ?,
    blurhash = ?,
    variants_json = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteMedia :exec
DELETE FROM media WHERE id = ?;

-- name: ListMediaByIDs :many
SELECT * FROM media WHERE id IN (sqlc.slice('ids'));
