-- name: CreateShieldSession :exec
INSERT INTO shield_sessions (id, expires_at)
VALUES (?, ?);

-- name: GetShieldSession :one
SELECT id, created_at, last_seen, expires_at
FROM shield_sessions
WHERE id = ?;

-- name: TouchShieldSession :exec
UPDATE shield_sessions
SET last_seen = datetime('now'), expires_at = ?
WHERE id = ?;

-- name: DeleteShieldSession :exec
DELETE FROM shield_sessions WHERE id = ?;

-- name: SweepExpiredShieldSessions :exec
DELETE FROM shield_sessions WHERE expires_at < datetime('now');

-- name: CreateShieldToken :exec
INSERT INTO shield_tokens (token, session_id, kind, hint, ciphertext)
VALUES (?, ?, ?, ?, ?);

-- name: GetShieldToken :one
SELECT token, session_id, kind, hint, ciphertext, created_at
FROM shield_tokens
WHERE token = ? AND session_id = ?;

-- name: ListShieldTokensBySession :many
SELECT token, kind, hint, created_at
FROM shield_tokens
WHERE session_id = ?
ORDER BY created_at;

-- name: CountShieldTokensBySession :one
SELECT COUNT(*) FROM shield_tokens WHERE session_id = ?;
