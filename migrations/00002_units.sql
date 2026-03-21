-- +goose Up

CREATE TABLE units (
    id            BIGSERIAL    PRIMARY KEY,
    name          TEXT         NOT NULL,
    slug          TEXT         NOT NULL UNIQUE,
    description   TEXT,
    logo_path     TEXT,
    contact_email TEXT,
    admin_group   TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE unit_group_bindings (
    unit_id    BIGINT NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    group_name TEXT   NOT NULL,
    PRIMARY KEY (unit_id, group_name)
);

CREATE INDEX idx_unit_group_bindings_group_name ON unit_group_bindings (group_name);

-- +goose Down

DROP TABLE IF EXISTS unit_group_bindings;
DROP TABLE IF EXISTS units;
