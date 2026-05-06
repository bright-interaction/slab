-- name: CreateWaitlistEntry :exec
INSERT OR IGNORE INTO waitlist (id, email, name, use_case, region, source_ip, user_agent)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetWaitlistByEmail :one
SELECT id, email, name, use_case, region, source_ip, user_agent,
       invited_at, invite_token, notes, created_at
FROM waitlist WHERE email = ?;

-- name: GetWaitlistByID :one
SELECT id, email, name, use_case, region, source_ip, user_agent,
       invited_at, invite_token, notes, created_at
FROM waitlist WHERE id = ?;

-- name: ListWaitlist :many
SELECT id, email, name, use_case, region, source_ip, user_agent,
       invited_at, invite_token, notes, created_at
FROM waitlist
ORDER BY created_at DESC
LIMIT 1000;

-- name: ListPendingWaitlist :many
SELECT id, email, name, use_case, region, source_ip, user_agent,
       invited_at, invite_token, notes, created_at
FROM waitlist
WHERE invited_at = ''
ORDER BY created_at ASC
LIMIT 1000;

-- name: MarkWaitlistInvited :exec
UPDATE waitlist
SET invited_at = datetime('now'), invite_token = ?, notes = ?
WHERE id = ?;

-- name: DeleteWaitlistEntry :exec
DELETE FROM waitlist WHERE id = ?;
