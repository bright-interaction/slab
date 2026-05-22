-- name: ListPageLocalesByPage :many
SELECT * FROM page_locales WHERE page_id = ? ORDER BY locale ASC;

-- name: ListPageLocalesBySite :many
SELECT pl.* FROM page_locales pl
JOIN pages p ON p.id = pl.page_id
WHERE p.site_id = ?
ORDER BY p.slug ASC, pl.locale ASC;

-- name: GetPageLocale :one
SELECT * FROM page_locales WHERE page_id = ? AND locale = ?;

-- name: UpsertPageLocale :exec
INSERT INTO page_locales (page_id, locale, slug_override, title, meta_title, meta_description, status)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(page_id, locale) DO UPDATE SET
    slug_override = excluded.slug_override,
    title = excluded.title,
    meta_title = excluded.meta_title,
    meta_description = excluded.meta_description,
    status = excluded.status,
    updated_at = datetime('now');

-- name: DeletePageLocale :exec
DELETE FROM page_locales WHERE page_id = ? AND locale = ?;

-- name: DeletePageLocalesByLocale :exec
DELETE FROM page_locales WHERE page_id IN (
    SELECT id FROM pages WHERE site_id = ?
) AND locale = ?;
