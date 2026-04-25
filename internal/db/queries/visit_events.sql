-- name: RecordVisitEvent :exec
INSERT INTO visit_events (id, site_id, fingerprint, session_id, path, referer, status, ms, ts)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListVisitsBySite :many
SELECT * FROM visit_events
WHERE site_id = ?
ORDER BY ts DESC
LIMIT ? OFFSET ?;

-- name: CountVisitsBySite :one
SELECT COUNT(*) FROM visit_events
WHERE site_id = ? AND ts >= ?;

-- name: CountVisitsByPath :many
SELECT path, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ?
GROUP BY path
ORDER BY count DESC;

-- name: TopReferers :many
SELECT referer, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ? AND referer != ''
GROUP BY referer
ORDER BY count DESC
LIMIT ?;
