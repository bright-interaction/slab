-- name: RecordVisitEvent :exec
INSERT INTO visit_events (
    id, site_id, fingerprint, session_id, path, referer, status, ms, ts,
    browser, os, device, country, lang, utm_source, utm_medium, utm_campaign
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListVisitsBySite :many
SELECT * FROM visit_events
WHERE site_id = ?
ORDER BY ts DESC
LIMIT ? OFFSET ?;

-- name: CountVisitsBySite :one
SELECT COUNT(*) FROM visit_events
WHERE site_id = ? AND ts >= ?;

-- name: CountUniqueVisitorsSince :one
SELECT COUNT(DISTINCT fingerprint) FROM visit_events
WHERE site_id = ? AND ts >= ?;

-- name: CountLiveVisitorsSince :one
-- Distinct fingerprints in the last N minutes (caller passes the cutoff
-- timestamp). Used for the "live now" widget.
SELECT COUNT(DISTINCT fingerprint) FROM visit_events
WHERE site_id = ? AND ts >= ?;

-- name: CountVisitsByPath :many
SELECT path, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ?
GROUP BY path
ORDER BY count DESC
LIMIT ?;

-- name: TopReferers :many
SELECT referer, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ? AND referer != ''
GROUP BY referer
ORDER BY count DESC
LIMIT ?;

-- name: TopBrowsers :many
SELECT browser AS name, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ? AND browser != ''
GROUP BY browser
ORDER BY count DESC
LIMIT ?;

-- name: TopOS :many
SELECT os AS name, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ? AND os != ''
GROUP BY os
ORDER BY count DESC
LIMIT ?;

-- name: TopDevices :many
SELECT device AS name, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ? AND device != ''
GROUP BY device
ORDER BY count DESC
LIMIT ?;

-- name: TopCountries :many
SELECT country AS name, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ? AND country != ''
GROUP BY country
ORDER BY count DESC
LIMIT ?;

-- name: TopLanguages :many
SELECT lang AS name, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ? AND lang != ''
GROUP BY lang
ORDER BY count DESC
LIMIT ?;

-- name: TopUTMSources :many
SELECT utm_source AS name, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ? AND utm_source != ''
GROUP BY utm_source
ORDER BY count DESC
LIMIT ?;

-- name: TopUTMCampaigns :many
SELECT utm_campaign AS name, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ? AND utm_campaign != ''
GROUP BY utm_campaign
ORDER BY count DESC
LIMIT ?;

-- name: TopStatuses :many
SELECT status, COUNT(*) AS count FROM visit_events
WHERE site_id = ? AND ts >= ?
GROUP BY status
ORDER BY count DESC;

-- name: PageviewsTimeSeriesDaily :many
-- Pageviews per UTC day for the requested window. Returned as ISO-date
-- buckets so the frontend can render a sparkline / bar chart directly.
SELECT
    substr(ts, 1, 10) AS day,
    COUNT(*) AS count,
    COUNT(DISTINCT fingerprint) AS unique_count
FROM visit_events
WHERE site_id = ? AND ts >= ?
GROUP BY day
ORDER BY day ASC;

-- name: PageviewsTimeSeriesHourly :many
-- Pageviews per UTC hour for the last day window. Used when range = 1d.
SELECT
    substr(ts, 1, 13) AS hour,
    COUNT(*) AS count,
    COUNT(DISTINCT fingerprint) AS unique_count
FROM visit_events
WHERE site_id = ? AND ts >= ?
GROUP BY hour
ORDER BY hour ASC;

-- name: DeleteVisitEventsBySiteOlderThan :execrows
-- Per-site retention purge. ts is RFC3339 (UTC). Caller passes the cutoff;
-- everything strictly older gets dropped. The (site_id, ts) index keeps this
-- O(rows-deleted) instead of a full scan.
DELETE FROM visit_events WHERE site_id = ? AND ts < ?;

-- name: CountVisitEventsBySiteOlderThan :one
-- Pre-flight count for the retention manager so the slog line can name how
-- many rows the upcoming DELETE will drop without re-running the query.
SELECT COUNT(*) FROM visit_events WHERE site_id = ? AND ts < ?;

-- name: ConversionPathsForIdentified :many
-- For every identified session, return its full path history ordered by ts.
-- The handler groups by session_id / fingerprint to assemble the journey.
SELECT
    ve.fingerprint,
    ve.path,
    ve.ts,
    vs.email,
    vs.identified_at
FROM visit_events ve
JOIN visit_sessions vs
  ON vs.site_id = ve.site_id AND vs.fingerprint = ve.fingerprint
WHERE ve.site_id = ?
  AND vs.identified_at != ''
  AND ve.ts <= vs.identified_at
ORDER BY vs.identified_at DESC, ve.ts ASC
LIMIT ?;
