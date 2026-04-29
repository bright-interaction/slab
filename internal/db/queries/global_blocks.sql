-- name: ListGlobalBlocksBySite :many
SELECT * FROM global_blocks WHERE site_id = ? ORDER BY slot ASC, sort_order ASC;

-- name: ListActiveGlobalBlocksBySite :many
SELECT * FROM global_blocks WHERE site_id = ? AND is_active = 1 ORDER BY slot ASC, sort_order ASC;

-- name: GetGlobalBlockByID :one
SELECT * FROM global_blocks WHERE id = ?;

-- name: CreateGlobalBlock :exec
INSERT INTO global_blocks (id, site_id, name, slot, block_type, data_json, style_json, sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateGlobalBlock :exec
UPDATE global_blocks SET
    name = ?,
    slot = ?,
    block_type = ?,
    data_json = ?,
    style_json = ?,
    is_active = ?,
    sort_order = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteGlobalBlock :exec
DELETE FROM global_blocks WHERE id = ?;
