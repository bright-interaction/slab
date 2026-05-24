-- name: InsertCWVEvent :exec
INSERT INTO cwv_events (id, site_id, page_path, metric, value, rating, device)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListCWVValuesForWindow :many
-- Returns every value for one (site, metric, device) over the last
-- N days. The dashboard handler computes p75 in Go (SQLite has no
-- native percentile). N is passed via datetime('now', ?); the
-- callsite formats e.g. '-7 days'.
SELECT value, rating, page_path, created_at
FROM cwv_events
WHERE site_id = ?
  AND metric = ?
  AND (device = ? OR ? = '')
  AND created_at >= datetime('now', ?)
ORDER BY value ASC;

-- name: CountCWVForSite :one
SELECT COUNT(*) FROM cwv_events WHERE site_id = ?;

-- name: DeleteCWVBySiteOlderThan :execrows
-- Per-site retention sweep. The retention manager passes an ISO
-- timestamp cutoff and gets back the number of rows deleted so the
-- sweep result struct can report per-table totals.
DELETE FROM cwv_events
WHERE site_id = ? AND created_at < ?;
