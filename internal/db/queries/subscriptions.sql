-- name: CreateSubscription :exec
INSERT INTO subscriptions (id, workspace_id, provider, external_id, external_customer_id,
                           plan, status, amount_cents, currency, interval_unit, interval_count,
                           current_period_end, cancel_at, metadata_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSubscriptionByID :one
SELECT id, workspace_id, provider, external_id, external_customer_id,
       plan, status, amount_cents, currency, interval_unit, interval_count,
       current_period_end, cancel_at, metadata_json, created_at, updated_at
FROM subscriptions WHERE id = ?;

-- name: GetSubscriptionByWorkspace :one
SELECT id, workspace_id, provider, external_id, external_customer_id,
       plan, status, amount_cents, currency, interval_unit, interval_count,
       current_period_end, cancel_at, metadata_json, created_at, updated_at
FROM subscriptions
WHERE workspace_id = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: GetSubscriptionByExternalID :one
SELECT id, workspace_id, provider, external_id, external_customer_id,
       plan, status, amount_cents, currency, interval_unit, interval_count,
       current_period_end, cancel_at, metadata_json, created_at, updated_at
FROM subscriptions WHERE external_id = ?;

-- name: ListSubscriptionsBySite :many
SELECT id, workspace_id, provider, external_id, external_customer_id,
       plan, status, amount_cents, currency, interval_unit, interval_count,
       current_period_end, cancel_at, metadata_json, created_at, updated_at
FROM subscriptions
ORDER BY created_at DESC
LIMIT 1000;

-- name: UpdateSubscriptionStatus :exec
UPDATE subscriptions
SET status = ?, current_period_end = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateSubscriptionExternal :exec
UPDATE subscriptions
SET external_id = ?, external_customer_id = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions WHERE id = ?;
