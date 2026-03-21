-- name: GetEntryByID :one
SELECT * FROM entries WHERE id = $1;

-- name: GetEntryBySlug :one
SELECT * FROM entries WHERE slug = $1;

-- name: ListEntriesByCalendarAndDateRange :many
SELECT * FROM entries
WHERE calendar_id = $1
  AND starts_at >= $2
  AND starts_at < $3
ORDER BY starts_at;

-- name: ListEntriesByDateRange :many
SELECT * FROM entries
WHERE starts_at >= $1 AND starts_at < $2
ORDER BY starts_at;

-- name: ListEntriesByRecurrenceRule :many
SELECT * FROM entries WHERE recurrence_rule_id = $1 ORDER BY starts_at;

-- name: CreateEntry :one
INSERT INTO entries (slug, calendar_id, name, type, starts_at, ends_at, location, description, response_deadline, recurrence_rule_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateEntry :one
UPDATE entries SET
    name = $2,
    starts_at = $3,
    ends_at = $4,
    location = $5,
    description = $6,
    response_deadline = $7,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteEntry :exec
DELETE FROM entries WHERE id = $1;

-- name: GetEntryShiftDetails :one
SELECT * FROM entry_shift_details WHERE entry_id = $1;

-- name: UpsertEntryShiftDetails :one
INSERT INTO entry_shift_details (entry_id, required_participants, max_participants)
VALUES ($1, $2, $3)
ON CONFLICT (entry_id) DO UPDATE SET
    required_participants = EXCLUDED.required_participants,
    max_participants = EXCLUDED.max_participants
RETURNING *;

-- name: GetMeetingAudienceUnits :many
SELECT unit_id FROM meeting_audience_units WHERE entry_id = $1;

-- name: DeleteMeetingAudienceUnits :exec
DELETE FROM meeting_audience_units WHERE entry_id = $1;

-- name: InsertMeetingAudienceUnit :exec
INSERT INTO meeting_audience_units (entry_id, unit_id) VALUES ($1, $2);

-- name: ListEntryAnnotations :many
SELECT * FROM entry_annotations WHERE entry_id = $1;

-- name: CreateEntryAnnotation :one
INSERT INTO entry_annotations (entry_id, kind, message)
VALUES ($1, $2, $3)
RETURNING *;
