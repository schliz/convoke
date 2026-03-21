-- name: GetAttendanceByID :one
SELECT * FROM attendances WHERE id = $1;

-- name: GetAttendanceByEntryAndUser :one
SELECT * FROM attendances WHERE entry_id = $1 AND user_id = $2;

-- name: ListAttendancesByEntry :many
SELECT * FROM attendances WHERE entry_id = $1 ORDER BY created_at;

-- name: ListAttendancesByUser :many
SELECT * FROM attendances WHERE user_id = $1 ORDER BY created_at DESC;

-- name: ListPendingAttendancesByUser :many
SELECT a.* FROM attendances a
JOIN entries e ON a.entry_id = e.id
WHERE a.user_id = $1 AND a.status = 'pending'
ORDER BY e.starts_at;

-- name: CreateAttendance :one
INSERT INTO attendances (entry_id, user_id, status, note)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateAttendanceStatus :one
UPDATE attendances SET
    status = $2,
    responded_at = CASE WHEN $2 != 'pending' THEN now() ELSE responded_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpsertAttendance :one
INSERT INTO attendances (entry_id, user_id, status, note)
VALUES ($1, $2, $3, $4)
ON CONFLICT (entry_id, user_id) DO UPDATE SET
    status = EXCLUDED.status,
    note = COALESCE(EXCLUDED.note, attendances.note),
    responded_at = CASE WHEN EXCLUDED.status != 'pending' THEN now() ELSE attendances.responded_at END,
    updated_at = now()
RETURNING *;

-- name: UpdateAttendanceStatusByEntryAndUser :one
UPDATE attendances SET
    status = $3,
    responded_at = CASE WHEN $3 != 'pending' THEN now() ELSE responded_at END,
    updated_at = now()
WHERE entry_id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteAttendance :exec
DELETE FROM attendances WHERE entry_id = $1 AND user_id = $2;

-- name: ListAttendeesByEntry :many
SELECT
    a.id, a.entry_id, a.user_id, a.status, a.confirmed, a.note,
    a.responded_at, a.created_at AS attendance_created_at,
    u.display_name, u.email
FROM attendances a
JOIN users u ON a.user_id = u.id
WHERE a.entry_id = $1
ORDER BY
    CASE a.status
        WHEN 'accepted' THEN 1
        WHEN 'needs_substitute' THEN 2
        WHEN 'pending' THEN 3
        WHEN 'declined' THEN 4
        WHEN 'replaced' THEN 5
    END,
    a.created_at;

-- name: CountAcceptedAttendees :one
SELECT COUNT(*) FROM attendances
WHERE entry_id = $1 AND status = 'accepted';

-- name: IsAttendee :one
SELECT EXISTS(
    SELECT 1 FROM attendances WHERE entry_id = $1 AND user_id = $2
) AS is_attendee;

-- name: ListEntriesWithUserAttendance :many
SELECT
    e.id, e.slug, e.calendar_id, e.name, e.type, e.starts_at, e.ends_at,
    e.location, e.description, e.response_deadline,
    a.status AS attendance_status, a.responded_at
FROM entries e
JOIN attendances a ON a.entry_id = e.id
WHERE a.user_id = $1
  AND e.starts_at >= $2
  AND e.starts_at < $3
ORDER BY e.starts_at;

-- name: CountAttendancesByEntryAndStatus :one
SELECT
    COUNT(*) FILTER (WHERE status = 'accepted') AS accepted,
    COUNT(*) FILTER (WHERE status = 'declined') AS declined,
    COUNT(*) FILTER (WHERE status = 'pending') AS pending,
    COUNT(*) FILTER (WHERE status = 'needs_substitute') AS needs_substitute
FROM attendances WHERE entry_id = $1;

-- name: GetSubstitutionRequestByAttendance :one
SELECT * FROM substitution_requests WHERE attendance_id = $1;

-- name: CreateSubstitutionRequest :one
INSERT INTO substitution_requests (attendance_id)
VALUES ($1)
RETURNING *;

-- name: ClaimSubstitutionRequest :one
UPDATE substitution_requests SET
    claimed_by_user_id = $2,
    claimed_at = now()
WHERE id = $1 AND claimed_by_user_id IS NULL
RETURNING *;

-- name: ListOpenSubstitutionRequests :many
SELECT sr.*, a.entry_id, a.user_id FROM substitution_requests sr
JOIN attendances a ON sr.attendance_id = a.id
WHERE sr.claimed_by_user_id IS NULL
ORDER BY sr.created_at;
