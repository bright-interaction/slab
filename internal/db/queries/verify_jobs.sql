-- name: CreateVerifyJob :exec
INSERT INTO verify_jobs
    (id, site_id, migration_id, status, total_urls, deployed_domain, created_at)
VALUES (?, ?, ?, 'queued', ?, ?, datetime('now'));

-- name: GetVerifyJob :one
SELECT * FROM verify_jobs WHERE id = ?;

-- name: ListVerifyJobsByMigration :many
SELECT * FROM verify_jobs
WHERE migration_id = ?
ORDER BY created_at DESC;

-- name: MarkVerifyJobRunning :exec
UPDATE verify_jobs
SET status = 'running', started_at = datetime('now')
WHERE id = ? AND status = 'queued';

-- name: UpdateVerifyJobProgress :exec
UPDATE verify_jobs
SET processed_urls = ?, ok_count = ?, fail_count = ?
WHERE id = ?;

-- name: FinishVerifyJob :exec
UPDATE verify_jobs
SET status = ?,
    processed_urls = ?,
    ok_count = ?,
    fail_count = ?,
    error = ?,
    completed_at = datetime('now')
WHERE id = ?;

-- name: ReapStaleVerifyJobs :exec
UPDATE verify_jobs
SET status = 'failed',
    error = 'server restart',
    completed_at = datetime('now')
WHERE status IN ('queued', 'running');

-- name: CountVerifyJobsByStatus :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN status = 'queued' THEN 1 ELSE 0 END), 0) AS INTEGER) AS queued_count,
    CAST(COALESCE(SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END), 0) AS INTEGER) AS running_count,
    CAST(COALESCE(SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), 0) AS INTEGER) AS done_count,
    CAST(COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS INTEGER) AS failed_count,
    CAST(COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0) AS INTEGER) AS cancelled_count
FROM verify_jobs
WHERE migration_id = ?;
