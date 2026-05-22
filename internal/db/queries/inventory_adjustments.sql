-- name: CreateInventoryAdjustment :exec
INSERT INTO inventory_adjustments (id, variant_id, site_id, delta, new_count, reason, note, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListInventoryAdjustmentsByVariant :many
SELECT * FROM inventory_adjustments
WHERE variant_id = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: ListInventoryAdjustmentsBySite :many
SELECT * FROM inventory_adjustments
WHERE site_id = ?
ORDER BY created_at DESC
LIMIT ?;
