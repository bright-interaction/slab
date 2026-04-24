-- name: ListCSSClassesBySite :many
SELECT * FROM css_classes WHERE site_id = ? ORDER BY category, sort_order, name;

-- name: GetCSSClassByID :one
SELECT * FROM css_classes WHERE id = ?;

-- name: GetCSSClassByName :one
SELECT * FROM css_classes WHERE site_id = ? AND name = ?;

-- name: CreateCSSClass :exec
INSERT INTO css_classes (id, site_id, name, category, css, usage_note, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'));

-- name: UpdateCSSClass :exec
UPDATE css_classes
SET name = ?, category = ?, css = ?, usage_note = ?, sort_order = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteCSSClass :exec
DELETE FROM css_classes WHERE id = ?;
