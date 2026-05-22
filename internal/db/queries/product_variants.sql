-- name: CreateProductVariant :exec
INSERT INTO product_variants (id, product_id, sku, name, price_cents, compare_at_price_cents, inventory_count, allow_backorder, sort_order, attributes_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetProductVariantByID :one
SELECT * FROM product_variants WHERE id = ?;

-- name: ListProductVariants :many
SELECT * FROM product_variants
WHERE product_id = ?
ORDER BY sort_order ASC, created_at ASC;

-- name: UpdateProductVariant :exec
UPDATE product_variants
SET sku = ?,
    name = ?,
    price_cents = ?,
    compare_at_price_cents = ?,
    inventory_count = ?,
    allow_backorder = ?,
    sort_order = ?,
    attributes_json = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: SetVariantInventoryCount :exec
UPDATE product_variants
SET inventory_count = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: AdjustVariantInventoryCount :exec
UPDATE product_variants
SET inventory_count = inventory_count + ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteProductVariant :exec
DELETE FROM product_variants WHERE id = ?;
