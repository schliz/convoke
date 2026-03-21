-- name: DeleteUserIDPGroups :exec
DELETE FROM user_idp_groups WHERE user_id = $1;

-- name: InsertUserIDPGroup :exec
INSERT INTO user_idp_groups (user_id, group_name) VALUES ($1, $2);

-- name: GetUserIDPGroups :many
SELECT group_name FROM user_idp_groups WHERE user_id = $1;

-- name: IsUserInGroup :one
SELECT EXISTS(
    SELECT 1 FROM user_idp_groups
    WHERE user_id = $1 AND group_name = $2
) AS is_member;
