-- name: GetEventByID :one
SELECT * FROM events WHERE id = $1;

-- name: GetEventBySlug :one
SELECT * FROM events WHERE slug = $1;

-- name: ListEventsByUnit :many
SELECT * FROM events WHERE unit_id = $1 ORDER BY start_date DESC;

-- name: CreateEvent :one
INSERT INTO events (slug, unit_id, name, start_date, end_date, website, description)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateEvent :one
UPDATE events SET
    name = $2,
    start_date = $3,
    end_date = $4,
    website = $5,
    description = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events WHERE id = $1;

-- name: ListEventCalendars :many
SELECT c.* FROM calendars c
JOIN event_calendars ec ON c.id = ec.calendar_id
WHERE ec.event_id = $1
ORDER BY ec.sort_order, c.name;

-- name: AddCalendarToEvent :exec
INSERT INTO event_calendars (event_id, calendar_id, sort_order)
VALUES ($1, $2, $3)
ON CONFLICT (event_id, calendar_id) DO UPDATE SET sort_order = EXCLUDED.sort_order;

-- name: RemoveCalendarFromEvent :exec
DELETE FROM event_calendars WHERE event_id = $1 AND calendar_id = $2;
