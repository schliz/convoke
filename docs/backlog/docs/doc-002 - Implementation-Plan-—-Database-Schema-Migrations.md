---
id: doc-002
title: Implementation Plan — Database Schema Migrations
type: other
created_date: '2026-03-21 15:58'
---
# Database Schema Migrations — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the scaffolding migration with the complete 24-table schema from the approved spec, set up sqlc for code generation, and verify everything runs cleanly.

**Architecture:** 9 goose migration files grouped by domain area (matching the spec sections), ordered to respect FK dependencies. The existing `00001_create_users.sql` is replaced. sqlc is configured to read schema from migration files and generate Go code with pgx/v5.

**Tech Stack:** PostgreSQL 18, goose v3 (embedded SQL migrations), sqlc with pgx/v5, Go 1.25

**Spec:** `docs/superpowers/specs/2026-03-21-database-schema-design.md`
**ADR:** `docs/backlog/documents/doc-001`

---

### Task 1: Set up sqlc and remove old migration

**Files:**
- Delete: `migrations/00001_create_users.sql`
- Create: `sqlc.yaml`
- Modify: `internal/store/user.go` (will be rewritten in TASK-003, just remove for now)
- Modify: `internal/model/user.go` (will be rewritten in TASK-003, just remove for now)
- Modify: `go.mod` (add sqlc-compatible dep if needed)

**Context:** The existing migration uses BIGSERIAL + TEXT[] groups which conflicts with the new schema (ADR-9). sqlc needs a config file pointing at our migration files for schema input.

- [ ] **Step 1: Delete old migration**

```bash
rm migrations/00001_create_users.sql
```

- [ ] **Step 2: Create sqlc.yaml at project root**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/store/queries/"
    schema: "migrations/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
```

- [ ] **Step 3: Create the queries directory**

```bash
mkdir -p internal/store/queries
mkdir -p internal/db
```

- [ ] **Step 4: Remove old store/user.go and model/user.go**

These files reference the old schema (TEXT[] groups, no idp_subject). They will be rewritten in TASK-003 using sqlc-generated types. Delete them now to avoid build errors against the new migrations.

```bash
rm internal/store/user.go internal/model/user.go
```

- [ ] **Step 5: Update auth.go to remove dependency on deleted store function**

The auth middleware calls `store.GetOrCreateUser`. Replace with a temporary TODO stub so the app compiles. The real implementation will come in TASK-003/004.

Read `internal/auth/auth.go` and replace the `GetOrCreateUser` call with a placeholder that satisfies the compiler. The key is: the app must still compile and start (in dev mode) after this change.

- [ ] **Step 6: Verify the app compiles**

```bash
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: remove old users migration and add sqlc config

Prepares for new schema per ADR-9. Old user.go store/model
files removed (will be regenerated via sqlc in TASK-003)."
```

---

### Task 2: Migration 00001 — Identity & Group Sync

**Files:**
- Create: `migrations/00001_identity.sql`

**Spec reference:** Section 1 (users, user_idp_groups)

- [ ] **Step 1: Write migration file**

```sql
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
```

- [ ] **Step 2: Verify migration runs**

```bash
make dev  # or: docker compose up -d postgres && go run ./cmd/server
```

Check logs for successful migration. Then:

```bash
docker compose exec postgres psql -U convoke -c '\dt'
```

Expected: `users` and `user_idp_groups` tables exist.

- [ ] **Step 3: Commit**

```bash
git add migrations/00001_identity.sql
git commit -m "feat: add migration 00001 — users and IdP groups"
```

---

### Task 3: Migration 00002 — Units

**Files:**
- Create: `migrations/00002_units.sql`

**Spec reference:** Section 2 (units, unit_group_bindings)

- [ ] **Step 1: Write migration file**

```sql
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
```

- [ ] **Step 2: Verify migration runs**

```bash
docker compose exec postgres psql -U convoke -c '\dt'
```

Expected: 4 tables total (users, user_idp_groups, units, unit_group_bindings).

- [ ] **Step 3: Commit**

```bash
git add migrations/00002_units.sql
git commit -m "feat: add migration 00002 — units and group bindings"
```

---

### Task 4: Migration 00003 — Calendars

**Files:**
- Create: `migrations/00003_calendars.sql`

**Spec reference:** Section 3 (calendars, calendar_custom_viewers). Events are in a separate migration since they're a different domain concept.

- [ ] **Step 1: Write migration file**

```sql
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
```

- [ ] **Step 2: Verify and commit**

```bash
docker compose exec postgres psql -U convoke -c '\dt'
git add migrations/00003_calendars.sql
git commit -m "feat: add migration 00003 — calendars and custom viewers"
```

---

### Task 5: Migration 00004 — Templates & Recurrence

**Files:**
- Create: `migrations/00004_templates.sql`

**Spec reference:** Section 6 (template_groups, templates, recurrence_rules). Must come before entries because `entries.recurrence_rule_id` references `recurrence_rules(id)`.

- [ ] **Step 1: Write migration file**

```sql
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
```

- [ ] **Step 2: Verify and commit**

```bash
docker compose exec postgres psql -U convoke -c '\dt'
git add migrations/00004_templates.sql
git commit -m "feat: add migration 00004 — template groups, templates, recurrence rules"
```

---

### Task 6: Migration 00005 — Entries

**Files:**
- Create: `migrations/00005_entries.sql`

**Spec reference:** Section 4 (entries, entry_shift_details, meeting_audience_units, entry_annotations)

- [ ] **Step 1: Write migration file**

```sql
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
```

- [ ] **Step 2: Verify and commit**

```bash
docker compose exec postgres psql -U convoke -c '\dt'
git add migrations/00005_entries.sql
git commit -m "feat: add migration 00005 — entries, shift details, audience, annotations"
```

---

### Task 7: Migration 00006 — Events

**Files:**
- Create: `migrations/00006_events.sql`

**Spec reference:** Section 3 (events, event_calendars). Separated from calendars because events are a distinct concept (ADR-7: loose grouping).

- [ ] **Step 1: Write migration file**

```sql
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
```

- [ ] **Step 2: Verify and commit**

```bash
docker compose exec postgres psql -U convoke -c '\dt'
git add migrations/00006_events.sql
git commit -m "feat: add migration 00006 — events and event-calendar junction"
```

---

### Task 8: Migration 00007 — Attendance & Substitution

**Files:**
- Create: `migrations/00007_attendance.sql`

**Spec reference:** Section 5 (attendances, substitution_requests)

- [ ] **Step 1: Write migration file**

```sql
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
```

- [ ] **Step 2: Verify and commit**

```bash
docker compose exec postgres psql -U convoke -c '\dt'
git add migrations/00007_attendance.sql
git commit -m "feat: add migration 00007 — attendances and substitution requests"
```

---

### Task 9: Migration 00008 — Calendar Integration

**Files:**
- Create: `migrations/00008_calendar_integration.sql`

**Spec reference:** Section 7 (feed_tokens, external_sources, external_entries)

- [ ] **Step 1: Write migration file**

```sql
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
```

- [ ] **Step 2: Verify and commit**

```bash
docker compose exec postgres psql -U convoke -c '\dt'
git add migrations/00008_calendar_integration.sql
git commit -m "feat: add migration 00008 — feed tokens, external sources and entries"
```

---

### Task 10: Migration 00009 — Notifications & Webhooks

**Files:**
- Create: `migrations/00009_notifications.sql`

**Spec reference:** Section 8 (notification_configs, user_notification_preferences, notifications, webhooks)

- [ ] **Step 1: Write migration file**

```sql
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
```

- [ ] **Step 2: Verify and commit**

```bash
docker compose exec postgres psql -U convoke -c '\dt'
git add migrations/00009_notifications.sql
git commit -m "feat: add migration 00009 — notifications, preferences, webhooks"
```

---

### Task 11: Full verification

**Files:** None (verification only)

- [ ] **Step 1: Reset database and run all migrations from scratch**

```bash
docker compose down -v && docker compose up -d postgres
# Wait for postgres to be ready
sleep 2
go run ./cmd/server
```

Check logs for all 9 migrations running successfully.

- [ ] **Step 2: Verify all 24 tables exist**

```bash
docker compose exec postgres psql -U convoke -c '\dt' | wc -l
```

Expected: 24 tables (plus goose_db_version).

- [ ] **Step 3: Verify goose down works**

```bash
docker compose exec postgres psql -U convoke -c "SELECT version FROM goose_db_version ORDER BY id DESC LIMIT 1;"
```

Expected: version 9.

- [ ] **Step 4: Verify sqlc can parse the schema**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc compile
```

Expected: no errors. This validates sqlc can read all migration files as schema input. (No queries to generate yet — that's TASK-003.)

- [ ] **Step 5: Add sqlc generate to Makefile**

Add a `sqlc` target to the Makefile:

```makefile
.PHONY: sqlc
sqlc:
	sqlc generate
```

- [ ] **Step 6: Final commit**

```bash
git add Makefile
git commit -m "chore: add sqlc generate target to Makefile"
```

---

## Migration Dependency Order

```
00001_identity        (no deps)
00002_units           (no deps)
00003_calendars       → units
00004_templates       → units, calendars
00005_entries         → calendars, recurrence_rules
00006_events          → units, calendars
00007_attendance      → entries, users
00008_calendar_integration → users, calendars
00009_notifications   → calendars, users, entries, units
```

## What This Does NOT Cover

- Go model types and sqlc query files → TASK-003
- Store methods and business logic → TASK-004
- Updating auth middleware for new user schema → TASK-003 (as part of model rewrite)
- Seed data for development → separate concern
