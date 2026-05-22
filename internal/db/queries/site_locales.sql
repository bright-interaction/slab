-- name: ListSiteLocales :many
SELECT * FROM site_locales WHERE site_id = ? ORDER BY is_default DESC, sort_order ASC, locale ASC;

-- name: GetSiteLocale :one
SELECT * FROM site_locales WHERE site_id = ? AND locale = ?;

-- name: GetDefaultSiteLocale :one
SELECT * FROM site_locales WHERE site_id = ? AND is_default = 1 LIMIT 1;

-- name: CountSiteLocales :one
SELECT COUNT(*) FROM site_locales WHERE site_id = ?;

-- name: UpsertSiteLocale :exec
INSERT INTO site_locales (site_id, locale, is_default, sort_order)
VALUES (?, ?, ?, ?)
ON CONFLICT(site_id, locale) DO UPDATE SET
    is_default = excluded.is_default,
    sort_order = excluded.sort_order,
    updated_at = datetime('now');

-- name: DeleteSiteLocale :exec
DELETE FROM site_locales WHERE site_id = ? AND locale = ?;

-- name: ClearDefaultSiteLocales :exec
UPDATE site_locales SET is_default = 0, updated_at = datetime('now')
WHERE site_id = ? AND locale != ?;

-- name: SetDefaultSiteLocale :exec
UPDATE site_locales SET is_default = 1, updated_at = datetime('now')
WHERE site_id = ? AND locale = ?;
