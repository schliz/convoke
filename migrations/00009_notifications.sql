-- +goose Up

CREATE TABLE notification_configs (
    id          BIGSERIAL    PRIMARY KEY,
    calendar_id BIGINT       NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    event_type  TEXT         NOT NULL
        CHECK (event_type IN (
            'new_entry', 'entry_changed', 'entry_canceled',
            'reminder_before_entry', 'response_deadline_approaching',
            'non_response_escalation', 'staffing_warning',
            'substitute_requested', 'substitute_found'
        )),
    enabled     BOOLEAN      NOT NULL DEFAULT true,
    lead_time   INTERVAL
);

CREATE UNIQUE INDEX idx_notification_configs_cal_type ON notification_configs (calendar_id, event_type);

CREATE TABLE user_notification_preferences (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT         NOT NULL
        CHECK (event_type IN (
            'new_entry', 'entry_changed', 'entry_canceled',
            'reminder_before_entry', 'response_deadline_approaching',
            'non_response_escalation', 'staffing_warning',
            'substitute_requested', 'substitute_found'
        )),
    channel    TEXT         NOT NULL CHECK (channel IN ('email', 'webhook')),
    enabled    BOOLEAN      NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX idx_user_notif_prefs_unique ON user_notification_preferences (user_id, event_type, channel);

CREATE TABLE notifications (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_id    BIGINT       REFERENCES entries(id) ON DELETE SET NULL,
    event_type  TEXT         NOT NULL
        CHECK (event_type IN (
            'new_entry', 'entry_changed', 'entry_canceled',
            'reminder_before_entry', 'response_deadline_approaching',
            'non_response_escalation', 'staffing_warning',
            'substitute_requested', 'substitute_found'
        )),
    channel     TEXT         NOT NULL CHECK (channel IN ('email', 'webhook')),
    status      TEXT         NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed', 'retrying')),
    payload     JSONB,
    error       TEXT,
    retry_count INT          NOT NULL DEFAULT 0,
    sent_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_id ON notifications (user_id);
CREATE INDEX idx_notifications_entry_id ON notifications (entry_id);
CREATE INDEX idx_notifications_status_created ON notifications (status, created_at);
CREATE INDEX idx_notifications_user_type_entry ON notifications (user_id, event_type, entry_id);

CREATE TABLE webhooks (
    id         BIGSERIAL    PRIMARY KEY,
    unit_id    BIGINT       REFERENCES units(id) ON DELETE CASCADE,
    name       TEXT         NOT NULL,
    url        TEXT         NOT NULL,
    secret     TEXT,
    enabled    BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_webhooks_unit_id ON webhooks (unit_id);

-- +goose Down

DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS user_notification_preferences;
DROP TABLE IF EXISTS notification_configs;
