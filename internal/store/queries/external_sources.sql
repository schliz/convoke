-- name: GetExternalSourceByID :one
SELECT * FROM external_sources WHERE id = $1;

-- name: ListExternalSourcesByCalendar :many
SELECT * FROM external_sources WHERE calendar_id = $1;

-- name: ListEnabledExternalSources :many
SELECT * FROM external_sources WHERE enabled = true;

-- name: CreateExternalSource :one
INSERT INTO external_sources (name, feed_url, calendar_id, refresh_interval, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateExternalSource :one
UPDATE external_sources SET
    name = $2,
    feed_url = $3,
    refresh_interval = $4,
    enabled = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateExternalSourceFetchStatus :exec
UPDATE external_sources SET
    last_fetched_at = now(),
    last_error = $2,
    updated_at = now()
WHERE id = $1;

-- name: UpsertExternalEntry :one
INSERT INTO external_entries (external_source_id, uid, summary, starts_at, ends_at, location, description, raw_ical, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (external_source_id, uid) DO UPDATE SET
    summary = EXCLUDED.summary,
    starts_at = EXCLUDED.starts_at,
    ends_at = EXCLUDED.ends_at,
    location = EXCLUDED.location,
    description = EXCLUDED.description,
    raw_ical = EXCLUDED.raw_ical,
    fetched_at = now()
RETURNING *;

-- name: ListExternalEntriesBySource :many
SELECT * FROM external_entries WHERE external_source_id = $1 ORDER BY starts_at;
