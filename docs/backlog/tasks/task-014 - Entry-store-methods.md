---
id: TASK-014
title: Entry store methods
status: In Progress
assignee: []
created_date: '2026-03-16 14:33'
updated_date: '2026-03-21 22:23'
labels:
  - backend
milestone: m-3
dependencies:
  - TASK-003
priority: medium
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement store-layer methods for entries in internal/store/. Follow the existing DBTX pattern.

Required store methods:
- CreateEntry(ctx, db, entry) — insert a new entry
- UpdateEntry(ctx, db, entry) — update entry properties
- DeleteEntry(ctx, db, id) — delete entry (attendance cascades via FK)
- GetEntryByID(ctx, db, id) — get a single entry with calendar info
- GetEntryForUpdate(ctx, db, id) — get entry with SELECT FOR UPDATE (for transactional attendance operations)
- ListEntriesByCalendar(ctx, db, calendarID, startDate, endDate) — list entries in a date range, ordered by start_at
- ListEntriesByUnit(ctx, db, unitID, startDate, endDate) — list entries across all of a unit's calendars
- ListEntriesByUser(ctx, db, userID, startDate, endDate) — list entries the user has accepted or is pending on (for personal dashboard)

Each method should return model.Entry structs. The ListEntries methods should support both shift and meeting types — callers filter by type if needed.

The date range queries are critical for performance since they power all calendar views. Ensure they use the idx_entries_calendar_start index effectively.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 CRUD methods for entries work correctly for both shift and meeting types
- [ ] #2 Date range queries return entries ordered by start_at
- [ ] #3 GetEntryForUpdate uses SELECT FOR UPDATE for safe concurrent access
- [ ] #4 ListEntriesByUser returns entries based on attendance records
- [ ] #5 Follows existing DBTX pattern
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
# TASK-014: Entry Store Methods — Implementation Plan

## 1. Context Summary

### Architecture (main repo, post-TASK-003)

The project has undergone an architectural shift since the worktree was created. The main repo now uses **sqlc-generated code** for all database access:

- **`internal/db/`** contains sqlc-generated types (`models.go`) and query methods (e.g., `entries.sql.go`, `attendances.sql.go`). The generated `db.Queries` struct wraps a `DBTX` interface and provides typed methods for every SQL query.
- **`internal/store/store.go`** has been simplified. It no longer defines its own `DBTX` interface. Instead, it holds a `*pgxpool.Pool` and a `*db.Queries`. The old package-level function pattern (`GetOrCreateUser(ctx, db, ...)`) from `user.go` has been removed; all database access goes through `s.Queries().<Method>()`.
- **`Store.WithTx`** now passes both `pgx.Tx` and `*db.Queries` to the callback: `fn(tx pgx.Tx, q *db.Queries) error`. Inside transactions, callers use the transaction-scoped `q` for type-safe queries and can fall back to `tx` for raw SQL when needed.

### Existing sqlc Queries for Entries

The file `internal/store/queries/entries.sql` already defines these sqlc queries:

| Query Name | Type | Purpose |
|---|---|---|
| `GetEntryByID` | `:one` | SELECT by id |
| `GetEntryBySlug` | `:one` | SELECT by slug |
| `ListEntriesByCalendarAndDateRange` | `:many` | Filter by calendar_id + starts_at range |
| `ListEntriesByDateRange` | `:many` | Filter by starts_at range (no calendar filter) |
| `ListEntriesByRecurrenceRule` | `:many` | Filter by recurrence_rule_id |
| `CreateEntry` | `:one` | INSERT with RETURNING |
| `UpdateEntry` | `:one` | UPDATE (name, starts_at, ends_at, location, description, response_deadline) |
| `DeleteEntry` | `:exec` | DELETE by id |
| `GetEntryShiftDetails` | `:one` | Get shift-specific row |
| `UpsertEntryShiftDetails` | `:one` | Upsert shift details |
| `GetMeetingAudienceUnits` | `:many` | List audience unit IDs |
| `DeleteMeetingAudienceUnits` | `:exec` | Clear audience |
| `InsertMeetingAudienceUnit` | `:exec` | Add audience unit |
| `ListEntryAnnotations` | `:many` | Get annotations |
| `CreateEntryAnnotation` | `:one` | Add annotation |

### What the Existing Queries Do NOT Cover

The task requires several queries that are **not yet in sqlc**:

1. **`GetEntryForUpdate`** — `SELECT ... FOR UPDATE` is not present.
2. **`ListEntriesByUnit`** — Requires a JOIN through `calendars` to resolve `unit_id`. No such query exists.
3. **`ListEntriesByUser`** — Requires a JOIN through `attendances` to find entries where the user has `accepted` or `pending` status. No such query exists.
4. **`GetEntryByID` with calendar info** — The task says "get a single entry with calendar info." The existing `GetEntryByID` returns only entry columns, not joined calendar data.

### Schema (from migrations)

**`entries` table** (`migrations/00005_entries.sql`):
- `id BIGSERIAL PRIMARY KEY`
- `slug TEXT NOT NULL UNIQUE`
- `calendar_id BIGINT NOT NULL REFERENCES calendars(id) ON DELETE CASCADE`
- `name TEXT NOT NULL`
- `type TEXT NOT NULL CHECK (type IN ('shift', 'meeting'))`
- `starts_at TIMESTAMPTZ NOT NULL`
- `ends_at TIMESTAMPTZ NOT NULL`
- `location TEXT` (nullable)
- `description TEXT` (nullable)
- `response_deadline TIMESTAMPTZ` (nullable)
- `recurrence_rule_id BIGINT REFERENCES recurrence_rules(id) ON DELETE SET NULL` (nullable)
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `CHECK (ends_at > starts_at)`

**Indexes on `entries`:**
- `idx_entries_calendar_starts` — `(calendar_id, starts_at)` — the critical composite index
- `idx_entries_starts_at` — `(starts_at)` — for queries not filtering by calendar
- `idx_entries_recurrence_rule_id` — `(recurrence_rule_id)`
- `idx_entries_idempotency` — `UNIQUE (calendar_id, name, starts_at)` — for template instantiation dedup

**`attendances` table** (`migrations/00007_attendance.sql`):
- `idx_attendances_user_status` — `(user_id, status)` — used by ListEntriesByUser

**`calendars` table** (`migrations/00003_calendars.sql`):
- `idx_calendars_unit_id` — `(unit_id)` — used by ListEntriesByUnit

### Go Types (from `internal/db/models.go`)

The `db.Entry` struct uses `pgtype` wrappers for nullable fields:
```go
type Entry struct {
    ID               int64
    Slug             string
    CalendarID       int64
    Name             string
    Type             string
    StartsAt         pgtype.Timestamptz
    EndsAt           pgtype.Timestamptz
    Location         pgtype.Text
    Description      pgtype.Text
    ResponseDeadline pgtype.Timestamptz
    RecurrenceRuleID pgtype.Int8
    CreatedAt        pgtype.Timestamptz
    UpdatedAt        pgtype.Timestamptz
}
```

The `db.Calendar` struct includes `Slug`, `UnitID`, `Name`, `Color`, and other fields needed for calendar context in entry views.

### Design Decisions (from memory/project_schema_decisions.md)

- **Data access:** sqlc (code generation from SQL). Real SQL, no ORM.
- **PKs:** BIGSERIAL internally + slug for URL-facing entities (entries have slugs).
- **Enums:** TEXT columns with CHECK constraints, not native PG enums.
- **Deletion:** Hard delete with CASCADE.

---

## 2. File Changes

### Files to Modify

| File | Change |
|---|---|
| `internal/store/queries/entries.sql` | Add 3 new sqlc queries: `GetEntryForUpdate`, `ListEntriesByUnit`, `ListEntriesByUser`. Modify `GetEntryByID` to optionally join calendar info (see discussion below). |
| `internal/db/entries.sql.go` | Regenerated by sqlc (DO NOT EDIT manually). |
| `internal/db/models.go` | Regenerated by sqlc if new row types are produced. |

### Files to Create

| File | Purpose |
|---|---|
| `internal/store/entry.go` | Store-layer wrapper functions that compose sqlc queries into the higher-level methods required by TASK-014. This is where `model.Entry` structs are assembled from `db.Entry` + `db.Calendar` + shift/meeting details when needed. |
| `internal/model/entry.go` | The `model.Entry` struct that handlers and templates work with. This is the domain-level type, distinct from the sqlc-generated `db.Entry`. |

### Rationale: Why a Separate `model.Entry`?

The task says "each method should return `model.Entry` structs." The existing `db.Entry` is a 1:1 database row mapping with `pgtype` wrappers. A `model.Entry` provides:

1. Ergonomic Go types (`time.Time` instead of `pgtype.Timestamptz`, `*string` instead of `pgtype.Text`).
2. A place to attach joined data (calendar name, calendar color, calendar slug) without modifying the sqlc-generated type.
3. A place to include shift details (required/max participants) or meeting audience when relevant.

The `internal/store/entry.go` functions convert between `db.Entry` and `model.Entry`.

---

## 3. Model Type

### `internal/model/entry.go`

```go
package model

import "time"

// EntryType represents the type of calendar entry.
type EntryType string

const (
    EntryTypeShift   EntryType = "shift"
    EntryTypeMeeting EntryType = "meeting"
)

// Entry is the domain-level entry type returned by store methods.
type Entry struct {
    ID               int64
    Slug             string
    CalendarID       int64
    Name             string
    Type             EntryType
    StartsAt         time.Time
    EndsAt           time.Time
    Location         *string
    Description      *string
    ResponseDeadline *time.Time
    RecurrenceRuleID *int64
    CreatedAt        time.Time
    UpdatedAt        time.Time

    // Joined calendar context (populated by methods that join calendar data).
    CalendarName  string  // from calendars.name
    CalendarSlug  string  // from calendars.slug
    CalendarColor *string // from calendars.color
    UnitID        int64   // from calendars.unit_id

    // Shift-specific (populated when Type == "shift" and details are fetched).
    RequiredParticipants *int32
    MaxParticipants      *int32
}
```
<!-- SECTION:PLAN:END -->
