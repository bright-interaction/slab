-- name: WriteAuditLog :exec
INSERT INTO audit_log (
    id, actor_user_id, actor_role, actor_ip,
    site_id, action, resource_type, resource_id, changes_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAuditLogBySite :many
SELECT id, actor_user_id, actor_role, actor_ip,
       site_id, action, resource_type, resource_id, changes_json, created_at
FROM audit_log
WHERE site_id = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: ListAuditLogGlobal :many
-- Used by the workspace-admin "everything that happened" feed. site_id
-- can be empty for non-site-scoped actions (member invites, secret
-- rotations).
SELECT id, actor_user_id, actor_role, actor_ip,
       site_id, action, resource_type, resource_id, changes_json, created_at
FROM audit_log
ORDER BY created_at DESC
LIMIT ?;

-- name: ListAuditLogByResource :many
SELECT id, actor_user_id, actor_role, actor_ip,
       site_id, action, resource_type, resource_id, changes_json, created_at
FROM audit_log
WHERE resource_type = ? AND resource_id = ?
ORDER BY created_at DESC
LIMIT ?;
