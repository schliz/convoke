-- +goose Up

CREATE TABLE template_groups (
    id              BIGSERIAL    PRIMARY KEY,
    unit_id         BIGINT       NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    calendar_id     BIGINT       NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    name            TEXT         NOT NULL,
    base_start_time TIME         NOT NULL,
    location        TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_template_groups_unit_id ON template_groups (unit_id);
CREATE INDEX idx_template_groups_calendar_id ON template_groups (calendar_id);

CREATE TABLE templates (
    id                       BIGSERIAL    PRIMARY KEY,
    template_group_id        BIGINT       NOT NULL REFERENCES template_groups(id) ON DELETE CASCADE,
    name                     TEXT         NOT NULL,
    type                     TEXT         NOT NULL CHECK (type IN ('shift', 'meeting')),
    start_offset             INTERVAL     NOT NULL,
    duration                 INTERVAL     NOT NULL,
    required_participants    INT,
    max_participants         INT,
    description              TEXT,
    response_deadline_offset INTERVAL,
    sort_order               INT          NOT NULL DEFAULT 0,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_templates_template_group_id ON templates (template_group_id);

CREATE TABLE recurrence_rules (
    id                   BIGSERIAL    PRIMARY KEY,
    template_group_id    BIGINT       NOT NULL REFERENCES template_groups(id) ON DELETE CASCADE,
    pattern_type         TEXT         NOT NULL
        CHECK (pattern_type IN (
            'nth_weekday_of_month', 'nth_day_of_month',
            'every_nth_weekday', 'nth_workday_of_month',
            'nth_day_of_year', 'nth_workday_of_year'
        )),
    pattern_params       JSONB        NOT NULL,
    first_occurrence     DATE         NOT NULL,
    auto_create_horizon  INT          NOT NULL DEFAULT 14,
    enabled              BOOLEAN      NOT NULL DEFAULT true,
    weekend_action       TEXT         NOT NULL DEFAULT 'ignore'
        CHECK (weekend_action IN ('ignore', 'skip', 'warn')),
    weekend_warning_text TEXT,
    holiday_action       TEXT         NOT NULL DEFAULT 'ignore'
        CHECK (holiday_action IN ('ignore', 'skip', 'warn')),
    holiday_warning_text TEXT,
    last_evaluated_at    TIMESTAMPTZ,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_recurrence_rules_template_group_id ON recurrence_rules (template_group_id);
CREATE INDEX idx_recurrence_rules_enabled ON recurrence_rules (enabled);

-- +goose Down

DROP TABLE IF EXISTS recurrence_rules;
DROP TABLE IF EXISTS templates;
DROP TABLE IF EXISTS template_groups;
