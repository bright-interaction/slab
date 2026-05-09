-- name: CreateWebhookSubscription :exec
INSERT INTO webhook_subscriptions (id, site_id, target_url, secret, event_types_json, active)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListWebhookSubscriptionsBySite :many
SELECT id, site_id, target_url, secret, event_types_json, active, last_delivered_at, last_error, created_at, updated_at
FROM webhook_subscriptions
WHERE site_id = ?
ORDER BY created_at DESC;

-- name: GetWebhookSubscription :one
SELECT id, site_id, target_url, secret, event_types_json, active, last_delivered_at, last_error, created_at, updated_at
FROM webhook_subscriptions
WHERE id = ?;

-- name: ListActiveWebhookSubscriptions :many
-- Used by the Emit hook to find every subscription on a site that
-- should receive a given event. Filtering by event_types_json is
-- done in Go because SQLite's JSON1 module isn't a hard dependency.
SELECT id, site_id, target_url, secret, event_types_json, active, last_delivered_at, last_error, created_at, updated_at
FROM webhook_subscriptions
WHERE site_id = ? AND active = 1;

-- name: UpdateWebhookSubscription :exec
UPDATE webhook_subscriptions
SET target_url = ?, event_types_json = ?, active = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateWebhookSubscriptionStatus :exec
UPDATE webhook_subscriptions
SET last_delivered_at = ?, last_error = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteWebhookSubscription :exec
DELETE FROM webhook_subscriptions WHERE id = ?;

-- name: CreateWebhookDelivery :exec
INSERT INTO webhook_deliveries (id, subscription_id, site_id, event_type, payload_json)
VALUES (?, ?, ?, ?, ?);

-- name: ListPendingWebhookDeliveries :many
-- Worker pull: picks up to N rows that are due for delivery (status
-- pending or retrying AND next_attempt_at <= now). The status filter
-- skips successful + dropped + in-flight rows.
SELECT id, subscription_id, site_id, event_type, payload_json, status, attempts, next_attempt_at, last_response_status, last_error, created_at, completed_at
FROM webhook_deliveries
WHERE status IN ('pending', 'retrying') AND next_attempt_at <= ?
ORDER BY next_attempt_at ASC
LIMIT ?;

-- name: ListWebhookDeliveriesBySite :many
SELECT id, subscription_id, site_id, event_type, payload_json, status, attempts, next_attempt_at, last_response_status, last_error, created_at, completed_at
FROM webhook_deliveries
WHERE site_id = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: MarkWebhookDeliverySucceeded :exec
UPDATE webhook_deliveries
SET status = 'success', attempts = attempts + 1, last_response_status = ?, last_error = '', completed_at = datetime('now')
WHERE id = ?;

-- name: MarkWebhookDeliveryRetrying :exec
UPDATE webhook_deliveries
SET status = 'retrying', attempts = attempts + 1, last_response_status = ?, last_error = ?, next_attempt_at = ?
WHERE id = ?;

-- name: MarkWebhookDeliveryDropped :exec
UPDATE webhook_deliveries
SET status = 'dropped', attempts = attempts + 1, last_response_status = ?, last_error = ?, completed_at = datetime('now')
WHERE id = ?;
