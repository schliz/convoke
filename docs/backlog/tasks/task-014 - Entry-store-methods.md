---
id: TASK-014
title: Entry store methods
status: In Progress
assignee: []
created_date: '2026-03-16 14:33'
updated_date: '2026-03-21 22:25'
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

---

## 4. Method Signatures

All methods are package-level functions in `internal/store/entry.go`, following the DBTX pattern. Since the project now uses sqlc, these methods accept `*db.Queries` (which wraps DBTX) rather than a raw DBTX interface.

However, looking at the main repo pattern, handlers call `s.Queries().<method>()` directly for simple operations. The store-layer functions in `entry.go` exist for operations that need to:
- Compose multiple sqlc queries (e.g., get entry + get shift details + get calendar info).
- Convert `db.Entry` to `model.Entry`.
- Execute raw SQL that sqlc cannot express (e.g., `SELECT FOR UPDATE`, complex JOINs).

```go
package store

import (
    "context"
    "time"

    "github.com/schliz/convoke/internal/db"
    "github.com/schliz/convoke/internal/model"
)

// CreateEntry inserts a new entry and its type-specific details.
// For shifts, it also inserts entry_shift_details.
// Returns the created entry as a model.Entry.
func CreateEntry(ctx context.Context, q *db.Queries, params CreateEntryParams) (*model.Entry, error)

// CreateEntryParams holds the input for CreateEntry.
type CreateEntryParams struct {
    Slug                 string
    CalendarID           int64
    Name                 string
    Type                 model.EntryType
    StartsAt             time.Time
    EndsAt               time.Time
    Location             *string
    Description          *string
    ResponseDeadline     *time.Time
    RecurrenceRuleID     *int64
    // Shift-specific (required when Type == EntryTypeShift)
    RequiredParticipants *int32
    MaxParticipants      *int32
}

// UpdateEntry updates an entry's mutable properties.
func UpdateEntry(ctx context.Context, q *db.Queries, params UpdateEntryParams) (*model.Entry, error)

// UpdateEntryParams holds the input for UpdateEntry.
type UpdateEntryParams struct {
    ID               int64
    Name             string
    StartsAt         time.Time
    EndsAt           time.Time
    Location         *string
    Description      *string
    ResponseDeadline *time.Time
}

// DeleteEntry removes an entry by ID. Attendance records cascade via FK.
func DeleteEntry(ctx context.Context, q *db.Queries, id int64) error

// GetEntryByID returns a single entry with calendar context.
func GetEntryByID(ctx context.Context, q *db.Queries, id int64) (*model.Entry, error)

// GetEntryBySlug returns a single entry by its slug with calendar context.
func GetEntryBySlug(ctx context.Context, q *db.Queries, slug string) (*model.Entry, error)

// GetEntryForUpdate returns an entry locked with SELECT FOR UPDATE.
// Must be called within a transaction (q should be transaction-scoped).
func GetEntryForUpdate(ctx context.Context, q *db.Queries, id int64) (*model.Entry, error)

// ListEntriesByCalendar returns entries in a calendar within a date range,
// ordered by starts_at. Uses idx_entries_calendar_starts.
func ListEntriesByCalendar(ctx context.Context, q *db.Queries, calendarID int64, start, end time.Time) ([]model.Entry, error)

// ListEntriesByUnit returns entries across all of a unit's calendars
// within a date range, ordered by starts_at.
func ListEntriesByUnit(ctx context.Context, q *db.Queries, unitID int64, start, end time.Time) ([]model.Entry, error)

// ListEntriesByUser returns entries the user has accepted or is pending on,
// within a date range, ordered by starts_at. Powers the personal dashboard.
func ListEntriesByUser(ctx context.Context, q *db.Queries, userID int64, start, end time.Time) ([]model.Entry, error)
```

---

## 5. SQL Queries

### 5.1 New sqlc Queries to Add to `entries.sql`

#### GetEntryWithCalendar (replaces plain GetEntryByID for store layer)

```sql
-- name: GetEntryWithCalendar :one
SELECT e.*,
       c.name AS calendar_name,
       c.slug AS calendar_slug,
       c.color AS calendar_color,
       c.unit_id AS unit_id
FROM entries e
JOIN calendars c ON e.calendar_id = c.id
WHERE e.id = $1;
```

#### GetEntryWithCalendarBySlug

```sql
-- name: GetEntryWithCalendarBySlug :one
SELECT e.*,
       c.name AS calendar_name,
       c.slug AS calendar_slug,
       c.color AS calendar_color,
       c.unit_id AS unit_id
FROM entries e
JOIN calendars c ON e.calendar_id = c.id
WHERE e.slug = $1;
```

#### GetEntryForUpdate

```sql
-- name: GetEntryForUpdate :one
SELECT e.*,
       c.name AS calendar_name,
       c.slug AS calendar_slug,
       c.color AS calendar_color,
       c.unit_id AS unit_id
FROM entries e
JOIN calendars c ON e.calendar_id = c.id
WHERE e.id = $1
FOR UPDATE OF e;
```

Key detail: `FOR UPDATE OF e` locks only the entry row, not the calendar row. This prevents unnecessary lock contention on the calendars table.

#### ListEntriesByUnit

```sql
-- name: ListEntriesByUnit :many
SELECT e.*,
       c.name AS calendar_name,
       c.slug AS calendar_slug,
       c.color AS calendar_color,
       c.unit_id AS unit_id
FROM entries e
JOIN calendars c ON e.calendar_id = c.id
WHERE c.unit_id = $1
  AND e.starts_at >= $2
  AND e.starts_at < $3
ORDER BY e.starts_at;
```

#### ListEntriesByUser

```sql
-- name: ListEntriesByUser :many
SELECT e.*,
       c.name AS calendar_name,
       c.slug AS calendar_slug,
       c.color AS calendar_color,
       c.unit_id AS unit_id
FROM entries e
JOIN calendars c ON e.calendar_id = c.id
JOIN attendances a ON a.entry_id = e.id
WHERE a.user_id = $1
  AND a.status IN ('accepted', 'pending')
  AND e.starts_at >= $2
  AND e.starts_at < $3
ORDER BY e.starts_at;
```

#### ListEntriesByCalendarWithCalendar (enhanced version of existing)

```sql
-- name: ListEntriesByCalendarWithCalendar :many
SELECT e.*,
       c.name AS calendar_name,
       c.slug AS calendar_slug,
       c.color AS calendar_color,
       c.unit_id AS unit_id
FROM entries e
JOIN calendars c ON e.calendar_id = c.id
WHERE e.calendar_id = $1
  AND e.starts_at >= $2
  AND e.starts_at < $3
ORDER BY e.starts_at;
```

### 5.2 Keeping Existing sqlc Queries

The existing queries (`GetEntryByID`, `CreateEntry`, `UpdateEntry`, `DeleteEntry`, etc.) should be kept as-is. They remain useful for internal operations that do not need calendar context. The new "WithCalendar" variants are additive.

---

## 6. Index Usage

### `idx_entries_calendar_starts (calendar_id, starts_at)`

This is the most important index. It is a composite B-tree that supports:

- **`ListEntriesByCalendar`**: `WHERE calendar_id = $1 AND starts_at >= $2 AND starts_at < $3` — perfect index scan. The planner will use the leading `calendar_id` equality to find the right subtree, then range-scan on `starts_at`. This is the ideal access pattern.

- **`ListEntriesByUnit`**: `WHERE c.unit_id = $1 AND e.starts_at >= $2 AND e.starts_at < $3` — the planner will first identify the unit's calendars via `idx_calendars_unit_id`, then for each calendar, use `idx_entries_calendar_starts` to range-scan. With a small number of calendars per unit (typical: 1-5), this is efficient.

### `idx_entries_starts_at (starts_at)`

Used by:
- **`ListEntriesByUser`**: After joining attendances on `user_id + status` (using `idx_attendances_user_status`), the planner may use this index for the `starts_at` range filter, or it may scan the attendance index first and then filter by date. For typical data volumes (a user has tens of active attendances), the attendance join will be the leading filter and the starts_at filter is applied as a cheap post-join predicate. No concern here.

### `idx_attendances_user_status (user_id, status)`

Used by **`ListEntriesByUser`**: The `WHERE a.user_id = $1 AND a.status IN ('accepted', 'pending')` predicate uses this index effectively. PostgreSQL can use the index for the equality on `user_id` and then filter on `status`.

### `idx_calendars_unit_id (unit_id)`

Used by **`ListEntriesByUnit`**: The join `calendars c ON e.calendar_id = c.id WHERE c.unit_id = $1` uses this index to find the unit's calendars first.

---

## 7. Edge Cases

### Empty Date Ranges

When `start == end` or `start > end`:
- `starts_at >= start AND starts_at < end` returns zero rows when `start >= end`. This is correct behavior, not an error. The store methods should not validate or reject empty ranges; the caller is responsible for providing meaningful ranges.

### Entries Spanning Midnight

Entries are filtered on `starts_at`, not on both `starts_at` and `ends_at`. This means:
- An entry starting at 23:00 and ending at 02:00 the next day will appear in the date range that includes 23:00, but not in the range that includes 02:00.
- This is the correct behavior for calendar views: entries are anchored to their start time. If a view needs to show entries that are "active" during a time window (regardless of start), a different query would be needed (`WHERE starts_at < $end AND ends_at > $start`). This is explicitly out of scope for TASK-014 — the task description says "date range queries" filtering on `starts_at`.

### SELECT FOR UPDATE Deadlock Prevention

`GetEntryForUpdate` uses `FOR UPDATE OF e` to lock only the entry row. Deadlocks can occur if:
1. Two transactions lock entries in different order.
2. A transaction holding an entry lock tries to acquire another lock.

**Mitigation strategy:**
- The primary use case is attendance operations on a single entry. The pattern is: `GetEntryForUpdate(entry_id)` -> check capacity -> `CreateAttendance`. This locks one entry per transaction, so no deadlock is possible from entry locks alone.
- If future code needs to lock multiple entries, it must sort entry IDs and lock them in ascending order.
- The `FOR UPDATE OF e` clause (not `FOR UPDATE`) ensures the joined calendar row is not locked, preventing spurious lock conflicts when two transactions operate on entries in the same calendar.

### `pgx.ErrNoRows` Handling

`GetEntryByID`, `GetEntryBySlug`, and `GetEntryForUpdate` will return `pgx.ErrNoRows` when the entry does not exist. The store methods should propagate this error; handlers can check with `errors.Is(err, pgx.ErrNoRows)` and convert to a 404.

### Nullable Fields Conversion

The `db.Entry` uses `pgtype.Text` and `pgtype.Timestamptz` for nullable fields. The conversion to `model.Entry` (`*string`, `*time.Time`) must handle:
- `pgtype.Text{Valid: false}` -> `nil`
- `pgtype.Text{Valid: true, String: "..."}` -> `&"..."`
- Same pattern for `pgtype.Timestamptz` -> `*time.Time`
- And `pgtype.Int8{Valid: false}` -> `nil` for `RecurrenceRuleID`

A helper function should be written for these conversions to avoid repetition.

---

## 8. Implementation Details

### Conversion Helper Functions

```go
// in internal/store/entry.go

func textToPtr(t pgtype.Text) *string {
    if !t.Valid {
        return nil
    }
    return &t.String
}

func ptrToText(s *string) pgtype.Text {
    if s == nil {
        return pgtype.Text{}
    }
    return pgtype.Text{String: *s, Valid: true}
}

func tsToTime(ts pgtype.Timestamptz) time.Time {
    return ts.Time
}

func tsToPtr(ts pgtype.Timestamptz) *time.Time {
    if !ts.Valid {
        return nil
    }
    return &ts.Time
}

func ptrToTs(t *time.Time) pgtype.Timestamptz {
    if t == nil {
        return pgtype.Timestamptz{}
    }
    return pgtype.Timestamptz{Time: *t, Valid: true}
}

func int8ToPtr(i pgtype.Int8) *int64 {
    if !i.Valid {
        return nil
    }
    return &i.Int64
}

func ptrToInt8(i *int64) pgtype.Int8 {
    if i == nil {
        return pgtype.Int8{}
    }
    return pgtype.Int8{Int64: *i, Valid: true}
}

func timeToTs(t time.Time) pgtype.Timestamptz {
    return pgtype.Timestamptz{Time: t, Valid: true}
}
```

### Entry Conversion from sqlc Row to model.Entry

The new sqlc queries that join calendar data will produce custom row types (sqlc generates these automatically). A conversion function maps from the generated row type to `model.Entry`:

```go
func entryFromRow(row db.GetEntryWithCalendarRow) model.Entry {
    return model.Entry{
        ID:               row.ID,
        Slug:             row.Slug,
        CalendarID:       row.CalendarID,
        Name:             row.Name,
        Type:             model.EntryType(row.Type),
        StartsAt:         tsToTime(row.StartsAt),
        EndsAt:           tsToTime(row.EndsAt),
        Location:         textToPtr(row.Location),
        Description:      textToPtr(row.Description),
        ResponseDeadline: tsToPtr(row.ResponseDeadline),
        RecurrenceRuleID: int8ToPtr(row.RecurrenceRuleID),
        CreatedAt:        tsToTime(row.CreatedAt),
        UpdatedAt:        tsToTime(row.UpdatedAt),
        CalendarName:     row.CalendarName,
        CalendarSlug:     row.CalendarSlug,
        CalendarColor:    textToPtr(row.CalendarColor),
        UnitID:           row.UnitID,
    }
}
```

Because sqlc generates a distinct row type for each query that has a non-trivial SELECT list, there will be multiple generated row types (e.g., `GetEntryWithCalendarRow`, `ListEntriesByUnitRow`, `ListEntriesByUserRow`). However, they all have the same fields (entry columns + calendar_name + calendar_slug + calendar_color + unit_id). If sqlc generates identical struct shapes, a single conversion function works. If not, multiple thin adapters are needed, or we use a common interface/generics.

**Practical approach:** Define `entryFromRow` as a generic-style function or create one per row type. Given the small number of variants, per-type functions are clearest.

### Method Implementations (Pseudocode)

**CreateEntry:**
1. Convert `CreateEntryParams` to `db.CreateEntryParams` (with pgtype conversions).
2. Call `q.CreateEntry(ctx, ...)`.
3. If `Type == EntryTypeShift` and `RequiredParticipants != nil`, call `q.UpsertEntryShiftDetails(ctx, ...)`.
4. Fetch the full entry with calendar context via `q.GetEntryWithCalendar(ctx, entry.ID)`.
5. Convert to `model.Entry` and return.

**UpdateEntry:**
1. Convert `UpdateEntryParams` to `db.UpdateEntryParams`.
2. Call `q.UpdateEntry(ctx, ...)`.
3. Fetch the full entry with calendar context.
4. Convert and return.

**DeleteEntry:**
1. Call `q.DeleteEntry(ctx, id)`.
2. Cascading FKs handle attendance and shift detail deletion.

**GetEntryByID:**
1. Call `q.GetEntryWithCalendar(ctx, id)`.
2. If entry type is "shift", also call `q.GetEntryShiftDetails(ctx, id)` and populate RequiredParticipants/MaxParticipants.
3. Convert and return.

**GetEntryForUpdate:**
1. Call `q.GetEntryForUpdate(ctx, id)` — this executes the `SELECT ... FOR UPDATE OF e` query.
2. Convert and return. No shift detail fetch needed here; the caller typically only needs the lock and basic entry data for the attendance operation.

**ListEntriesByCalendar:**
1. Call `q.ListEntriesByCalendarWithCalendar(ctx, ...)` with `timeToTs(start)` and `timeToTs(end)`.
2. Convert each row to `model.Entry`.
3. Return the slice.

**ListEntriesByUnit:**
1. Call `q.ListEntriesByUnit(ctx, ...)`.
2. Convert and return.

**ListEntriesByUser:**
1. Call `q.ListEntriesByUser(ctx, ...)`.
2. Convert and return.

---

## 9. Testing Strategy

### Unit Tests (`internal/store/entry_test.go`)

These require a live PostgreSQL database. The project does not currently have a test database setup, but the pattern should be prepared:

1. **TestCreateEntry_Shift** — Create an entry with type "shift", verify shift details are persisted, verify returned model has correct fields.
2. **TestCreateEntry_Meeting** — Create an entry with type "meeting", verify no shift details, verify all nullable fields (location, description, response_deadline) are correctly handled when nil and when set.
3. **TestUpdateEntry** — Update fields and verify the returned model reflects changes, verify updated_at changes.
4. **TestDeleteEntry** — Create an entry with attendances, delete it, verify cascading deletion.
5. **TestGetEntryByID** — Verify calendar context fields are populated, verify shift details are populated for shifts.
6. **TestGetEntryByID_NotFound** — Verify `pgx.ErrNoRows` is returned.
7. **TestGetEntryForUpdate** — Verify locking within a transaction (create two transactions, lock in first, verify second blocks or returns expected behavior).
8. **TestListEntriesByCalendar** — Insert entries across multiple calendars and date ranges, verify filtering and ordering.
9. **TestListEntriesByCalendar_EmptyRange** — Start == end returns empty slice.
10. **TestListEntriesByUnit** — Insert entries in multiple calendars of one unit and one entry in another unit's calendar, verify only the correct unit's entries are returned.
11. **TestListEntriesByUser** — Create entries with various attendance statuses, verify only accepted/pending entries in the date range are returned.
12. **TestListEntriesByUser_NoAttendances** — Verify empty slice returned, not error.

### Conversion Tests

13. **TestTextToPtr** — Verify nil and non-nil cases.
14. **TestTsToPtr** — Verify nil and non-nil cases.

### Test Setup

Each test should:
1. Use a shared test database with applied migrations.
2. Run within a transaction that is rolled back after the test (for isolation).
3. Seed required parent rows (users, units, calendars) before creating entries.

---

## 10. Open Questions

### Q1: Should `model.Entry` be a flat struct or embed calendar info in a sub-struct?

The plan proposes a flat struct with `CalendarName`, `CalendarSlug`, `CalendarColor`, `UnitID` fields. An alternative is:

```go
type Entry struct {
    // ... entry fields ...
    Calendar *CalendarInfo // nil when not joined
}

type CalendarInfo struct {
    Name  string
    Slug  string
    Color *string
    UnitID int64
}
```

The flat approach is simpler and matches the sqlc row output. The nested approach is cleaner semantically. **Recommendation: start flat, refactor if needed.**

### Q2: Should `GetEntryByID` always fetch shift details?

The plan proposes an extra query for shift details when `Type == "shift"`. This is 2 queries per call. Alternative: add a LEFT JOIN on `entry_shift_details` to the main query so it is always 1 query. However, sqlc does not handle LEFT JOINs with optional columns elegantly (nullable fields for shift details even when the entry is a shift). **Recommendation: use 2 queries. The overhead is negligible (one extra indexed PK lookup) and the code is clearer.**

### Q3: Should the store methods accept `*db.Queries` or should they be methods on `*Store`?

The main repo's `Store` now wraps `*db.Queries`. The task description says "package-level functions accepting DBTX as first arg." The project has moved away from that pattern. The implementation should use `*db.Queries` as the receiver of the sqlc-generated functions, and the new store functions should accept `*db.Queries` as a parameter (enabling both pool-scoped and transaction-scoped use). **This matches how `auth.go` calls `s.Queries().UpsertUser(...)` — the Queries are obtained from Store and passed around.**

### Q4: Should we add the `model.Entry` type now or wait for a broader model layer task?

The `internal/model/` directory currently only has `user.go` with a simple User struct (from before the sqlc migration). The main repo has moved to `db.Entry` as the primary type. Adding `model.Entry` introduces a new convention. **Recommendation: Yes, add it. The task explicitly requires returning `model.Entry` structs, and the ergonomic benefits (native Go types, joined data) justify the conversion layer. The old `model/user.go` should eventually be updated to match, but that is out of scope for this task.**
<!-- SECTION:PLAN:END -->
