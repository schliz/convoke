-- name: GetCalendarByID :one
SELECT * FROM calendars WHERE id = $1;

-- name: GetCalendarBySlug :one
SELECT * FROM calendars WHERE slug = $1;

-- name: ListCalendarsByUnit :many
SELECT * FROM calendars WHERE unit_id = $1 ORDER BY sort_order, name;

-- name: CreateCalendar :one
INSERT INTO calendars (slug, unit_id, name, creation_policy, visibility, participation, participant_visibility, color, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateCalendar :one
UPDATE calendars SET
    name = $2,
    creation_policy = $3,
    visibility = $4,
    participation = $5,
    participant_visibility = $6,
    color = $7,
    sort_order = $8,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteCalendar :exec
DELETE FROM calendars WHERE id = $1;

-- name: ListVisibleCalendarsForUser :many
SELECT DISTINCT c.id, c.slug, c.unit_id, c.name, c.creation_policy, c.visibility,
    c.participation, c.participant_visibility, c.color, c.sort_order, c.created_at, c.updated_at
FROM calendars c
WHERE c.visibility = 'association'
   OR (c.visibility = 'unit' AND EXISTS(
       SELECT 1 FROM unit_group_bindings ugb
       JOIN user_idp_groups uig ON ugb.group_name = uig.group_name
       WHERE ugb.unit_id = c.unit_id AND uig.user_id = $1
   ))
   OR (c.visibility = 'custom' AND EXISTS(
       SELECT 1 FROM calendar_custom_viewers ccv
       JOIN unit_group_bindings ugb ON ccv.unit_id = ugb.unit_id
       JOIN user_idp_groups uig ON ugb.group_name = uig.group_name
       WHERE ccv.calendar_id = c.id AND uig.user_id = $1
   ))
ORDER BY c.sort_order, c.name;

-- name: GetCalendarWithUnit :one
SELECT c.id, c.slug, c.unit_id, c.name, c.creation_policy, c.visibility,
    c.participation, c.participant_visibility, c.color, c.sort_order,
    c.created_at, c.updated_at,
    u.name AS unit_name, u.slug AS unit_slug
FROM calendars c
JOIN units u ON c.unit_id = u.id
WHERE c.id = $1;

-- name: ListAllCalendars :many
SELECT * FROM calendars ORDER BY sort_order, name;

-- name: GetCustomViewerUnits :many
SELECT u.id, u.name, u.slug
FROM calendar_custom_viewers ccv
JOIN units u ON ccv.unit_id = u.id
WHERE ccv.calendar_id = $1
ORDER BY u.name;

-- name: DeleteCalendarCustomViewers :exec
DELETE FROM calendar_custom_viewers WHERE calendar_id = $1;

-- name: InsertCalendarCustomViewer :exec
INSERT INTO calendar_custom_viewers (calendar_id, unit_id) VALUES ($1, $2);
