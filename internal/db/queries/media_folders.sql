-- name: ListMediaFoldersBySite :many
SELECT site_id, name, is_system, created_at
FROM media_folders
WHERE site_id = ?
ORDER BY is_system DESC, name ASC;

-- name: GetMediaFolder :one
SELECT site_id, name, is_system, created_at
FROM media_folders
WHERE site_id = ? AND name = ?;

-- name: CreateMediaFolder :exec
INSERT INTO media_folders (site_id, name, is_system)
VALUES (?, ?, ?);

-- name: EnsureMediaFolder :exec
INSERT OR IGNORE INTO media_folders (site_id, name, is_system)
VALUES (?, ?, ?);

-- name: DeleteMediaFolder :exec
DELETE FROM media_folders WHERE site_id = ? AND name = ? AND is_system = 0;
