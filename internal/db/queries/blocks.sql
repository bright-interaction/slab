-- name: ListBlocksByPage :many
SELECT * FROM blocks WHERE page_id = ? ORDER BY sort_order ASC;

-- name: GetBlockByID :one
SELECT * FROM blocks WHERE id = ?;

-- name: CreateBlock :exec
INSERT INTO blocks (id, page_id, block_type, sort_order, data_json, style_json, is_visible)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateBlock :exec
UPDATE blocks SET
    block_type = ?,
    sort_order = ?,
    data_json = ?,
    style_json = ?,
    is_visible = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateBlockOrder :exec
UPDATE blocks SET sort_order = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteBlock :exec
DELETE FROM blocks WHERE id = ?;

-- name: DeleteBlocksByPage :exec
DELETE FROM blocks WHERE page_id = ?;
