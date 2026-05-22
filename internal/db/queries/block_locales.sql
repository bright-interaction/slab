-- name: ListBlockLocalesByBlock :many
SELECT * FROM block_locales WHERE block_id = ? ORDER BY locale ASC;

-- name: ListBlockLocalesByPage :many
SELECT bl.* FROM block_locales bl
JOIN blocks b ON b.id = bl.block_id
WHERE b.page_id = ?
ORDER BY b.sort_order ASC, bl.locale ASC;

-- name: GetBlockLocale :one
SELECT * FROM block_locales WHERE block_id = ? AND locale = ?;

-- name: UpsertBlockLocale :exec
INSERT INTO block_locales (block_id, locale, data_json, is_visible)
VALUES (?, ?, ?, ?)
ON CONFLICT(block_id, locale) DO UPDATE SET
    data_json = excluded.data_json,
    is_visible = excluded.is_visible,
    updated_at = datetime('now');

-- name: DeleteBlockLocale :exec
DELETE FROM block_locales WHERE block_id = ? AND locale = ?;

-- name: DeleteBlockLocalesByLocale :exec
DELETE FROM block_locales WHERE block_id IN (
    SELECT b.id FROM blocks b
    JOIN pages p ON p.id = b.page_id
    WHERE p.site_id = ?
) AND locale = ?;
