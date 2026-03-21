-- +goose Up

CREATE TABLE feed_tokens (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope      TEXT         NOT NULL
        CHECK (scope IN ('calendar', 'unit', 'personal', 'all_visible')),
    scope_id   BIGINT,
    token      TEXT         NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_feed_tokens_user_id ON feed_tokens (user_id);
CREATE INDEX idx_feed_tokens_user_scope ON feed_tokens (user_id, scope, scope_id);

CREATE TABLE external_sources (
    id               BIGSERIAL    PRIMARY KEY,
    name             TEXT         NOT NULL,
    feed_url         TEXT         NOT NULL,
    calendar_id      BIGINT       NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    refresh_interval INTERVAL     NOT NULL DEFAULT '1 hour',
    enabled          BOOLEAN      NOT NULL DEFAULT true,
    last_fetched_at  TIMESTAMPTZ,
    last_error       TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_external_sources_calendar_id ON external_sources (calendar_id);

CREATE TABLE external_entries (
    id                 BIGSERIAL    PRIMARY KEY,
    external_source_id BIGINT       NOT NULL REFERENCES external_sources(id) ON DELETE CASCADE,
    uid                TEXT         NOT NULL,
    summary            TEXT,
    starts_at          TIMESTAMPTZ  NOT NULL,
    ends_at            TIMESTAMPTZ,
    location           TEXT,
    description        TEXT,
    raw_ical           TEXT,
    fetched_at         TIMESTAMPTZ  NOT NULL
);

CREATE UNIQUE INDEX idx_external_entries_source_uid ON external_entries (external_source_id, uid);
CREATE INDEX idx_external_entries_source_id ON external_entries (external_source_id);
CREATE INDEX idx_external_entries_starts_at ON external_entries (starts_at);

-- +goose Down

DROP TABLE IF EXISTS external_entries;
DROP TABLE IF EXISTS external_sources;
DROP TABLE IF EXISTS feed_tokens;
