-- name: CreateClarification :exec
INSERT INTO clarifications (id, site_id, requested_by, question, context, options_json, status, resolution_option, resolution_text, resolved_by, created_at, resolved_at)
VALUES (?, ?, ?, ?, ?, ?, 'pending', -1, '', '', datetime('now'), '');

-- name: GetClarificationByID :one
SELECT * FROM clarifications WHERE id = ? AND site_id = ?;

-- name: ListPendingClarificationsBySite :many
SELECT * FROM clarifications WHERE site_id = ? AND status = 'pending' ORDER BY created_at DESC LIMIT ?;

-- name: ListPendingClarificationsByAgent :many
SELECT * FROM clarifications WHERE site_id = ? AND requested_by = ? AND status = 'pending' ORDER BY created_at DESC LIMIT ?;

-- name: ListAllClarificationsBySite :many
SELECT * FROM clarifications WHERE site_id = ? ORDER BY created_at DESC LIMIT ?;

-- name: ResolveClarification :exec
UPDATE clarifications
SET status = 'resolved',
    resolution_option = ?,
    resolution_text = ?,
    resolved_by = ?,
    resolved_at = datetime('now')
WHERE id = ? AND site_id = ? AND status = 'pending';

-- name: CancelClarification :exec
UPDATE clarifications
SET status = 'cancelled',
    resolved_by = ?,
    resolved_at = datetime('now')
WHERE id = ? AND site_id = ? AND status = 'pending';

-- name: CountPendingClarificationsBySite :one
SELECT COUNT(*) FROM clarifications WHERE site_id = ? AND status = 'pending';
