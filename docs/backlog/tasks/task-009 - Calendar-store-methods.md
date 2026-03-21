---
id: TASK-009
title: Calendar store methods
status: In Progress
assignee: []
created_date: '2026-03-16 14:32'
updated_date: '2026-03-21 22:28'
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

## 7. SQL Queries -- Complete Listing

### Existing queries (no changes needed)

Already in `internal/store/queries/calendars.sql`:

```sql
-- name: GetCalendarByID :one
SELECT * FROM calendars WHERE id = $1;

-- name: GetCalendarBySlug :one
SELECT * FROM calendars WHERE slug = $1;

-- name: ListCalendarsByUnit :many
SELECT * FROM calendars WHERE unit_id = $1 ORDER BY sort_order, name;

-- name: CreateCalendar :one
INSERT INTO calendars (slug, unit_id, name, creation_policy, visibility,
    participation, participant_visibility, color, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateCalendar :one
UPDATE calendars SET
    name = $2, creation_policy = $3, visibility = $4, participation = $5,
    participant_visibility = $6, color = $7, sort_order = $8, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteCalendar :exec
DELETE FROM calendars WHERE id = $1;

-- name: ListVisibleCalendarsForUser :many
SELECT DISTINCT c.id, c.slug, c.unit_id, c.name, c.creation_policy, c.visibility,
    c.participation, c.participant_visibility, c.color, c.sort_order,
    c.created_at, c.updated_at
FROM calendars c
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

-- name: DeleteCalendarCustomViewers :exec
DELETE FROM calendar_custom_viewers WHERE calendar_id = $1;

-- name: InsertCalendarCustomViewer :exec
INSERT INTO calendar_custom_viewers (calendar_id, unit_id) VALUES ($1, $2);
```

### New queries to add

```sql
-- name: GetCalendarWithUnit :one
SELECT c.id, c.slug, c.unit_id, c.name, c.creation_policy, c.visibility,
    c.participation, c.participant_visibility, c.color, c.sort_order,
    c.created_at, c.updated_at,
    u.name AS unit_name, u.slug AS unit_slug
FROM calendars c
JOIN units u ON c.unit_id = u.id
WHERE c.id = $1;

-- name: ListAllCalendars :many
SELECT * FROM calendars ORDER BY sort_order, name;

-- name: GetCustomViewerUnits :many
SELECT u.id, u.name, u.slug
FROM calendar_custom_viewers ccv
JOIN units u ON ccv.unit_id = u.id
WHERE ccv.calendar_id = $1
ORDER BY u.name;
```

## 8. Edge Cases

### Visibility Edge Cases

1. **Empty custom viewer list with `visibility='custom'`**: The calendar
   is invisible to everyone except admins. The `ListVisibleCalendarsForUser`
   query naturally handles this -- the `EXISTS` subquery returns false when
   `calendar_custom_viewers` has no rows for that calendar. This is valid
   and should not be blocked at the store layer; the handler/UI should
   warn the admin.

2. **User belongs to no IdP groups**: Both the `unit` and `custom`
   visibility paths return no results because `user_idp_groups` has no
   rows for the user. Only `association` calendars are visible. This is
   correct behavior.

3. **Calendar's owning unit has no group bindings**: For
   `visibility='unit'`, the `unit_group_bindings` join returns no rows, so
   nobody can see the calendar (except admins). This is an admin
   configuration error but not a data integrity issue.

4. **Custom viewers include the owning unit**: This is valid and expected.
   The query handles it without issues (no special case needed).

5. **Admin bypass for visibility**: Handled at the Go level by checking
   `isAdmin` and calling `ListAllCalendars` instead of the visibility
   query. This avoids complicating the SQL.

### CRUD Edge Cases

6. **Deleting a calendar with entries**: PostgreSQL CASCADE handles this.
   All entries, entry_shift_details, attendances, etc. are deleted
   transitively. The store method does not need to manually delete
   children.

7. **Updating visibility from 'custom' to 'unit'**: The `UpdateCalendar`
   wrapper always clears custom viewers on update. If the new visibility
   is not 'custom', the custom viewer list ends up empty (correct).

8. **Updating visibility from 'unit' to 'custom' without providing
   viewer units**: The calendar becomes invisible to non-admins. Same as
   edge case #1.

9. **Duplicate slug on create**: PostgreSQL UNIQUE constraint on
   `calendars.slug` returns a `pgconn.PgError` with code `23505`. The
   handler should catch this and return a validation error. The store
   layer should propagate the error without masking it.

10. **Calendar not found on GetCalendarByID/GetCalendarBySlug**: Returns
    `pgx.ErrNoRows`. The handler should check with `errors.Is(err,
    pgx.ErrNoRows)` and return a 404.

## 9. Testing Strategy

### Unit Tests (Recommended Approach)

Since the store layer is thin wrappers around sqlc-generated code, the
most valuable tests are **integration tests against a real PostgreSQL
database**. Pure unit tests with mocked DBTX would test the mock, not
the SQL.

### Integration Test Setup

Use `testcontainers-go` or a test-local PostgreSQL instance. Each test:

1. Creates a fresh database (or uses a transaction that rolls back).
2. Runs goose migrations.
3. Seeds required data (units, unit_group_bindings, users, user_idp_groups).
4. Executes the store method under test.
5. Asserts results.

### Test Cases

**`CreateCalendar`**:
- Create a calendar with default visibility ('association').
- Create a calendar with `visibility='custom'` and custom viewer units.
  Verify `calendar_custom_viewers` rows are inserted.
- Create a calendar with duplicate slug -- expect unique constraint error.

**`UpdateCalendar`**:
- Update name and color.
- Change visibility from 'custom' to 'unit' -- verify custom viewers are
  cleared.
- Change visibility from 'unit' to 'custom' with viewer units -- verify
  viewers are inserted.

**`DeleteCalendar`**:
- Delete a calendar that has entries -- verify entries are cascaded.
- Delete a calendar that has custom viewers -- verify junction rows are
  cascaded.
- Delete non-existent calendar -- expect no error (DELETE WHERE id = X
  with no matching row is not an error in PostgreSQL).

**`GetCalendarByID` (with unit info)**:
- Get existing calendar -- verify unit_name and unit_slug are populated.
- Get non-existent ID -- expect `pgx.ErrNoRows`.

**`ListCalendarsByUnit`**:
- Unit with multiple calendars -- verify sort_order, then name ordering.
- Unit with no calendars -- expect empty slice, no error.

**`ListVisibleCalendars`**:
- Admin user (`isAdmin=true`) -- sees all calendars.
- Non-admin user member of one unit -- sees association calendars + that
  unit's unit-scoped calendars.
- Non-admin user member of a unit in a custom viewer list -- sees
  association + custom-visible calendar.
- Non-admin user member of no units -- sees only association calendars.
- Calendar with `visibility='custom'` and empty viewer list -- invisible
  to all non-admin users.

**`GetCustomViewerUnits`**:
- Calendar with custom viewers -- returns list of units.
- Calendar with no custom viewers -- returns empty slice.

### Test File Location

`internal/store/calendar_test.go` (integration tests require database
access, so they should use a build tag or test flag to skip in CI without
a database).

## 10. Open Questions

1. **Is TASK-009 already satisfied by TASK-003's sqlc output?** The raw
   sqlc-generated methods cover all six required operations. This plan
   assumes the intent is to add a wrapper layer in `internal/store/` that
   composes sqlc calls (transactions for custom viewers, admin bypass for
   visibility, joined queries for unit info). Confirm this assumption.

2. **Slug generation**: The `CreateCalendar` query requires a `slug`
   parameter. Who generates it? Options:
   - (a) The handler generates it from the calendar name (e.g.,
     `slugify(name)` with collision retry).
   - (b) The store layer generates it (adds a dependency on a slug
     library like `nanoid`).
   - (c) The database generates it (requires a trigger or DEFAULT
     expression).
   The plan currently assumes (a) -- the caller provides the slug in
   `CreateCalendarParams`.

3. **Return types for `ListVisibleCalendars`**: Should the admin path
   return `[]db.Calendar` (plain calendar rows) or a joined type with
   unit info? The plan assumes plain `[]db.Calendar` for consistency with
   the non-admin path. Unit info can be resolved separately if needed.

4. **Unit membership of the owning unit**: For `visibility='unit'`,
   should the owning unit's admins (via `units.admin_group`) also see the
   calendar, even if they are not unit members via `unit_group_bindings`?
   The current query only checks member group bindings, not admin group.
   This may be intentional (unit admins are typically also members) but
   should be confirmed.

5. **Integration test infrastructure**: Does the project have a test
   database setup? The `test/seed.sql` file exists but is empty. A test
   harness (testcontainers or similar) may need to be set up as a
   prerequisite.

## 11. Implementation Steps (Execution Order)

1. Add the three new SQL queries (`GetCalendarWithUnit`,
   `ListAllCalendars`, `GetCustomViewerUnits`) to
   `internal/store/queries/calendars.sql`.

2. Run `sqlc generate` to regenerate `internal/db/calendars.sql.go`.

3. Create `internal/store/calendar.go` with the wrapper functions.

4. Write integration tests in `internal/store/calendar_test.go`.

5. Verify all acceptance criteria:
   - AC#1: CRUD methods exist and work (Create, Update, Delete, Get).
   - AC#2: `ListCalendarsByUnit` returns ordered by sort_order (already
     in sqlc query).
   - AC#3: `ListVisibleCalendars` handles all three visibility modes +
     admin bypass.
   - AC#4: Follows the existing sqlc + `*db.Queries` pattern.
<!-- SECTION:PLAN:END -->
