-- name: GetNotificationConfigsByCalendar :many
SELECT * FROM notification_configs WHERE calendar_id = $1;

-- name: UpsertNotificationConfig :one
INSERT INTO notification_configs (calendar_id, event_type, enabled, lead_time)
VALUES ($1, $2, $3, $4)
ON CONFLICT (calendar_id, event_type) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    lead_time = EXCLUDED.lead_time
RETURNING *;

-- name: GetUserNotificationPreferences :many
SELECT * FROM user_notification_preferences WHERE user_id = $1;

-- name: UpsertUserNotificationPreference :one
INSERT INTO user_notification_preferences (user_id, event_type, channel, enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, event_type, channel) DO UPDATE SET enabled = EXCLUDED.enabled
RETURNING *;

-- name: CreateNotification :one
INSERT INTO notifications (user_id, entry_id, event_type, channel, payload)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListPendingNotifications :many
SELECT * FROM notifications
WHERE status IN ('pending', 'retrying')
ORDER BY created_at
LIMIT $1;

-- name: UpdateNotificationStatus :exec
UPDATE notifications SET
    status = $2,
    error = $3,
    retry_count = CASE WHEN $2 = 'retrying' THEN retry_count + 1 ELSE retry_count END,
    sent_at = CASE WHEN $2 = 'sent' THEN now() ELSE sent_at END
WHERE id = $1;
