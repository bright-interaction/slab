-- name: CreateMigrationVerification :exec
INSERT INTO migration_verifications
    (id, site_id, migration_id, source_url, status_code, final_url, hops, ok, error, checked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'));

-- name: ListMigrationVerifications :many
SELECT * FROM migration_verifications
WHERE migration_id = ?
ORDER BY checked_at DESC, source_url;

-- name: ListMigrationVerificationsBySite :many
SELECT * FROM migration_verifications
WHERE site_id = ?
ORDER BY checked_at DESC
LIMIT ?;

-- name: DeleteMigrationVerificationsByMigration :exec
DELETE FROM migration_verifications WHERE migration_id = ?;

-- name: CountMigrationVerificationsByOK :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN ok = 1 THEN 1 ELSE 0 END), 0) AS INTEGER) AS ok_count,
    CAST(COALESCE(SUM(CASE WHEN ok = 0 THEN 1 ELSE 0 END), 0) AS INTEGER) AS fail_count
FROM migration_verifications
WHERE migration_id = ?;
