-- name: GetFeedTokenByToken :one
SELECT * FROM feed_tokens WHERE token = $1 AND revoked_at IS NULL;

-- name: ListFeedTokensByUser :many
SELECT * FROM feed_tokens WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreateFeedToken :one
INSERT INTO feed_tokens (user_id, scope, scope_id, token)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: RevokeFeedToken :exec
UPDATE feed_tokens SET revoked_at = now() WHERE id = $1;
