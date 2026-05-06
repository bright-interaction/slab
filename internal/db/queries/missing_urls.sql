-- name: UpsertMissingURL :exec
INSERT INTO missing_urls (id, site_id, path, referer, hit_count, first_seen, last_seen)
VALUES (?, ?, ?, ?, 1, datetime('now'), datetime('now'))
ON CONFLICT(site_id, path) DO UPDATE
  SET hit_count = hit_count + 1,
      last_seen = datetime('now'),
      referer = CASE WHEN excluded.referer != '' THEN excluded.referer ELSE missing_urls.referer END;

-- name: ListMissingURLsBySite :many
SELECT * FROM missing_urls
WHERE site_id = ?
ORDER BY hit_count DESC, last_seen DESC
LIMIT ?;

-- name: GetMissingURLByID :one
SELECT * FROM missing_urls WHERE id = ?;

-- name: GetMissingURLByPath :one
SELECT * FROM missing_urls WHERE site_id = ? AND path = ?;

-- name: DeleteMissingURL :exec
DELETE FROM missing_urls WHERE id = ?;

-- name: DeleteMissingURLByPath :exec
DELETE FROM missing_urls WHERE site_id = ? AND path = ?;

-- name: CountMissingURLsBySite :one
SELECT COUNT(*) FROM missing_urls WHERE site_id = ?;
