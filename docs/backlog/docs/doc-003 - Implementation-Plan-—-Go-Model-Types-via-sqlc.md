---
id: doc-003
title: Implementation Plan — Go Model Types via sqlc
type: other
created_date: '2026-03-21 16:02'
---
# Go Model Types via sqlc — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate type-safe Go model types for all 24 database tables using sqlc, and write the foundational SQL queries that the store layer will need.

**Architecture:** sqlc reads the goose migration files as schema, plus hand-written `.sql` query files, and generates Go code in `internal/db/`. The generated `models.go` contains structs for every table. Query files produce typed Go functions. The old `internal/model/` package is replaced by sqlc output.

**Tech Stack:** sqlc with pgx/v5, Go 1.25, PostgreSQL 18

**Spec:** `docs/superpowers/specs/2026-03-21-database-schema-design.md`
**ADR:** `docs/backlog/documents/doc-001` (ADR-2: sqlc for data access)
**Depends on:** TASK-002 (migrations must exist for sqlc to parse schema)

---

### Note on TASK-003 scope change

The original task says "Create Go structs in internal/model/". With ADR-2 (sqlc), types are auto-generated into `internal/db/` from the migration SQL. This is better than manual structs because:
- Types stay in sync with schema automatically
- Row scanning is generated (no manual Scan calls)
- Query functions are type-safe at compile time

The `internal/model/` directory will be removed. All entity types come from `internal/db/`.

---

### Task 1: Verify sqlc setup and generate models

**Files:**
- Verify: `sqlc.yaml` (created in TASK-002)
- Create: `internal/store/queries/placeholder.sql` (minimal query so sqlc generates)
- Generated: `internal/db/models.go`, `internal/db/db.go`, `internal/db/querier.go`

**Context:** sqlc generates `models.go` with structs for all tables found in the schema (migrations/). It also needs at least one query file. We create a minimal placeholder.

- [ ] **Step 1: Verify sqlc.yaml exists and is correct**

Read `sqlc.yaml` at project root. It should contain:

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

- [ ] **Step 2: Create placeholder query file**

Create `internal/store/queries/users.sql`:

```sql
-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;
```

- [ ] **Step 3: Run sqlc generate**

```bash
sqlc generate
```

Expected: files created in `internal/db/`: `models.go`, `db.go`, `users.sql.go`

- [ ] **Step 4: Verify models.go has all 24 table structs**

Read `internal/db/models.go` and verify it contains structs for:
users, UserIdpGroup, Unit, UnitGroupBinding, Calendar, CalendarCustomViewer, Event, EventCalendar, Entry, EntryShiftDetail, MeetingAudienceUnit, EntryAnnotation, Attendance, SubstitutionRequest, TemplateGroup, Template, RecurrenceRule, FeedToken, ExternalSource, ExternalEntry, NotificationConfig, UserNotificationPreference, Notification, Webhook

- [ ] **Step 5: Verify type mappings are correct**

Check that sqlc mapped PostgreSQL types to Go types correctly:
- `BIGSERIAL / BIGINT` → `int64`
- `TEXT NOT NULL` → `string`
- `TEXT` (nullable) → `pgtype.Text`
- `BOOLEAN NOT NULL` → `bool`
- `TIMESTAMPTZ NOT NULL` → `pgtype.Timestamptz`
- `TIMESTAMPTZ` (nullable) → `pgtype.Timestamptz`
- `INT NOT NULL` → `int32`
- `INT` (nullable) → `pgtype.Int4`
- `JSONB` → `[]byte` or `pgtype.JSON`
- `DATE` → `pgtype.Date`
- `TIME` → `pgtype.Time`
- `INTERVAL` → `pgtype.Interval`

If any types are wrong, add sqlc overrides to `sqlc.yaml`:

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
        overrides:
          - db_type: "timestamptz"
            go_type: "time.Time"
            nullable: false
          - db_type: "timestamptz"
            go_type:
              import: "database/sql"
              type: "NullTime"
            nullable: true
```

Adjust overrides based on what sqlc actually generates. The key requirement is that non-nullable timestamps are `time.Time`, not pgtype wrappers.

- [ ] **Step 6: Commit**

```bash
git add internal/store/queries/users.sql internal/db/ sqlc.yaml
git commit -m "feat: generate initial sqlc models from schema

sqlc reads migration files as schema and generates Go types
for all 24 database tables in internal/db/models.go."
```

---

### Task 2: Write user and identity queries

**Files:**
- Modify: `internal/store/queries/users.sql`

**Spec reference:** Section 1 (users, user_idp_groups)

These are the queries the auth middleware and user store will need.

- [ ] **Step 1: Write user queries**

```sql
-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByIDPSubject :one
SELECT * FROM users WHERE idp_subject = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpsertUser :one
INSERT INTO users (idp_subject, username, display_name, email, is_assoc_admin, last_login_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (idp_subject) DO UPDATE SET
    username = EXCLUDED.username,
    display_name = EXCLUDED.display_name,
    email = EXCLUDED.email,
    is_assoc_admin = EXCLUDED.is_assoc_admin,
    last_login_at = now(),
    updated_at = now()
RETURNING *;

-- name: UpdateUserPreferences :one
UPDATE users SET
    timezone = $2,
    locale = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;
```

- [ ] **Step 2: Write IdP group sync queries**

Create `internal/store/queries/user_idp_groups.sql`:

```sql
-- name: DeleteUserIDPGroups :exec
DELETE FROM user_idp_groups WHERE user_id = $1;

-- name: InsertUserIDPGroup :exec
INSERT INTO user_idp_groups (user_id, group_name) VALUES ($1, $2);

-- name: GetUserIDPGroups :many
SELECT group_name FROM user_idp_groups WHERE user_id = $1;

-- name: IsUserInGroup :one
SELECT EXISTS(
    SELECT 1 FROM user_idp_groups
    WHERE user_id = $1 AND group_name = $2
) AS is_member;
```

- [ ] **Step 3: Run sqlc generate and verify**

```bash
sqlc generate
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/store/queries/ internal/db/
git commit -m "feat: add user and IdP group queries"
```

---

### Task 3: Write unit and membership queries

**Files:**
- Create: `internal/store/queries/units.sql`
- Create: `internal/store/queries/unit_group_bindings.sql`

**Spec reference:** Section 2 (units, unit_group_bindings)

- [ ] **Step 1: Write unit queries**

```sql
-- name: GetUnitByID :one
SELECT * FROM units WHERE id = $1;

-- name: GetUnitBySlug :one
SELECT * FROM units WHERE slug = $1;

-- name: ListUnits :many
SELECT * FROM units ORDER BY name;

-- name: CreateUnit :one
INSERT INTO units (name, slug, description, logo_path, contact_email, admin_group)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateUnit :one
UPDATE units SET
    name = $2,
    description = $3,
    logo_path = $4,
    contact_email = $5,
    admin_group = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteUnit :exec
DELETE FROM units WHERE id = $1;
```

- [ ] **Step 2: Write membership resolution queries**

```sql
-- name: IsUserMemberOfUnit :one
SELECT EXISTS(
    SELECT 1 FROM user_idp_groups uig
    JOIN unit_group_bindings ugb ON uig.group_name = ugb.group_name
    WHERE uig.user_id = $1 AND ugb.unit_id = $2
) AS is_member;

-- name: IsUserAdminOfUnit :one
SELECT EXISTS(
    SELECT 1 FROM user_idp_groups uig
    JOIN units u ON uig.group_name = u.admin_group
    WHERE uig.user_id = $1 AND u.id = $2
) AS is_admin;

-- name: ListUnitsForUser :many
SELECT DISTINCT u.* FROM units u
JOIN unit_group_bindings ugb ON u.id = ugb.unit_id
JOIN user_idp_groups uig ON ugb.group_name = uig.group_name
WHERE uig.user_id = $1
ORDER BY u.name;

-- name: SetUnitGroupBindings :exec
DELETE FROM unit_group_bindings WHERE unit_id = $1;

-- name: InsertUnitGroupBinding :exec
INSERT INTO unit_group_bindings (unit_id, group_name) VALUES ($1, $2);

-- name: GetUnitGroupBindings :many
SELECT group_name FROM unit_group_bindings WHERE unit_id = $1;
```

- [ ] **Step 3: Run sqlc generate and verify**

```bash
sqlc generate
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/store/queries/ internal/db/
git commit -m "feat: add unit and membership resolution queries"
```

---

### Task 4: Write calendar queries

**Files:**
- Create: `internal/store/queries/calendars.sql`

**Spec reference:** Section 3 (calendars, calendar_custom_viewers)

- [ ] **Step 1: Write calendar queries**

```sql
-- name: GetCalendarByID :one
SELECT * FROM calendars WHERE id = $1;

-- name: GetCalendarBySlug :one
SELECT * FROM calendars WHERE slug = $1;

-- name: ListCalendarsByUnit :many
SELECT * FROM calendars WHERE unit_id = $1 ORDER BY sort_order, name;

-- name: CreateCalendar :one
INSERT INTO calendars (slug, unit_id, name, creation_policy, visibility, participation, participant_visibility, color, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateCalendar :one
UPDATE calendars SET
    name = $2,
    creation_policy = $3,
    visibility = $4,
    participation = $5,
    participant_visibility = $6,
    color = $7,
    sort_order = $8,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteCalendar :exec
DELETE FROM calendars WHERE id = $1;

-- name: ListVisibleCalendarsForUser :many
SELECT DISTINCT c.* FROM calendars c
WHERE c.visibility = 'association'
   OR (c.visibility = 'unit' AND EXISTS(
       SELECT 1 FROM unit_group_bindings ugb
       JOIN user_idp_groups uig ON ugb.group_name = uig.group_name
       WHERE ugb.unit_id = c.unit_id AND uig.user_id = $1
   ))
   OR (c.visibility = 'custom' AND EXISTS(
       SELECT 1 FROM calendar_custom_viewers ccv
       JOIN unit_group_bindings ugb ON ccv.unit_id = ugb.unit_id
       JOIN user_idp_groups uig ON ugb.group_name = uig.group_name
       WHERE ccv.calendar_id = c.id AND uig.user_id = $1
   ))
ORDER BY c.sort_order, c.name;

-- name: SetCalendarCustomViewers :exec
DELETE FROM calendar_custom_viewers WHERE calendar_id = $1;

-- name: InsertCalendarCustomViewer :exec
INSERT INTO calendar_custom_viewers (calendar_id, unit_id) VALUES ($1, $2);
```

- [ ] **Step 2: Run sqlc generate and verify**

```bash
sqlc generate
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/store/queries/ internal/db/
git commit -m "feat: add calendar and visibility queries"
```

---

### Task 5: Write entry queries

**Files:**
- Create: `internal/store/queries/entries.sql`

**Spec reference:** Section 4 (entries, entry_shift_details, meeting_audience_units, entry_annotations)

- [ ] **Step 1: Write entry queries**

```sql
-- name: GetEntryByID :one
SELECT * FROM entries WHERE id = $1;

-- name: GetEntryBySlug :one
SELECT * FROM entries WHERE slug = $1;

-- name: ListEntriesByCalendarAndDateRange :many
SELECT * FROM entries
WHERE calendar_id = $1
  AND starts_at >= $2
  AND starts_at < $3
ORDER BY starts_at;

-- name: CreateEntry :one
INSERT INTO entries (slug, calendar_id, name, type, starts_at, ends_at, location, description, response_deadline, recurrence_rule_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateEntry :one
UPDATE entries SET
    name = $2,
    starts_at = $3,
    ends_at = $4,
    location = $5,
    description = $6,
    response_deadline = $7,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteEntry :exec
DELETE FROM entries WHERE id = $1;

-- name: GetEntryShiftDetails :one
SELECT * FROM entry_shift_details WHERE entry_id = $1;

-- name: UpsertEntryShiftDetails :one
INSERT INTO entry_shift_details (entry_id, required_participants, max_participants)
VALUES ($1, $2, $3)
ON CONFLICT (entry_id) DO UPDATE SET
    required_participants = EXCLUDED.required_participants,
    max_participants = EXCLUDED.max_participants
RETURNING *;

-- name: GetMeetingAudienceUnits :many
SELECT unit_id FROM meeting_audience_units WHERE entry_id = $1;

-- name: SetMeetingAudienceUnits :exec
DELETE FROM meeting_audience_units WHERE entry_id = $1;

-- name: InsertMeetingAudienceUnit :exec
INSERT INTO meeting_audience_units (entry_id, unit_id) VALUES ($1, $2);

-- name: ListEntryAnnotations :many
SELECT * FROM entry_annotations WHERE entry_id = $1;

-- name: CreateEntryAnnotation :one
INSERT INTO entry_annotations (entry_id, kind, message)
VALUES ($1, $2, $3)
RETURNING *;
```

- [ ] **Step 2: Run sqlc generate and verify**

```bash
sqlc generate
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/store/queries/ internal/db/
git commit -m "feat: add entry, shift detail, audience, and annotation queries"
```

---

### Task 6: Write attendance and substitution queries

**Files:**
- Create: `internal/store/queries/attendances.sql`

**Spec reference:** Section 5 (attendances, substitution_requests)

- [ ] **Step 1: Write attendance queries**

```sql
-- name: GetAttendanceByID :one
SELECT * FROM attendances WHERE id = $1;

-- name: GetAttendanceByEntryAndUser :one
SELECT * FROM attendances WHERE entry_id = $1 AND user_id = $2;

-- name: ListAttendancesByEntry :many
SELECT * FROM attendances WHERE entry_id = $1 ORDER BY created_at;

-- name: ListAttendancesByUser :many
SELECT * FROM attendances WHERE user_id = $1 ORDER BY created_at DESC;

-- name: ListPendingAttendancesByUser :many
SELECT a.* FROM attendances a
JOIN entries e ON a.entry_id = e.id
WHERE a.user_id = $1 AND a.status = 'pending'
ORDER BY e.starts_at;

-- name: CreateAttendance :one
INSERT INTO attendances (entry_id, user_id, status, note)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateAttendanceStatus :one
UPDATE attendances SET
    status = $2,
    responded_at = CASE WHEN $2 != 'pending' THEN now() ELSE responded_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CountAttendancesByEntryAndStatus :one
SELECT
    COUNT(*) FILTER (WHERE status = 'accepted') AS accepted,
    COUNT(*) FILTER (WHERE status = 'declined') AS declined,
    COUNT(*) FILTER (WHERE status = 'pending') AS pending,
    COUNT(*) FILTER (WHERE status = 'needs_substitute') AS needs_substitute
FROM attendances WHERE entry_id = $1;

-- name: GetSubstitutionRequestByAttendance :one
SELECT * FROM substitution_requests WHERE attendance_id = $1;

-- name: CreateSubstitutionRequest :one
INSERT INTO substitution_requests (attendance_id)
VALUES ($1)
RETURNING *;

-- name: ClaimSubstitutionRequest :one
UPDATE substitution_requests SET
    claimed_by_user_id = $2,
    claimed_at = now()
WHERE id = $1 AND claimed_by_user_id IS NULL
RETURNING *;
```

- [ ] **Step 2: Run sqlc generate and verify**

```bash
sqlc generate
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/store/queries/ internal/db/
git commit -m "feat: add attendance and substitution queries"
```

---

### Task 7: Write event queries

**Files:**
- Create: `internal/store/queries/events.sql`

**Spec reference:** Section 3 (events, event_calendars)

- [ ] **Step 1: Write event queries**

```sql
-- name: GetEventByID :one
SELECT * FROM events WHERE id = $1;

-- name: GetEventBySlug :one
SELECT * FROM events WHERE slug = $1;

-- name: ListEventsByUnit :many
SELECT * FROM events WHERE unit_id = $1 ORDER BY start_date DESC;

-- name: CreateEvent :one
INSERT INTO events (slug, unit_id, name, start_date, end_date, website, description)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateEvent :one
UPDATE events SET
    name = $2,
    start_date = $3,
    end_date = $4,
    website = $5,
    description = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events WHERE id = $1;

-- name: ListEventCalendars :many
SELECT c.* FROM calendars c
JOIN event_calendars ec ON c.id = ec.calendar_id
WHERE ec.event_id = $1
ORDER BY ec.sort_order, c.name;

-- name: AddCalendarToEvent :exec
INSERT INTO event_calendars (event_id, calendar_id, sort_order)
VALUES ($1, $2, $3)
ON CONFLICT (event_id, calendar_id) DO UPDATE SET sort_order = EXCLUDED.sort_order;

-- name: RemoveCalendarFromEvent :exec
DELETE FROM event_calendars WHERE event_id = $1 AND calendar_id = $2;
```

- [ ] **Step 2: Run sqlc generate and verify**

```bash
sqlc generate
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/store/queries/ internal/db/
git commit -m "feat: add event and event-calendar queries"
```

---

### Task 8: Write remaining queries (templates, integration, notifications)

**Files:**
- Create: `internal/store/queries/templates.sql`
- Create: `internal/store/queries/feed_tokens.sql`
- Create: `internal/store/queries/external_sources.sql`
- Create: `internal/store/queries/notifications.sql`
- Create: `internal/store/queries/webhooks.sql`

**Spec reference:** Sections 6, 7, 8

- [ ] **Step 1: Write template queries**

```sql
-- name: GetTemplateGroupByID :one
SELECT * FROM template_groups WHERE id = $1;

-- name: ListTemplateGroupsByCalendar :many
SELECT * FROM template_groups WHERE calendar_id = $1 ORDER BY name;

-- name: CreateTemplateGroup :one
INSERT INTO template_groups (unit_id, calendar_id, name, base_start_time, location)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteTemplateGroup :exec
DELETE FROM template_groups WHERE id = $1;

-- name: ListTemplatesByGroup :many
SELECT * FROM templates WHERE template_group_id = $1 ORDER BY sort_order, name;

-- name: CreateTemplate :one
INSERT INTO templates (template_group_id, name, type, start_offset, duration, required_participants, max_participants, description, response_deadline_offset, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: DeleteTemplate :exec
DELETE FROM templates WHERE id = $1;

-- name: GetRecurrenceRuleByID :one
SELECT * FROM recurrence_rules WHERE id = $1;

-- name: ListRecurrenceRulesByTemplateGroup :many
SELECT * FROM recurrence_rules WHERE template_group_id = $1;

-- name: ListEnabledRecurrenceRules :many
SELECT * FROM recurrence_rules WHERE enabled = true;

-- name: CreateRecurrenceRule :one
INSERT INTO recurrence_rules (template_group_id, pattern_type, pattern_params, first_occurrence, auto_create_horizon, enabled, weekend_action, weekend_warning_text, holiday_action, holiday_warning_text)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateRecurrenceRuleLastEvaluated :exec
UPDATE recurrence_rules SET last_evaluated_at = now() WHERE id = $1;
```

- [ ] **Step 2: Write feed token queries**

```sql
-- name: GetFeedTokenByToken :one
SELECT * FROM feed_tokens WHERE token = $1 AND revoked_at IS NULL;

-- name: ListFeedTokensByUser :many
SELECT * FROM feed_tokens WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreateFeedToken :one
INSERT INTO feed_tokens (user_id, scope, scope_id, token)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: RevokeFeedToken :exec
UPDATE feed_tokens SET revoked_at = now() WHERE id = $1;
```

- [ ] **Step 3: Write external source queries**

```sql
-- name: GetExternalSourceByID :one
SELECT * FROM external_sources WHERE id = $1;

-- name: ListExternalSourcesByCalendar :many
SELECT * FROM external_sources WHERE calendar_id = $1;

-- name: ListEnabledExternalSources :many
SELECT * FROM external_sources WHERE enabled = true;

-- name: CreateExternalSource :one
INSERT INTO external_sources (name, feed_url, calendar_id, refresh_interval, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateExternalSourceFetchStatus :exec
UPDATE external_sources SET
    last_fetched_at = now(),
    last_error = $2,
    updated_at = now()
WHERE id = $1;

-- name: UpsertExternalEntry :one
INSERT INTO external_entries (external_source_id, uid, summary, starts_at, ends_at, location, description, raw_ical, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (external_source_id, uid) DO UPDATE SET
    summary = EXCLUDED.summary,
    starts_at = EXCLUDED.starts_at,
    ends_at = EXCLUDED.ends_at,
    location = EXCLUDED.location,
    description = EXCLUDED.description,
    raw_ical = EXCLUDED.raw_ical,
    fetched_at = now()
RETURNING *;

-- name: ListExternalEntriesBySource :many
SELECT * FROM external_entries WHERE external_source_id = $1 ORDER BY starts_at;
```

- [ ] **Step 4: Write notification queries**

```sql
-- name: GetNotificationConfigsByCalendar :many
SELECT * FROM notification_configs WHERE calendar_id = $1;

-- name: UpsertNotificationConfig :one
INSERT INTO notification_configs (calendar_id, event_type, enabled, lead_time)
VALUES ($1, $2, $3, $4)
ON CONFLICT (calendar_id, event_type) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    lead_time = EXCLUDED.lead_time
RETURNING *;

-- name: GetUserNotificationPreferences :many
SELECT * FROM user_notification_preferences WHERE user_id = $1;

-- name: UpsertUserNotificationPreference :one
INSERT INTO user_notification_preferences (user_id, event_type, channel, enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, event_type, channel) DO UPDATE SET enabled = EXCLUDED.enabled
RETURNING *;

-- name: CreateNotification :one
INSERT INTO notifications (user_id, entry_id, event_type, channel, payload)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListPendingNotifications :many
SELECT * FROM notifications
WHERE status IN ('pending', 'retrying')
ORDER BY created_at
LIMIT $1;

-- name: UpdateNotificationStatus :exec
UPDATE notifications SET
    status = $2,
    error = $3,
    retry_count = CASE WHEN $2 = 'retrying' THEN retry_count + 1 ELSE retry_count END,
    sent_at = CASE WHEN $2 = 'sent' THEN now() ELSE sent_at END
WHERE id = $1;
```

- [ ] **Step 5: Write webhook queries**

```sql
-- name: ListWebhooksByUnit :many
SELECT * FROM webhooks WHERE unit_id = $1 AND enabled = true;

-- name: ListAssociationWebhooks :many
SELECT * FROM webhooks WHERE unit_id IS NULL AND enabled = true;

-- name: CreateWebhook :one
INSERT INTO webhooks (unit_id, name, url, secret, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhooks WHERE id = $1;
```

- [ ] **Step 6: Run sqlc generate and verify everything compiles**

```bash
sqlc generate
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/store/queries/ internal/db/
git commit -m "feat: add template, integration, and notification queries"
```

---

### Task 9: Update auth middleware for new user schema

**Files:**
- Modify: `internal/auth/auth.go`

**Context:** The auth middleware was stubbed out in TASK-002 when the old store.GetOrCreateUser was removed. Now we have sqlc-generated query functions. Rewire auth to use them.

- [ ] **Step 1: Read current auth.go stub**

Read `internal/auth/auth.go` and understand the current stub.

- [ ] **Step 2: Update auth middleware to use sqlc queries**

The middleware needs to:
1. Extract user identity from X-Forwarded headers (idp_subject, email, display_name, groups)
2. Call `db.UpsertUser` to create/update the user
3. Call `db.DeleteUserIDPGroups` then `db.InsertUserIDPGroup` for each group (sync groups)
4. Determine `is_assoc_admin` from the configured admin group
5. Store the user in request context

The exact implementation depends on how auth.go is currently structured. Read it first, then make the minimum changes to use the new sqlc-generated `db.Queries` type instead of the old store pattern.

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/auth/auth.go
git commit -m "feat: rewire auth middleware to use sqlc-generated queries"
```

---

### Task 10: Remove internal/model/ directory and clean up imports

**Files:**
- Delete: `internal/model/` directory (if it still exists)
- Modify: any files importing `internal/model`

- [ ] **Step 1: Check if internal/model/ still exists**

```bash
ls internal/model/ 2>/dev/null
```

If it exists, check what's in it and whether anything imports it.

- [ ] **Step 2: Remove and fix imports**

If the directory exists and is empty or contains only the deleted user.go, remove it. If other files import from `internal/model`, update them to import from `internal/db` instead.

```bash
rm -rf internal/model/
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "chore: remove internal/model/, all types now from sqlc in internal/db/"
```

---

## Verification Checklist

After all tasks, verify:
- [ ] `sqlc generate` produces no errors
- [ ] `go build ./...` compiles cleanly
- [ ] `internal/db/models.go` has structs for all 24 tables
- [ ] Query files exist for all major entity operations
- [ ] Auth middleware compiles and uses sqlc queries
- [ ] No references to `internal/model/` remain
