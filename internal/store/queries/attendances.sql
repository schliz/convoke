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
