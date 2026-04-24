-- name: ListComponentsBySite :many
SELECT * FROM components WHERE site_id = ? ORDER BY category, name;

-- name: GetComponentByID :one
SELECT * FROM components WHERE id = ?;

-- name: GetComponentByName :one
SELECT * FROM components WHERE site_id = ? AND name = ?;

-- name: CreateComponent :exec
INSERT INTO components (id, site_id, name, category, template, props_schema, css_classes, usage_note, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'));

-- name: UpdateComponent :exec
UPDATE components
SET name = ?, category = ?, template = ?, props_schema = ?, css_classes = ?, usage_note = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteComponent :exec
DELETE FROM components WHERE id = ?;
