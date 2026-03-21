---
id: TASK-009
title: Calendar store methods
status: In Progress
assignee: []
created_date: '2026-03-16 14:32'
updated_date: '2026-03-21 22:26'
labels:
  - backend
milestone: m-2
dependencies:
  - TASK-003
priority: medium
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement store-layer methods for calendars in internal/store/. Follow the existing DBTX pattern from store.go and user.go.

Required store methods:
- CreateCalendar(ctx, db, calendar) — insert a new calendar
- UpdateCalendar(ctx, db, calendar) — update calendar properties
- DeleteCalendar(ctx, db, id) — delete calendar (entries cascade via FK)
- GetCalendarByID(ctx, db, id) — get a single calendar with its unit info
- ListCalendarsByUnit(ctx, db, unitID) — list calendars for a unit, ordered by sort_order
- ListVisibleCalendars(ctx, db, userGroups, isAdmin) — list calendars visible to a user based on visibility settings (association-wide, unit membership, or custom unit list)

The visibility check is the most nuanced method: it must evaluate the calendar's visibility setting against the user's group memberships. For visibility='association', all authenticated users see it. For visibility='unit', only unit members. For visibility='custom', check against the explicit unit list.

All methods accept DBTX as a parameter (not a receiver method on Store).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 CRUD methods for calendars exist and work correctly
- [ ] #2 ListCalendarsByUnit returns calendars ordered by sort_order
- [ ] #3 ListVisibleCalendars correctly evaluates all three visibility modes (association, unit, custom)
- [ ] #4 Follows existing DBTX pattern in internal/store/
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
# TASK-009: Calendar Store Methods -- Implementation Plan

## 1. Context Summary

### Architecture Overview

The project uses a **sqlc-generated data access layer**. SQL queries live in
`internal/store/queries/*.sql`. Running `sqlc generate` produces Go code in
`internal/db/`, including:

- `db.go` -- `DBTX` interface and `Queries` struct (receiver for all
  generated methods).
- `models.go` -- one Go struct per database table (e.g., `db.Calendar`).
- `<table>.sql.go` -- one Go file per query file, containing the SQL
  constants, param structs, and methods.

The `Store` struct (`internal/store/store.go`) wraps a `*pgxpool.Pool` and
exposes `Queries()` for non-transactional use and `WithTx()` for
transactional use. Callers interact with the generated `*db.Queries` methods
directly -- there is **no hand-written store layer** between handlers and
sqlc. The task description says "all methods accept DBTX as a parameter (not
a receiver method on Store)" but the actual codebase pattern uses sqlc's
`*db.Queries` receiver methods, which already accept `DBTX` internally via
`db.New(dbtx)`. The implementation should follow what the codebase actually
does.

### Existing sqlc Queries for Calendars

TASK-003 already created `internal/store/queries/calendars.sql` with these
queries:

| sqlc name                      | Kind   | Purpose                              |
|-------------------------------|--------|--------------------------------------|
| `GetCalendarByID`             | `:one` | Get by PK                            |
| `GetCalendarBySlug`           | `:one` | Get by slug                          |
| `ListCalendarsByUnit`         | `:many`| List for a unit, ordered by sort_order, name |
| `CreateCalendar`              | `:one` | Insert, returns full row             |
| `UpdateCalendar`              | `:one` | Update mutable fields, returns full row |
| `DeleteCalendar`              | `:exec`| Delete by PK (entries cascade via FK) |
| `ListVisibleCalendarsForUser` | `:many`| Visibility-aware list by user ID     |
| `DeleteCalendarCustomViewers` | `:exec`| Clear custom viewers for a calendar  |
| `InsertCalendarCustomViewer`  | `:exec`| Add one custom viewer unit           |

The corresponding generated Go code already exists in
`internal/db/calendars.sql.go`. The generated `db.Calendar` struct, param
structs (`CreateCalendarParams`, `UpdateCalendarParams`,
`InsertCalendarCustomViewerParams`), and all method implementations are
complete.

### Key Schema Details

**`calendars` table** (from `migrations/00003_calendars.sql`):

- `id BIGSERIAL PRIMARY KEY` -- int64 in Go
- `slug TEXT NOT NULL UNIQUE` -- opaque URL-safe identifier
- `unit_id BIGINT NOT NULL REFERENCES units(id) ON DELETE CASCADE`
- `name TEXT NOT NULL`
- `creation_policy TEXT NOT NULL DEFAULT 'admins_only'` (CHECK constraint)
- `visibility TEXT NOT NULL DEFAULT 'association'` (CHECK: association | unit | custom)
- `participation TEXT NOT NULL DEFAULT 'viewers'` (CHECK: viewers | unit | nobody)
- `participant_visibility TEXT NOT NULL DEFAULT 'everyone'` (CHECK: everyone | unit | participants_only)
- `color TEXT` -- nullable, hex color
- `sort_order INT NOT NULL DEFAULT 0`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`calendar_custom_viewers` table**:

- `calendar_id BIGINT NOT NULL REFERENCES calendars(id) ON DELETE CASCADE`
- `unit_id BIGINT NOT NULL REFERENCES units(id) ON DELETE CASCADE`
- `PRIMARY KEY (calendar_id, unit_id)`

Used only when `visibility = 'custom'`. Lists the units whose members can
see the calendar.

**Enum values are plain TEXT with CHECK constraints** (not PostgreSQL native
ENUMs), per ADR in memory. pgx scans them as `string` -- no type
registration needed.

**Nullable columns** use `pgtype.Text`, `pgtype.Timestamptz`, etc. (pgx v5
pgtype wrappers). Non-nullable columns use native Go types (`string`,
`int64`, `int32`, `bool`).

### Visibility Resolution Logic

The existing `ListVisibleCalendarsForUser` query (already in
`calendars.sql`) implements the three-tier visibility check:

1. **`association`** -- always visible to any authenticated user. The query
   includes `WHERE c.visibility = 'association'` unconditionally.

2. **`unit`** -- visible only to members of the owning unit. Resolved by
   joining `unit_group_bindings` (unit -> IdP group) with `user_idp_groups`
   (user -> IdP group): if any group overlaps, the user is a member.

3. **`custom`** -- visible to members of explicitly listed units. Resolved
   by joining `calendar_custom_viewers` -> `unit_group_bindings` ->
   `user_idp_groups`.

The query takes a single `$1` parameter: the user's database ID (int64).
Group membership is resolved entirely in SQL via the normalized junction
tables, not by passing groups as a parameter.

**Admin bypass**: The existing query does NOT include an admin override.
Association admins should see all calendars regardless of visibility. This
needs to be handled either:
- (a) In the query itself (add `OR $2 = true` with an isAdmin param), or
- (b) At the handler layer (admins call `ListCalendarsByUnit` or a
  list-all query instead).

Option (b) is cleaner and matches the existing pattern where admin checks
happen in handler/middleware code, not in SQL.

### Related Tables and Cascade Behavior

Deleting a calendar cascades to:
- `entries` (via `entries.calendar_id` ON DELETE CASCADE)
- `calendar_custom_viewers` (via `calendar_custom_viewers.calendar_id` ON DELETE CASCADE)
- `event_calendars` (via `event_calendars.calendar_id` ON DELETE CASCADE)
- `template_groups` (via `template_groups.calendar_id` -- needs verification, but the FK exists)
- `notification_configs` (via `notification_configs.calendar_id`)

All cascades are handled by PostgreSQL foreign keys. The store method just
issues `DELETE FROM calendars WHERE id = $1`.

### Mismatch Between Task Description and Codebase

The task description specifies six free-standing functions that accept
`DBTX` as a parameter. However, the codebase has evolved since the task
was written:

- **TASK-003** introduced sqlc, which generates `*db.Queries` receiver
  methods (not free-standing functions).
- The `Store` struct no longer exposes a `DB() DBTX` method. It exposes
  `Queries() *db.Queries` and `WithTx()`.
- The auth middleware calls `s.Queries().UpsertUser(...)` directly.

**The sqlc-generated code already implements all six required methods.**
The question is what additional work TASK-009 needs.

## 2. Implementation Options

### Option A: TASK-009 Is Already Done (Recommend Confirming)

All six methods from the task description map to existing sqlc-generated
code:

| Task Requirement                        | sqlc Method                          | Status |
|----------------------------------------|--------------------------------------|--------|
| `CreateCalendar(ctx, db, calendar)`    | `q.CreateCalendar(ctx, params)`      | Done   |
| `UpdateCalendar(ctx, db, calendar)`    | `q.UpdateCalendar(ctx, params)`      | Done   |
| `DeleteCalendar(ctx, db, id)`          | `q.DeleteCalendar(ctx, id)`          | Done   |
| `GetCalendarByID(ctx, db, id)`         | `q.GetCalendarByID(ctx, id)`         | Done   |
| `ListCalendarsByUnit(ctx, db, unitID)` | `q.ListCalendarsByUnit(ctx, unitID)` | Done   |
| `ListVisibleCalendars(ctx, db, ...)`   | `q.ListVisibleCalendarsForUser(ctx, userID)` | Done |

Plus helper methods: `GetCalendarBySlug`, `DeleteCalendarCustomViewers`,
`InsertCalendarCustomViewer`.

If the team considers sqlc-generated methods as the "store layer", then
TASK-009 is already satisfied by TASK-003 output and just needs
verification/testing.

### Option B: Add a Thin Wrapper Layer in `internal/store/calendar.go`

If the team wants application-level store functions that compose multiple
sqlc calls (e.g., create calendar + set custom viewers in one call, or
get calendar with unit info joined), then a wrapper file is needed.

This is the more likely intent, given:
- The task mentions "get a single calendar **with its unit info**" -- the
  existing `GetCalendarByID` returns only calendar columns, not unit name.
- Managing custom viewers requires multiple sqlc calls (delete all, then
  insert each), which is a transaction concern.
- The task mentions `ListVisibleCalendars(ctx, db, userGroups, isAdmin)`
  with an admin bypass, which the current query doesn't handle.

**Recommendation: Implement Option B** -- a thin `internal/store/calendar.go`
that wraps sqlc queries and adds:
1. Joined result types (calendar + unit name)
2. Transactional custom viewer management
3. Admin-aware visibility listing
4. Slug generation for new calendars

## 3. File Changes

### Files to Create

| File | Purpose |
|------|---------|
| `internal/store/calendar.go` | Calendar store methods wrapping sqlc queries |

### Files to Potentially Modify

| File | Change |
|------|--------|
| `internal/store/queries/calendars.sql` | Add query for calendar-with-unit-info join; possibly an admin-all-calendars query |
| `internal/db/calendars.sql.go` | Re-generated by `sqlc generate` after query changes |
| `internal/db/models.go` | Re-generated (unlikely to change unless new tables added) |

### Files NOT Modified

- `internal/store/store.go` -- no changes needed, existing `Queries()` and `WithTx()` suffice.
- `internal/db/db.go` -- generated, never hand-edited.

## 4. New sqlc Queries to Add

### `GetCalendarWithUnit` -- Calendar joined with unit info

```sql
-- name: GetCalendarWithUnit :one
SELECT c.id, c.slug, c.unit_id, c.name, c.creation_policy, c.visibility,
    c.participation, c.participant_visibility, c.color, c.sort_order,
    c.created_at, c.updated_at,
    u.name AS unit_name, u.slug AS unit_slug
FROM calendars c
JOIN units u ON c.unit_id = u.id
WHERE c.id = $1;
```

This satisfies the "get a single calendar with its unit info" requirement.
sqlc will generate a `GetCalendarWithUnitRow` struct containing all
calendar fields plus `UnitName` and `UnitSlug`.

### `ListAllCalendars` -- For admin users who bypass visibility

```sql
-- name: ListAllCalendars :many
SELECT c.id, c.slug, c.unit_id, c.name, c.creation_policy, c.visibility,
    c.participation, c.participant_visibility, c.color, c.sort_order,
    c.created_at, c.updated_at,
    u.name AS unit_name, u.slug AS unit_slug
FROM calendars c
JOIN units u ON c.unit_id = u.id
ORDER BY u.name, c.sort_order, c.name;
```

### `GetCustomViewerUnits` -- List units that are custom viewers of a calendar

```sql
-- name: GetCustomViewerUnits :many
SELECT u.id, u.name, u.slug
FROM calendar_custom_viewers ccv
JOIN units u ON ccv.unit_id = u.id
WHERE ccv.calendar_id = $1
ORDER BY u.name;
```

## 5. Method Signatures for `internal/store/calendar.go`

```go
package store

import (
    "context"
    "fmt"

    "github.com/schliz/convoke/internal/db"
)

// CreateCalendar inserts a new calendar and, if visibility is 'custom',
// sets the custom viewer units. Runs in a transaction.
func CreateCalendar(
    ctx context.Context,
    s *Store,
    params db.CreateCalendarParams,
    customViewerUnitIDs []int64,
) (db.Calendar, error)

// UpdateCalendar updates calendar properties and, if visibility is
// 'custom', replaces the custom viewer units. Runs in a transaction.
func UpdateCalendar(
    ctx context.Context,
    s *Store,
    params db.UpdateCalendarParams,
    customViewerUnitIDs []int64,
) (db.Calendar, error)

// DeleteCalendar deletes a calendar by ID. Entries, custom viewers,
// event_calendars, and other dependent rows cascade via FK.
func DeleteCalendar(
    ctx context.Context,
    q *db.Queries,
    id int64,
) error

// GetCalendarByID returns a calendar with its owning unit's name and slug.
func GetCalendarByID(
    ctx context.Context,
    q *db.Queries,
    id int64,
) (db.GetCalendarWithUnitRow, error)

// GetCalendarBySlug returns a calendar by its URL slug.
func GetCalendarBySlug(
    ctx context.Context,
    q *db.Queries,
    slug string,
) (db.Calendar, error)

// ListCalendarsByUnit returns all calendars for a unit, ordered by
// sort_order then name.
func ListCalendarsByUnit(
    ctx context.Context,
    q *db.Queries,
    unitID int64,
) ([]db.Calendar, error)

// ListVisibleCalendars returns calendars visible to a user. If isAdmin
// is true, returns all calendars (admin bypass). Otherwise evaluates
// visibility rules against the user's group memberships via the
// normalized junction tables.
func ListVisibleCalendars(
    ctx context.Context,
    q *db.Queries,
    userID int64,
    isAdmin bool,
) ([]db.Calendar, error)
```

### Design Decisions in the Signatures

1. **`CreateCalendar` and `UpdateCalendar` take `*Store`** (not
   `*db.Queries`) because they need `WithTx()` to atomically manage the
   calendar row and its custom viewers.

2. **Read-only methods take `*db.Queries`** because they are single
   queries and can work with either pool or transaction-scoped Queries.

3. **`DeleteCalendar` takes `*db.Queries`** -- it's a single DELETE and
   cascading is handled by the database.

4. **`ListVisibleCalendars` takes `userID int64` and `isAdmin bool`**
   rather than `userGroups []string`. The visibility query resolves groups
   entirely in SQL via the `user_idp_groups` junction table. Passing
   group strings would require an `ANY($1::text[])` array parameter which
   is less clean. The `isAdmin` flag simply selects which query to run.

5. **Custom viewer unit IDs are passed as `[]int64`** alongside the
   calendar params. This avoids needing a separate "SetCustomViewers"
   public method while keeping the transaction boundary clear.

## 6. Implementation Details

### `CreateCalendar` Implementation

```go
func CreateCalendar(
    ctx context.Context,
    s *Store,
    params db.CreateCalendarParams,
    customViewerUnitIDs []int64,
) (db.Calendar, error) {
    var cal db.Calendar
    err := s.WithTx(ctx, func(tx pgx.Tx, q *db.Queries) error {
        var err error
        cal, err = q.CreateCalendar(ctx, params)
        if err != nil {
            return fmt.Errorf("create calendar: %w", err)
        }
        if params.Visibility == "custom" {
            for _, unitID := range customViewerUnitIDs {
                if err := q.InsertCalendarCustomViewer(ctx, db.InsertCalendarCustomViewerParams{
                    CalendarID: cal.ID,
                    UnitID:     unitID,
                }); err != nil {
                    return fmt.Errorf("insert custom viewer: %w", err)
                }
            }
        }
        return nil
    })
    return cal, err
}
```

### `UpdateCalendar` Implementation

```go
func UpdateCalendar(
    ctx context.Context,
    s *Store,
    params db.UpdateCalendarParams,
    customViewerUnitIDs []int64,
) (db.Calendar, error) {
    var cal db.Calendar
    err := s.WithTx(ctx, func(tx pgx.Tx, q *db.Queries) error {
        var err error
        cal, err = q.UpdateCalendar(ctx, params)
        if err != nil {
            return fmt.Errorf("update calendar: %w", err)
        }
        // Always clear and re-set custom viewers (idempotent).
        if err := q.DeleteCalendarCustomViewers(ctx, cal.ID); err != nil {
            return fmt.Errorf("delete custom viewers: %w", err)
        }
        if params.Visibility == "custom" {
            for _, unitID := range customViewerUnitIDs {
                if err := q.InsertCalendarCustomViewer(ctx, db.InsertCalendarCustomViewerParams{
                    CalendarID: cal.ID,
                    UnitID:     unitID,
                }); err != nil {
                    return fmt.Errorf("insert custom viewer: %w", err)
                }
            }
        }
        return nil
    })
    return cal, err
}
```

### `ListVisibleCalendars` Implementation

```go
func ListVisibleCalendars(
    ctx context.Context,
    q *db.Queries,
    userID int64,
    isAdmin bool,
) ([]db.Calendar, error) {
    if isAdmin {
        return q.ListAllCalendars(ctx)
    }
    return q.ListVisibleCalendarsForUser(ctx, userID)
}
```

**Note on return types**: `ListAllCalendars` as defined above returns a
joined row type (`ListAllCalendarsRow`), while `ListVisibleCalendarsForUser`
returns `[]db.Calendar`. To unify the return type, either:
- (a) Make `ListAllCalendars` return only calendar columns (no join), or
- (b) Add unit info to the visibility query too, or
- (c) Accept different return types for admin vs. non-admin paths.

**Recommended approach**: Option (a) -- add a simple
`ListAllCalendarsSimple` query that returns `SELECT * FROM calendars ORDER BY sort_order, name`
for admin bypass, keeping the return type as `[]db.Calendar`. The joined
query (`ListAllCalendars` with unit info) is available separately for
admin UI pages that need it.

### Simple pass-through methods

`DeleteCalendar`, `GetCalendarBySlug`, `ListCalendarsByUnit` are thin
wrappers that delegate directly to the sqlc method. They exist to provide
a consistent API surface and a place to add logging, error wrapping, or
validation in the future.

```go
func DeleteCalendar(ctx context.Context, q *db.Queries, id int64) error {
    return q.DeleteCalendar(ctx, id)
}

func GetCalendarBySlug(ctx context.Context, q *db.Queries, slug string) (db.Calendar, error) {
    return q.GetCalendarBySlug(ctx, slug)
}

func ListCalendarsByUnit(ctx context.Context, q *db.Queries, unitID int64) ([]db.Calendar, error) {
    return q.ListCalendarsByUnit(ctx, unitID)
}
```
<!-- SECTION:PLAN:END -->
