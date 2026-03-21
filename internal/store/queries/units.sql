-- name: GetUnitByID :one
SELECT * FROM units WHERE id = $1;

-- name: GetUnitBySlug :one
SELECT * FROM units WHERE slug = $1;

-- name: ListUnits :many
SELECT * FROM units ORDER BY name;

-- name: CreateUnit :one
INSERT INTO units (name, slug, description, logo_path, contact_email, admin_group)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateUnit :one
UPDATE units SET
    name = $2,
    description = $3,
    logo_path = $4,
    contact_email = $5,
    admin_group = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteUnit :exec
DELETE FROM units WHERE id = $1;
