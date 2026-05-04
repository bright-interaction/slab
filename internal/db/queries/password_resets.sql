-- name: CreatePasswordReset :exec
INSERT INTO password_resets (id, user_id, token_hash, expires_at, requester_ip)
VALUES (?, ?, ?, ?, ?);

-- name: GetPasswordResetByTokenHash :one
SELECT id, user_id, token_hash, expires_at, used_at, requester_ip, created_at
FROM password_resets
WHERE token_hash = ?;

-- name: MarkPasswordResetUsed :exec
UPDATE password_resets SET used_at = datetime('now') WHERE id = ?;

-- name: PurgeExpiredPasswordResets :exec
DELETE FROM password_resets WHERE expires_at < datetime('now', '-1 day');
