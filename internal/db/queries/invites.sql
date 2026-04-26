-- name: CreateInvite :exec
INSERT INTO invites (id, email, role, token, created_by, expires_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetInviteByToken :one
SELECT * FROM invites WHERE token = ?;

-- name: ListPendingInvites :many
SELECT * FROM invites
WHERE used_at = '' AND expires_at > datetime('now')
ORDER BY created_at DESC;

-- name: MarkInviteUsed :exec
UPDATE invites SET used_at = datetime('now') WHERE id = ?;

-- name: DeleteInvite :exec
DELETE FROM invites WHERE id = ?;
