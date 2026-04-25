-- name: UpsertVisitSession :exec
INSERT INTO visit_sessions (id, site_id, fingerprint, started_at, last_seen_at, page_count)
VALUES (?, ?, ?, ?, ?, 1)
ON CONFLICT(site_id, fingerprint) DO UPDATE SET
    last_seen_at = excluded.last_seen_at,
    page_count = visit_sessions.page_count + 1;

-- name: IdentifyVisitSession :exec
UPDATE visit_sessions SET
    visitor_id = ?,
    email = ?,
    consent_method = ?,
    consent_categories_json = ?,
    identified_at = ?,
    last_seen_at = ?
WHERE site_id = ? AND fingerprint = ?;

-- name: GetSessionByFingerprint :one
SELECT * FROM visit_sessions
WHERE site_id = ? AND fingerprint = ?;

-- name: ListIdentifiedSessions :many
SELECT * FROM visit_sessions
WHERE site_id = ? AND identified_at != ''
ORDER BY identified_at DESC
LIMIT ? OFFSET ?;
