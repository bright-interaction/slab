-- name: CreateDesignReference :exec
INSERT INTO design_references (id, site_id, url, label, ref_type, fetched_json, fetched_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListDesignReferences :many
SELECT * FROM design_references
WHERE site_id = ?
ORDER BY created_at DESC;

-- name: GetDesignReference :one
SELECT * FROM design_references
WHERE id = ? AND site_id = ?;

-- name: UpdateDesignReferenceFetched :exec
UPDATE design_references SET
    fetched_json = ?,
    fetched_at = ?
WHERE id = ? AND site_id = ?;

-- name: DeleteDesignReference :exec
DELETE FROM design_references WHERE id = ? AND site_id = ?;
