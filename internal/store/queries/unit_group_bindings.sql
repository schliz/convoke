-- name: IsUserMemberOfUnit :one
SELECT EXISTS(
    SELECT 1 FROM user_idp_groups uig
    JOIN unit_group_bindings ugb ON uig.group_name = ugb.group_name
    WHERE uig.user_id = $1 AND ugb.unit_id = $2
) AS is_member;

-- name: IsUserAdminOfUnit :one
SELECT EXISTS(
    SELECT 1 FROM user_idp_groups uig
    JOIN units u ON uig.group_name = u.admin_group
    WHERE uig.user_id = $1 AND u.id = $2
) AS is_admin;

-- name: ListUnitsForUser :many
SELECT DISTINCT u.* FROM units u
JOIN unit_group_bindings ugb ON u.id = ugb.unit_id
JOIN user_idp_groups uig ON ugb.group_name = uig.group_name
WHERE uig.user_id = $1
ORDER BY u.name;

-- name: DeleteUnitGroupBindings :exec
DELETE FROM unit_group_bindings WHERE unit_id = $1;

-- name: InsertUnitGroupBinding :exec
INSERT INTO unit_group_bindings (unit_id, group_name) VALUES ($1, $2);

-- name: GetUnitGroupBindings :many
SELECT group_name FROM unit_group_bindings WHERE unit_id = $1;
