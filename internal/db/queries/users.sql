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

-- name: UpdateUserName :exec
UPDATE users SET name = ?, updated_at = datetime('now') WHERE id = ?;

-- name: UpdateUserRole :exec
UPDATE users SET role = ?, token_version = token_version + 1, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: SetUserTOTPSecret :exec
-- Stages a fresh secret without flipping enrolled_at, so the user
-- can scan the QR + try a code before the secret is locked in.
UPDATE users SET totp_secret = ?, updated_at = datetime('now') WHERE id = ?;

-- name: ConfirmUserTOTPEnrollment :exec
-- Run after a successful first-code verify. Sets enrolled_at and
-- writes the bcrypt-hashed recovery codes JSON.
UPDATE users SET totp_enrolled_at = datetime('now'),
                 totp_recovery_json = ?,
                 token_version = token_version + 1,
                 updated_at = datetime('now')
WHERE id = ?;

-- name: ClearUserTOTP :exec
-- Disables MFA and bumps token_version so any session that thought
-- it had MFA is invalidated. Only callable by the user themselves
-- (after re-confirming current password) or by a workspace admin.
UPDATE users SET totp_secret = '',
                 totp_enrolled_at = '',
                 totp_recovery_json = '[]',
                 token_version = token_version + 1,
                 updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateUserTOTPRecovery :exec
-- Rewrites the recovery code list (e.g. after a single-use code is
-- consumed during a recovery login).
UPDATE users SET totp_recovery_json = ?, updated_at = datetime('now') WHERE id = ?;
