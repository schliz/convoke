-- +goose Up

CREATE TABLE attendances (
    id           BIGSERIAL    PRIMARY KEY,
    entry_id     BIGINT       NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    user_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       TEXT         NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'declined', 'needs_substitute', 'replaced')),
    confirmed    BOOLEAN      NOT NULL DEFAULT false,
    note         TEXT,
    responded_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_attendances_entry_user ON attendances (entry_id, user_id);
CREATE INDEX idx_attendances_entry_id ON attendances (entry_id);
CREATE INDEX idx_attendances_user_id ON attendances (user_id);
CREATE INDEX idx_attendances_user_status ON attendances (user_id, status);

CREATE TABLE substitution_requests (
    id                 BIGSERIAL    PRIMARY KEY,
    attendance_id      BIGINT       NOT NULL UNIQUE REFERENCES attendances(id) ON DELETE CASCADE,
    claimed_by_user_id BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    claimed_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE IF EXISTS substitution_requests;
DROP TABLE IF EXISTS attendances;
