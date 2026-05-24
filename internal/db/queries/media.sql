-- name: ListMediaBySite :many
SELECT * FROM media WHERE site_id = ? ORDER BY created_at DESC;

-- name: ListMediaBySitePaginated :many
SELECT * FROM media WHERE site_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListMediaInFolderPaginated :many
SELECT * FROM media WHERE site_id = ? AND folder = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListUnfiledMediaPaginated :many
SELECT * FROM media WHERE site_id = ? AND folder = '' ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountMediaBySite :one
SELECT COUNT(*) FROM media WHERE site_id = ?;

-- name: CountMediaInFolder :one
SELECT COUNT(*) FROM media WHERE site_id = ? AND folder = ?;

-- name: CountUnfiledMedia :one
SELECT COUNT(*) FROM media WHERE site_id = ? AND folder = '';

-- name: GetMediaByID :one
SELECT * FROM media WHERE id = ?;

-- name: CreateMedia :exec
INSERT INTO media (id, site_id, filename, alt_text, mime_type, file_size, width, height, original_path, folder)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateMedia :exec
UPDATE media SET
    alt_text = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateMediaFolder :exec
UPDATE media SET
    folder = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: ClearMediaFolder :exec
UPDATE media SET
    folder = '',
    updated_at = datetime('now')
WHERE site_id = ? AND folder = ?;

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

-- name: UpdateMediaFocalPoint :exec
UPDATE media SET
    focal_x = ?,
    focal_y = ?,
    updated_at = datetime('now')
WHERE id = ? AND site_id = ?;
