-- name: ListDeployTargetsBySite :many
SELECT * FROM deploy_targets WHERE site_id = ? ORDER BY is_default DESC, name ASC;

-- name: GetDeployTarget :one
SELECT * FROM deploy_targets WHERE id = ?;

-- name: CreateDeployTarget :exec
INSERT INTO deploy_targets (id, site_id, name, kind, config_json, is_default)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateDeployTarget :exec
UPDATE deploy_targets SET
    name = ?,
    kind = ?,
    config_json = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteDeployTarget :exec
DELETE FROM deploy_targets WHERE id = ?;

-- name: ClearDefaultDeployTargets :exec
UPDATE deploy_targets SET is_default = 0, updated_at = datetime('now')
WHERE site_id = ?;

-- name: SetDeployTargetDefault :exec
UPDATE deploy_targets SET is_default = 1, updated_at = datetime('now')
WHERE id = ?;
