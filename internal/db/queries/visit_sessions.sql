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

-- name: SetVisitSessionEmail :execrows
-- Phase 31.3 (2026-05-06): form-submit / explicit-identify path. Sets
-- email + stamps identified_at if it wasn't already (a re-identify keeps
-- the original timestamp so analytics + retention treat the session as
-- one continuous identified visitor). Returns rows affected so the
-- caller can detect the no-session case (0) without a second roundtrip.
UPDATE visit_sessions SET
    email = ?,
    identified_at = COALESCE(NULLIF(identified_at, ''), ?),
    last_seen_at = ?
WHERE site_id = ? AND fingerprint = ?;

-- name: ListIdentifiedSessions :many
SELECT * FROM visit_sessions
WHERE site_id = ? AND identified_at != ''
ORDER BY identified_at DESC
LIMIT ? OFFSET ?;

-- name: UpsertVisitorMetadataByVisitorID :execrows
UPDATE visit_sessions
SET metadata_json = ?,
    metadata_expires_at = ?,
    identity_confirmed_at = ?,
    last_seen_at = datetime('now')
WHERE site_id = ? AND visitor_id = ? AND visitor_id != '';

-- name: UpsertVisitorMetadataByFingerprint :execrows
UPDATE visit_sessions
SET metadata_json = ?,
    metadata_expires_at = ?,
    identity_confirmed_at = ?,
    last_seen_at = datetime('now')
WHERE site_id = ? AND fingerprint = ?;

-- name: GetVisitorMetadataByFingerprint :one
SELECT id, site_id, fingerprint, visitor_id, email, consent_method,
       consent_categories_json, started_at, last_seen_at, page_count,
       identified_at, metadata_json, metadata_expires_at, identity_confirmed_at
FROM visit_sessions
WHERE site_id = ? AND fingerprint = ?;

-- ListVisitorMetadataKeys is intentionally implemented as a raw *sql.DB
-- query in code (see internal/agent/context.go listVisitorMetadataKeys),
-- not via sqlc, because sqlc's static analyzer can't parse SQLite's
-- json_each(vs.metadata_json) key/value virtual columns.

-- name: DeleteVisitSessionsBySiteOlderThan :execrows
-- Retention purge for stale sessions. last_seen_at is RFC3339 UTC; caller
-- passes the cutoff. Identified sessions are kept regardless (they back the
-- analytics dashboard's identified-visitor list and represent CRM-confirmed
-- contacts) -- only fully anonymous sessions older than the cutoff drop.
DELETE FROM visit_sessions
WHERE site_id = ? AND last_seen_at < ? AND identified_at = '';
