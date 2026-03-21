-- name: ListWebhooksByUnit :many
SELECT * FROM webhooks WHERE unit_id = $1 AND enabled = true;

-- name: ListAssociationWebhooks :many
SELECT * FROM webhooks WHERE unit_id IS NULL AND enabled = true;

-- name: CreateWebhook :one
INSERT INTO webhooks (unit_id, name, url, secret, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateWebhook :one
UPDATE webhooks SET
    name = $2,
    url = $3,
    secret = $4,
    enabled = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhooks WHERE id = $1;
