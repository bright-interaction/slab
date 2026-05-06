-- name: AddWorkspaceMember :exec
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES (?, ?, ?)
ON CONFLICT(workspace_id, user_id) DO UPDATE SET role = excluded.role;

-- name: RemoveWorkspaceMember :exec
DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?;

-- name: GetWorkspaceMembership :one
SELECT workspace_id, user_id, role, created_at
FROM workspace_members
WHERE workspace_id = ? AND user_id = ?;

-- name: ListWorkspaceMembers :many
SELECT wm.workspace_id, wm.user_id, wm.role, wm.created_at,
       u.email, u.name
FROM workspace_members wm
JOIN users u ON u.id = wm.user_id
WHERE wm.workspace_id = ?
ORDER BY wm.created_at ASC;

-- name: ListWorkspaceIDsForUser :many
SELECT workspace_id FROM workspace_members WHERE user_id = ?;

-- name: BackfillWorkspaceMembersForAdmin :exec
-- Bootstrap helper: grant the admin owner-role on every existing
-- workspace that has zero member rows. Idempotent: once any member
-- exists, the workspace is skipped. Used by ensureDefaultWorkspace
-- on first boot of a Phase 30+ binary against an existing DB.
INSERT INTO workspace_members (workspace_id, user_id, role)
SELECT w.id, ?, 'owner' FROM workspaces w
WHERE NOT EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id = w.id);
