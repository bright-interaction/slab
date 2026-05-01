-- name: AddSiteMember :exec
INSERT INTO site_members (site_id, user_id, role) VALUES (?, ?, ?)
ON CONFLICT(site_id, user_id) DO UPDATE SET role = excluded.role;

-- name: RemoveSiteMember :exec
DELETE FROM site_members WHERE site_id = ? AND user_id = ?;

-- name: GetSiteMembership :one
SELECT site_id, user_id, role, created_at
FROM site_members WHERE site_id = ? AND user_id = ?;

-- name: ListSiteMembers :many
SELECT sm.site_id, sm.user_id, sm.role, sm.created_at,
       u.email, u.name
FROM site_members sm
JOIN users u ON u.id = sm.user_id
WHERE sm.site_id = ?
ORDER BY sm.created_at ASC;

-- name: ListSiteIDsForUser :many
SELECT site_id FROM site_members WHERE user_id = ?;

-- name: BackfillSiteMembersForAdmin :exec
-- One-shot bootstrap: grant the admin ownership of every existing site
-- that currently has zero member rows. Idempotent: once a site has any
-- member, this becomes a no-op for that site. Preserves the legacy
-- single-workspace behaviour through the C1 migration without a
-- separate data migration script.
INSERT INTO site_members (site_id, user_id, role)
SELECT s.id, ?, 'owner' FROM sites s
WHERE NOT EXISTS (SELECT 1 FROM site_members WHERE site_id = s.id);
