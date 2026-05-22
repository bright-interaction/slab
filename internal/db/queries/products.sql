-- name: CreateProduct :exec
INSERT INTO products (id, site_id, name, slug, description, category, status, images_json, base_price_cents, currency, requires_shipping, tax_class, weight_grams, sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetProductByID :one
SELECT * FROM products WHERE id = ? AND site_id = ?;

-- name: GetProductBySlug :one
SELECT * FROM products WHERE site_id = ? AND slug = ?;

-- name: ListProductsBySite :many
SELECT * FROM products
WHERE site_id = ?
ORDER BY sort_order ASC, created_at DESC
LIMIT ?;

-- name: ListActiveProductsBySite :many
SELECT * FROM products
WHERE site_id = ? AND status = 'active'
ORDER BY sort_order ASC, created_at DESC
LIMIT ?;

-- name: UpdateProduct :exec
UPDATE products
SET name = ?,
    slug = ?,
    description = ?,
    category = ?,
    status = ?,
    images_json = ?,
    base_price_cents = ?,
    currency = ?,
    requires_shipping = ?,
    tax_class = ?,
    weight_grams = ?,
    sort_order = ?,
    updated_at = datetime('now')
WHERE id = ? AND site_id = ?;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = ? AND site_id = ?;

-- name: CountProductsBySite :one
SELECT CAST(COUNT(*) AS INTEGER) FROM products WHERE site_id = ?;
