-- name: CreateOrder :exec
INSERT INTO orders (
    id, site_id, order_number, status,
    customer_name, customer_email, customer_phone,
    shipping_address_json, billing_address_json,
    subtotal_cents, discount_cents, shipping_cents, tax_cents, total_cents,
    currency, discount_code_id,
    payment_provider, payment_id, payment_status, payment_checkout_url,
    notes, metadata_json
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = ? AND site_id = ?;

-- name: GetOrderByOrderNumber :one
SELECT * FROM orders WHERE site_id = ? AND order_number = ?;

-- name: GetOrderByPaymentID :one
SELECT * FROM orders WHERE payment_id = ? LIMIT 1;

-- name: ListOrdersBySite :many
SELECT * FROM orders
WHERE site_id = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: ListOrdersBySiteStatus :many
SELECT * FROM orders
WHERE site_id = ? AND status = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: CountOrdersBySite :one
SELECT CAST(COUNT(*) AS INTEGER) FROM orders WHERE site_id = ?;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = ?,
    payment_status = ?,
    paid_at = CASE WHEN ? = 'paid' AND paid_at = '' THEN datetime('now') ELSE paid_at END,
    fulfilled_at = CASE WHEN ? = 'fulfilled' AND fulfilled_at = '' THEN datetime('now') ELSE fulfilled_at END,
    cancelled_at = CASE WHEN ? = 'cancelled' AND cancelled_at = '' THEN datetime('now') ELSE cancelled_at END,
    refunded_at = CASE WHEN ? = 'refunded' AND refunded_at = '' THEN datetime('now') ELSE refunded_at END,
    updated_at = datetime('now')
WHERE id = ? AND site_id = ?;

-- name: UpdateOrderRefundID :exec
UPDATE orders
SET refund_id = ?,
    updated_at = datetime('now')
WHERE id = ? AND site_id = ?;

-- name: UpdateOrderNotes :exec
UPDATE orders
SET notes = ?,
    updated_at = datetime('now')
WHERE id = ? AND site_id = ?;

-- name: UpdateOrderPayment :exec
UPDATE orders
SET payment_id = ?,
    payment_checkout_url = ?,
    updated_at = datetime('now')
WHERE id = ? AND site_id = ?;
