-- name: ListGuardrailsBySite :many
SELECT * FROM guardrail_rules WHERE site_id = ? ORDER BY rule_type, target;

-- name: ListActiveGuardrailsBySite :many
SELECT * FROM guardrail_rules WHERE site_id = ? AND is_active = 1 ORDER BY rule_type, target;

-- name: GetGuardrailByID :one
SELECT * FROM guardrail_rules WHERE id = ?;

-- name: CreateGuardrail :exec
INSERT INTO guardrail_rules (id, site_id, rule_type, target, value, severity, is_active, created_at)
VALUES (?, ?, ?, ?, ?, ?, 1, datetime('now'));

-- name: UpdateGuardrail :exec
UPDATE guardrail_rules
SET rule_type = ?, target = ?, value = ?, severity = ?, is_active = ?
WHERE id = ?;

-- name: DeleteGuardrail :exec
DELETE FROM guardrail_rules WHERE id = ?;
