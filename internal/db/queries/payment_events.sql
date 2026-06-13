-- name: CreatePaymentEvent :exec
INSERT OR IGNORE INTO payment_events (id, site_id, order_id, provider, payment_id, event_type, raw_json, processed)
VALUES (?, ?, ?, ?, ?, ?, ?, 0);

-- name: MarkPaymentEventProcessed :execrows
UPDATE payment_events
SET processed = 1
WHERE provider = ? AND payment_id = ? AND event_type = ? AND processed = 0;

-- name: GetPaymentEventByLookup :one
SELECT * FROM payment_events
WHERE provider = ? AND payment_id = ? AND event_type = ?;

-- name: ListPaymentEventsByOrder :many
SELECT * FROM payment_events
WHERE order_id = ?
ORDER BY created_at DESC
LIMIT ?;
