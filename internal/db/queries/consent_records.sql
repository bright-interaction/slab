-- name: RecordConsent :exec
INSERT INTO consent_records (
    id, site_id, session_id, fingerprint, domain, page_url, referrer, user_agent,
    ip_hash, consent_method, consent_version, categories_json, gpc_active, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: ListConsentBySite :many
-- No method filter. Caller passes method='' or omits filtering. Splitting
-- this from ListConsentBySiteByMethod avoids sqlc generating positional
-- ?6 references when sqlc.arg(method) is reused, which the go-sqlite3
-- driver mis-binds (caused HTTP 500 on the Proofs admin page).
SELECT * FROM consent_records
WHERE site_id = ?
  AND created_at >= ?
  AND created_at <= ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListConsentBySiteByMethod :many
SELECT * FROM consent_records
WHERE site_id = ?
  AND created_at >= ?
  AND created_at <= ?
  AND consent_method = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountConsentBySite :one
SELECT COUNT(*) AS total FROM consent_records
WHERE site_id = ?
  AND created_at >= ?
  AND created_at <= ?;

-- name: CountConsentBySiteByMethod :one
SELECT COUNT(*) AS total FROM consent_records
WHERE site_id = ?
  AND created_at >= ?
  AND created_at <= ?
  AND consent_method = ?;

-- name: GetConsentByID :one
SELECT * FROM consent_records WHERE id = ? AND site_id = ?;

-- name: ConsentStatsBySite :one
SELECT
    COUNT(*)                                                 AS total,
    SUM(CASE WHEN consent_method = 'accept-all' THEN 1 ELSE 0 END) AS accepts,
    SUM(CASE WHEN consent_method = 'reject-all' THEN 1 ELSE 0 END) AS rejects,
    SUM(CASE WHEN consent_method = 'custom'     THEN 1 ELSE 0 END) AS customs,
    SUM(CASE WHEN consent_method = 'gpc'        THEN 1 ELSE 0 END) AS gpcs,
    SUM(CASE WHEN consent_method = 'dns'        THEN 1 ELSE 0 END) AS dns_count,
    SUM(CASE WHEN consent_method = 'do-not-sell' THEN 1 ELSE 0 END) AS do_not_sell_count
FROM consent_records
WHERE site_id = ? AND created_at >= ? AND created_at <= ?;

-- name: ConsentDailyBySite :many
SELECT
    date(created_at_iso) AS day,
    consent_method,
    COUNT(*)             AS n
FROM consent_records
WHERE site_id = ? AND created_at >= ? AND created_at <= ?
GROUP BY day, consent_method
ORDER BY day ASC;

-- name: StreamConsentBySite :many
SELECT * FROM consent_records
WHERE site_id = ?
  AND created_at >= ?
  AND created_at <= ?
ORDER BY created_at ASC;

-- name: DeleteConsentOlderThan :exec
DELETE FROM consent_records WHERE created_at < ?;

-- name: DeleteConsentBySiteOlderThan :execrows
-- Per-site retention purge for the GDPR proof log. created_at is unix-ms;
-- caller passes the cutoff. Returns the affected row count so the
-- retention manager can log per-site stats.
DELETE FROM consent_records WHERE site_id = ? AND created_at < ?;

-- name: GetConsentSalt :one
SELECT salt FROM consent_salts WHERE day_utc = ?;

-- name: UpsertConsentSalt :exec
INSERT INTO consent_salts (day_utc, salt) VALUES (?, ?)
ON CONFLICT(day_utc) DO NOTHING;

-- name: DeleteConsentSaltsOlderThan :exec
DELETE FROM consent_salts WHERE day_utc < ?;
