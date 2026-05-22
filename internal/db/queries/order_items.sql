-- name: CreateOrderItem :exec
INSERT INTO order_items (id, order_id, variant_id, product_id, product_name, variant_name, sku, quantity, unit_price_cents, total_cents)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListOrderItems :many
SELECT * FROM order_items
WHERE order_id = ?
ORDER BY created_at ASC;
