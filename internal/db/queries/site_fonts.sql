-- name: CreateSiteFont :exec
INSERT INTO site_fonts (id, site_id, family_name, weight, style, file_path, file_size, original_name)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListSiteFonts :many
SELECT * FROM site_fonts
WHERE site_id = ?
ORDER BY family_name ASC, weight ASC, style ASC;

-- name: GetSiteFont :one
SELECT * FROM site_fonts
WHERE id = ? AND site_id = ?;

-- name: DeleteSiteFont :exec
DELETE FROM site_fonts WHERE id = ? AND site_id = ?;

-- name: ListDistinctFontFamilies :many
SELECT DISTINCT family_name FROM site_fonts
WHERE site_id = ?
ORDER BY family_name ASC;
