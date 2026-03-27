-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: CreateUser :exec
INSERT INTO users (id, email, password_hash, name, role)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = ?, token_version = token_version + 1, updated_at = datetime('now')
WHERE id = ?;

-- name: IncrementTokenVersion :exec
UPDATE users SET token_version = token_version + 1, updated_at = datetime('now')
WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;
