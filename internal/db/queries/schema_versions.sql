-- name: RecordSchemaVersion :exec
INSERT OR IGNORE INTO schema_versions (version, note) VALUES (?, ?);

-- name: ListSchemaVersions :many
SELECT version, applied_at, note
FROM schema_versions
ORDER BY version ASC;

-- name: GetMaxSchemaVersion :one
SELECT COALESCE(MAX(version), 0) FROM schema_versions;
