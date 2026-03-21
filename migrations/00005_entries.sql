-- +goose Up

CREATE TABLE entries (
    id                 BIGSERIAL    PRIMARY KEY,
    slug               TEXT         NOT NULL UNIQUE,
    calendar_id        BIGINT       NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    name               TEXT         NOT NULL,
    type               TEXT         NOT NULL CHECK (type IN ('shift', 'meeting')),
    starts_at          TIMESTAMPTZ  NOT NULL,
    ends_at            TIMESTAMPTZ  NOT NULL,
    location           TEXT,
    description        TEXT,
    response_deadline  TIMESTAMPTZ,
    recurrence_rule_id BIGINT       REFERENCES recurrence_rules(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CHECK (ends_at > starts_at)
);

CREATE INDEX idx_entries_calendar_starts ON entries (calendar_id, starts_at);
CREATE INDEX idx_entries_starts_at ON entries (starts_at);
CREATE INDEX idx_entries_recurrence_rule_id ON entries (recurrence_rule_id);
CREATE UNIQUE INDEX idx_entries_idempotency ON entries (calendar_id, name, starts_at);

CREATE TABLE entry_shift_details (
    entry_id             BIGINT PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
    required_participants INT   NOT NULL CHECK (required_participants >= 1),
    max_participants     INT    NOT NULL DEFAULT 0
);

CREATE TABLE meeting_audience_units (
    entry_id BIGINT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    unit_id  BIGINT NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, unit_id)
);

CREATE TABLE entry_annotations (
    id       BIGSERIAL PRIMARY KEY,
    entry_id BIGINT    NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    kind     TEXT      NOT NULL,
    message  TEXT      NOT NULL
);

CREATE INDEX idx_entry_annotations_entry_id ON entry_annotations (entry_id);

-- +goose Down

DROP TABLE IF EXISTS entry_annotations;
DROP TABLE IF EXISTS meeting_audience_units;
DROP TABLE IF EXISTS entry_shift_details;
DROP TABLE IF EXISTS entries;
