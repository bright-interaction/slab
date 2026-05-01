-- name: RecordVisitEngagement :exec
INSERT INTO visit_engagement (
    id, site_id, fingerprint, path, ts,
    screen_w, screen_h, viewport_w, viewport_h,
    prefers_dark, prefers_reduced_motion,
    time_on_page_ms, max_scroll_pct
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AvgEngagementSince :one
-- Aggregate metrics across all engagement beacons in a window. NULL-safe
-- via COALESCE so an empty window returns zeros instead of NaN.
SELECT
    COUNT(*)                                     AS samples,
    COALESCE(AVG(time_on_page_ms), 0)            AS avg_time_on_page_ms,
    COALESCE(AVG(max_scroll_pct), 0)             AS avg_scroll_pct,
    COALESCE(SUM(prefers_dark) * 100.0 / NULLIF(COUNT(*), 0), 0) AS dark_pct,
    COALESCE(SUM(prefers_reduced_motion) * 100.0 / NULLIF(COUNT(*), 0), 0) AS reduced_motion_pct
FROM visit_engagement
WHERE site_id = ? AND ts >= ?;

-- name: AvgEngagementByPath :many
-- Per-path engagement so the dashboard can show "users spend 2m12s on
-- /pricing but bounce at /blog/x in 8s". Capped at the busiest 10 paths.
SELECT
    path,
    COUNT(*)                                AS samples,
    COALESCE(AVG(time_on_page_ms), 0)       AS avg_time_on_page_ms,
    COALESCE(AVG(max_scroll_pct), 0)        AS avg_scroll_pct
FROM visit_engagement
WHERE site_id = ? AND ts >= ?
GROUP BY path
ORDER BY samples DESC
LIMIT ?;

-- name: TopViewports :many
-- Group by 100px-bucketed viewport width so we don't get a long tail of
-- one-off sizes. ~16 buckets cover the realistic phone-to-4k range.
SELECT
    (viewport_w / 100) * 100 AS bucket,
    COUNT(*) AS count
FROM visit_engagement
WHERE site_id = ? AND ts >= ? AND viewport_w > 0
GROUP BY bucket
ORDER BY count DESC
LIMIT ?;

-- name: DeleteVisitEngagementBySiteOlderThan :execrows
-- Per-site retention purge for engagement rows (screen size, time-on-page,
-- scroll depth). ts is RFC3339 UTC; caller passes the cutoff.
DELETE FROM visit_engagement WHERE site_id = ? AND ts < ?;
