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

-- name: ListBlocksBySite :many
-- Audit H1: single-query alternative to "list pages, loop with
-- ListBlocksByPage". Returns every block for the site joined with its
-- page id; caller groups by page_id in Go. Removes the N+1 pattern in
-- agent context Build.
SELECT b.id, b.page_id, b.block_type, b.sort_order, b.data_json,
       b.style_json, b.is_visible, b.created_at, b.updated_at
FROM blocks b
JOIN pages p ON p.id = b.page_id
WHERE p.site_id = ?
ORDER BY b.page_id, b.sort_order ASC;
