-- +goose Up

CREATE TABLE users (
    id             BIGSERIAL    PRIMARY KEY,
    idp_subject    TEXT         NOT NULL UNIQUE,
    username       TEXT         NOT NULL,
    display_name   TEXT         NOT NULL,
    email          TEXT         NOT NULL,
    timezone       TEXT,
    locale         TEXT,
    is_assoc_admin BOOLEAN      NOT NULL DEFAULT false,
    last_login_at  TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);

CREATE TABLE user_idp_groups (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_name TEXT   NOT NULL,
    PRIMARY KEY (user_id, group_name)
);

CREATE INDEX idx_user_idp_groups_group_name ON user_idp_groups (group_name);

-- +goose Down

DROP TABLE IF EXISTS user_idp_groups;
DROP TABLE IF EXISTS users;
