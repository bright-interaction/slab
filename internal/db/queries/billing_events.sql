-- name: RecordBillingEvent :exec
-- UNIQUE(provider, external_event_id) makes duplicate inserts no-op
-- via INSERT OR IGNORE. Idempotency at the schema layer means handlers
-- don't need a separate "have we seen this id" check.
INSERT OR IGNORE INTO billing_events (id, workspace_id, provider, external_event_id, event_type, payload_json)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetBillingEventByExternalID :one
SELECT id, workspace_id, provider, external_event_id, event_type,
       payload_json, processed_at, error, created_at
FROM billing_events
WHERE provider = ? AND external_event_id = ?;

-- name: MarkBillingEventProcessed :exec
UPDATE billing_events
SET processed_at = datetime('now'), error = ?
WHERE id = ?;

-- name: ListUnprocessedBillingEvents :many
SELECT id, workspace_id, provider, external_event_id, event_type,
       payload_json, processed_at, error, created_at
FROM billing_events
WHERE processed_at = ''
ORDER BY created_at ASC
LIMIT 100;

-- name: PurgeOldBillingEvents :exec
-- Run by retention.Manager. 30-day retention; processed events older
-- than the cutoff are dropped. Unprocessed events stay forever so the
-- operator can investigate.
DELETE FROM billing_events
WHERE processed_at != '' AND processed_at < ?;
