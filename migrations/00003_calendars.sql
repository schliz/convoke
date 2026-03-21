-- +goose Up

CREATE TABLE calendars (
    id                     BIGSERIAL    PRIMARY KEY,
    slug                   TEXT         NOT NULL UNIQUE,
    unit_id                BIGINT       NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    name                   TEXT         NOT NULL,
    creation_policy        TEXT         NOT NULL DEFAULT 'admins_only'
        CHECK (creation_policy IN ('admins_only', 'unit_members')),
    visibility             TEXT         NOT NULL DEFAULT 'association'
        CHECK (visibility IN ('association', 'unit', 'custom')),
    participation          TEXT         NOT NULL DEFAULT 'viewers'
        CHECK (participation IN ('viewers', 'unit', 'nobody')),
    participant_visibility TEXT         NOT NULL DEFAULT 'everyone'
        CHECK (participant_visibility IN ('everyone', 'unit', 'participants_only')),
    color                  TEXT,
    sort_order             INT          NOT NULL DEFAULT 0,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_calendars_unit_id ON calendars (unit_id);

CREATE TABLE calendar_custom_viewers (
    calendar_id BIGINT NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    unit_id     BIGINT NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    PRIMARY KEY (calendar_id, unit_id)
);

-- +goose Down

DROP TABLE IF EXISTS calendar_custom_viewers;
DROP TABLE IF EXISTS calendars;
