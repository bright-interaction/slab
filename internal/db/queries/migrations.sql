-- name: ListMigrationsBySite :many
SELECT * FROM migrations WHERE site_id = ? ORDER BY created_at DESC;

-- name: GetMigrationByID :one
SELECT * FROM migrations WHERE id = ?;

-- name: CreateMigration :exec
INSERT INTO migrations (id, site_id, source_url, source_type, manifest_json, status, error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'));

-- name: UpdateMigrationManifest :exec
UPDATE migrations
SET manifest_json = ?, status = ?, error = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateMigrationStatus :exec
UPDATE migrations
SET status = ?, error = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteMigration :exec
DELETE FROM migrations WHERE id = ?;
