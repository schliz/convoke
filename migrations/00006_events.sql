-- +goose Up

CREATE TABLE events (
    id          BIGSERIAL    PRIMARY KEY,
    slug        TEXT         NOT NULL UNIQUE,
    unit_id     BIGINT       NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    name        TEXT         NOT NULL,
    start_date  DATE         NOT NULL,
    end_date    DATE         NOT NULL,
    website     TEXT,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CHECK (end_date >= start_date)
);

CREATE INDEX idx_events_unit_id ON events (unit_id);

CREATE TABLE event_calendars (
    event_id    BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    calendar_id BIGINT NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    sort_order  INT    NOT NULL DEFAULT 0,
    PRIMARY KEY (event_id, calendar_id)
);

-- +goose Down

DROP TABLE IF EXISTS event_calendars;
DROP TABLE IF EXISTS events;
