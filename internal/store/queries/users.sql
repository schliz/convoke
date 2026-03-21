-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByIDPSubject :one
SELECT * FROM users WHERE idp_subject = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpsertUser :one
INSERT INTO users (idp_subject, username, display_name, email, is_assoc_admin, last_login_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (idp_subject) DO UPDATE SET
    username = EXCLUDED.username,
    display_name = EXCLUDED.display_name,
    email = EXCLUDED.email,
    is_assoc_admin = EXCLUDED.is_assoc_admin,
    last_login_at = now(),
    updated_at = now()
RETURNING *;

-- name: UpdateUserPreferences :one
UPDATE users SET
    timezone = $2,
    locale = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;
