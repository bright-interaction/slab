-- name: ListKnowledgebaseBySite :many
SELECT * FROM knowledgebase_entries
WHERE site_id = ? ORDER BY category, sort_order;

-- name: ListActiveKnowledgebaseBySite :many
SELECT * FROM knowledgebase_entries
WHERE site_id = ? AND is_active = 1 ORDER BY category, sort_order;

-- name: GetKnowledgebaseByID :one
SELECT * FROM knowledgebase_entries WHERE id = ?;

-- name: CreateKnowledgebaseEntry :exec
INSERT INTO knowledgebase_entries (id, site_id, category, title, content, is_active, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, 1, ?, datetime('now'), datetime('now'));

-- name: UpdateKnowledgebaseEntry :exec
UPDATE knowledgebase_entries
SET category = ?, title = ?, content = ?, is_active = ?, sort_order = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteKnowledgebaseEntry :exec
DELETE FROM knowledgebase_entries WHERE id = ?;
