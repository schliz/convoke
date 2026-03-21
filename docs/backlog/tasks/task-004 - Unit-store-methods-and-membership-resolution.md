---
id: TASK-004
title: Unit store methods and membership resolution
status: In Progress
assignee: []
created_date: '2026-03-16 14:31'
updated_date: '2026-03-21 22:24'
labels:
  - backend
milestone: m-1
dependencies:
  - TASK-003
priority: medium
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement store-layer methods for units in internal/store/. The existing codebase has internal/store/store.go with the DBTX interface pattern and internal/store/user.go as a reference.

Required store methods:
- ListUnits(ctx, db) — list all units
- GetUnitByID(ctx, db, id) — get a single unit
- GetUnitBySlug(ctx, db, slug) — get unit by URL slug
- ListUnitsByUserGroups(ctx, db, groups []string) — list units whose group bindings overlap with the user's IdP groups
- IsUnitMember(ctx, db, unitID, userGroups) — check if user's groups match any of the unit's group bindings
- IsUnitAdmin(ctx, db, unitID, userGroups, isAssocAdmin) — check if user is unit admin (via unit's admin group binding) or association admin

Unit membership is resolved by comparing the user's IdP groups (from auth.RequestUser.Groups) against the unit's group bindings stored in the unit_group_bindings table. There is no membership table — this is a join query.

Follow the DBTX pattern: store methods are package-level functions accepting DBTX as first arg (after context).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Can list all units and filter by user group membership
- [ ] #2 Can resolve unit membership from IdP groups via group bindings table
- [ ] #3 Can check unit admin status (unit admin group or association admin)
- [ ] #4 Follows existing DBTX pattern in internal/store/
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
# TASK-004 Implementation Plan: Unit Store Methods and Membership Resolution

> Date: 2026-03-21
> Status: Draft
> Dependencies: TASK-003 (sqlc-generated types and queries in `internal/db/`)

---

## 1. Context Summary

### Existing Patterns

**Store pattern (DBTX):** The codebase uses package-level functions in `internal/store/` that accept `context.Context` and `store.DBTX` as their first two arguments. The `DBTX` interface wraps `Exec`, `Query`, and `QueryRow` from pgx/v5. The `Store` struct holds a `*pgxpool.Pool` and exposes `DB()` (returns the pool as DBTX) and `WithTx()` (wraps a function in a transaction). Reference implementation: `internal/store/user.go` with `GetOrCreateUser(ctx, db, ...)`.

**sqlc layer (TASK-003):** TASK-003 introduces sqlc code generation. sqlc reads migration files as schema and query `.sql` files from `internal/store/queries/`, generating typed Go code in `internal/db/`. The generated code includes:
- `internal/db/models.go` -- structs for all 24 tables (Unit, UnitGroupBinding, etc.)
- `internal/db/db.go` -- a `DBTX` interface and `Queries` struct
- Per-query `.go` files with typed functions on the `Queries` struct

The TASK-003 plan (doc-003) already drafts the unit-related sqlc queries in `internal/store/queries/units.sql` and `internal/store/queries/unit_group_bindings.sql`. These include `GetUnitByID`, `GetUnitBySlug`, `ListUnits`, `IsUserMemberOfUnit`, `IsUserAdminOfUnit`, and `ListUnitsForUser`.

**Reconciling patterns:** The existing store uses manual `db.QueryRow(...).Scan(...)` calls with package-level functions accepting `store.DBTX`. The sqlc-generated code uses methods on a `db.Queries` struct that also accepts a `DBTX` interface (which pgxpool.Pool and pgx.Tx both satisfy). TASK-004 must decide how these two approaches coexist.

### Schema (from migration plan doc-002)

**units table:**
```sql
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
```

**unit_group_bindings table:**
```sql
CREATE TABLE unit_group_bindings (
    unit_id    BIGINT NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    group_name TEXT   NOT NULL,
    PRIMARY KEY (unit_id, group_name)
);
CREATE INDEX idx_unit_group_bindings_group_name ON unit_group_bindings (group_name);
```

**user_idp_groups table** (for membership resolution):
```sql
CREATE TABLE user_idp_groups (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_name TEXT   NOT NULL,
    PRIMARY KEY (user_id, group_name)
);
CREATE INDEX idx_user_idp_groups_group_name ON user_idp_groups (group_name);
```

### Authorization Model

- **Association admin:** Identified by `is_assoc_admin` on the `users` table (synced from IdP on login). Can manage everything.
- **Unit admin:** User whose IdP groups include the unit's `admin_group` value. Can manage the unit's calendars, entries, etc.
- **Unit member:** User who has at least one IdP group that matches a row in `unit_group_bindings` for that unit.
- **Membership resolution:** There is NO membership table. Membership is evaluated at query time by joining `user_idp_groups` with `unit_group_bindings` on `group_name`.

### Key Design Decisions (from ADR doc-001)

- ADR-3: BIGSERIAL PKs (int64 in Go), not UUID
- ADR-4: Normalized `user_idp_groups` junction table, not TEXT[] array
- ADR-2: sqlc for data access layer

### Auth Context

`auth.RequestUser` (from `internal/auth/auth.go`) carries `ID int64`, `Email string`, `IsAdmin bool`, and `Groups []string`. The `Groups` field holds the user's IdP groups, and `IsAdmin` reflects `is_assoc_admin`. After TASK-003 rewires auth, these values come from the sqlc-generated `UpsertUser` call and the `user_idp_groups` table.

---

## 2. Architectural Decision: Store Method Approach

### Decision: Option C (Hybrid)

**Rationale:** The task description specifically asks for methods that accept `userGroups []string` (the groups from the request context), not `userID int64`. The sqlc-generated queries (`IsUserMemberOfUnit`, `ListUnitsForUser`) join through `user_idp_groups` by `user_id`. This works for database-persisted users but the TASK-004 API needs to accept groups directly -- this is the right API because:

1. It avoids a database round-trip to look up the user's groups (they are already in the request context from the auth middleware).
2. It works even before the user's groups have been synced to the database on this request.
3. It matches what the task description explicitly requires.

Therefore:
- **Simple CRUD** (`ListUnits`, `GetUnitByID`, `GetUnitBySlug`): Thin wrappers around sqlc-generated queries.
- **Membership methods** (`ListUnitsByUserGroups`, `IsUnitMember`, `IsUnitAdmin`): Package-level functions with hand-written SQL that use `ANY($1::text[])` to match groups directly, without joining through `user_idp_groups`.

---

## 3. File Changes

### New Files

| File | Purpose |
|------|---------|
| `internal/store/unit.go` | All 6 store methods for units |
| `internal/store/unit_test.go` | Unit tests using pgxmock |

### Modified Files

| File | Change |
|------|--------|
| `go.mod` / `go.sum` | Add `github.com/pashagolub/pgxmock/v5` as test dependency (if not already present) |

---

## 4. Method Signatures

All methods are package-level functions in `package store`, following the existing `GetOrCreateUser` pattern.

```go
func ListUnits(ctx context.Context, dbtx DBTX) ([]db.Unit, error)
func GetUnitByID(ctx context.Context, dbtx DBTX, id int64) (*db.Unit, error)
func GetUnitBySlug(ctx context.Context, dbtx DBTX, slug string) (*db.Unit, error)
func ListUnitsByUserGroups(ctx context.Context, dbtx DBTX, groups []string) ([]db.Unit, error)
func IsUnitMember(ctx context.Context, dbtx DBTX, unitID int64, userGroups []string) (bool, error)
func IsUnitAdmin(ctx context.Context, dbtx DBTX, unitID int64, userGroups []string, isAssocAdmin bool) (bool, error)
```

## 5. SQL Queries

### 5.1 ListUnits

```sql
SELECT id, name, slug, description, logo_path, contact_email, admin_group, created_at, updated_at
FROM units
ORDER BY name
```

If a sqlc query `ListUnits` already exists from TASK-003, use the generated function. Otherwise, hand-write this as a direct `db.Query` call.

### 5.2 GetUnitByID

```sql
SELECT id, name, slug, description, logo_path, contact_email, admin_group, created_at, updated_at
FROM units
WHERE id = $1
```

Single-row query. Use `db.QueryRow(...).Scan(...)` or the sqlc-generated function.

### 5.3 GetUnitBySlug

```sql
SELECT id, name, slug, description, logo_path, contact_email, admin_group, created_at, updated_at
FROM units
WHERE slug = $1
```

Single-row query. Same pattern as GetUnitByID.

### 5.4 ListUnitsByUserGroups

This is the core membership resolution query. It joins `units` with `unit_group_bindings` and filters by the user's groups passed as a `text[]` parameter.

```sql
SELECT DISTINCT u.id, u.name, u.slug, u.description, u.logo_path, u.contact_email, u.admin_group, u.created_at, u.updated_at
FROM units u
JOIN unit_group_bindings ugb ON u.id = ugb.unit_id
WHERE ugb.group_name = ANY($1::text[])
ORDER BY u.name
```

**Key detail:** pgx natively supports passing `[]string` as a PostgreSQL `text[]` parameter. The `ANY($1::text[])` operator checks if `group_name` is contained in the provided array. The `DISTINCT` is necessary because a user might be in multiple groups that bind to the same unit.

**Empty groups handling:** If `$1` is an empty array `'{}'`, `ANY` will match nothing, returning zero rows. This is correct behavior. The Go code should still short-circuit and return `nil, nil` before executing the query when `groups` is nil or empty, to avoid the round-trip.

### 5.5 IsUnitMember

```sql
SELECT EXISTS(
    SELECT 1
    FROM unit_group_bindings
    WHERE unit_id = $1
      AND group_name = ANY($2::text[])
) AS is_member
```

Returns a single boolean. Uses `QueryRow(...).Scan(&result)`.

### 5.6 IsUnitAdmin

This method has two paths:

1. If `isAssocAdmin` is true, return `true` immediately (no database query needed).
2. Otherwise, check if any of `userGroups` matches the unit's `admin_group`.

```sql
SELECT EXISTS(
    SELECT 1
    FROM units
    WHERE id = $1
      AND admin_group IS NOT NULL
      AND admin_group = ANY($2::text[])
) AS is_admin
```

**Key detail:** If the unit has no `admin_group` (NULL), this returns false. This is correct per the requirements.

**Go implementation:** The `isAssocAdmin` check is a pure Go short-circuit, not a SQL parameter.

```go
func IsUnitAdmin(ctx context.Context, dbtx DBTX, unitID int64, userGroups []string, isAssocAdmin bool) (bool, error) {
    if isAssocAdmin {
        return true, nil
    }
    if len(userGroups) == 0 {
        return false, nil
    }
    var isAdmin bool
    err := dbtx.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM units
            WHERE id = $1
              AND admin_group IS NOT NULL
              AND admin_group = ANY($2::text[])
        )
    `, unitID, userGroups).Scan(&isAdmin)
    return isAdmin, err
}
```

## 6. Implementation Details

### 6.1 Row Scanning Strategy

Two approaches depending on TASK-003 state:

**If sqlc-generated queries exist for CRUD operations:** Use `db.New(dbtx).ListUnits(ctx)`, `db.New(dbtx).GetUnitByID(ctx, id)`, etc. These return `[]db.Unit` and `db.Unit` with scanning already handled.

**If writing hand-crafted SQL (for membership methods or if sqlc queries are not yet available):** Use manual `Scan` for hand-written queries (membership methods), matching the pattern in `user.go`.

### 6.2 Scanning a Unit Row (Helper)

To avoid repeating the scan call list, define a private helper:

```go
func scanUnit(row pgx.Row) (*db.Unit, error) {
    var u db.Unit
    err := row.Scan(
        &u.ID, &u.Name, &u.Slug, &u.Description,
        &u.LogoPath, &u.ContactEmail, &u.AdminGroup,
        &u.CreatedAt, &u.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return &u, nil
}
```

And a multi-row variant:

```go
const unitColumns = `id, name, slug, description, logo_path, contact_email, admin_group, created_at, updated_at`

func scanUnits(rows pgx.Rows) ([]db.Unit, error) {
    defer rows.Close()
    var units []db.Unit
    for rows.Next() {
        var u db.Unit
        if err := rows.Scan(
            &u.ID, &u.Name, &u.Slug, &u.Description,
            &u.LogoPath, &u.ContactEmail, &u.AdminGroup,
            &u.CreatedAt, &u.UpdatedAt,
        ); err != nil {
            return nil, err
        }
        units = append(units, u)
    }
    return units, rows.Err()
}
```

### 6.3 pgx.Row vs pgx.Rows

- `store.DBTX.QueryRow` returns `pgx.Row` (single row, error deferred to Scan).
- `store.DBTX.Query` returns `pgx.Rows` (iterator, must be closed).

### 6.4 Adaptation to sqlc-generated types

If the actual `db.Unit` struct uses `pgtype.Text` for nullable columns rather than `*string`, the Scan calls use `&u.Description` etc. and pgx handles the pgtype scanning natively. No conversion needed.

---

## 7. Edge Cases

### 7.1 Empty Groups Slice

All membership methods must handle `groups == nil` or `groups == []string{}`:
- `ListUnitsByUserGroups`: Return empty slice (`nil, nil`) immediately.
- `IsUnitMember`: Return `false, nil` immediately.
- `IsUnitAdmin`: Only the `isAssocAdmin` bypass applies; if false, return `false, nil`.

### 7.2 Unit Not Found

- `GetUnitByID` and `GetUnitBySlug`: Return `nil, pgx.ErrNoRows` when no matching row exists.
- `IsUnitMember` and `IsUnitAdmin` with a non-existent `unitID`: The `EXISTS` subquery returns `false`.

### 7.3 Unit With No Group Bindings

- `IsUnitMember`: Returns `false` because no rows exist in `unit_group_bindings` for that unit.
- `ListUnitsByUserGroups`: The unit is excluded because the JOIN produces no rows.

### 7.4 Unit With No Admin Group

- `IsUnitAdmin` with `admin_group IS NULL`: Returns `false` due to the `admin_group IS NOT NULL` condition. Only association admins can manage such a unit.

### 7.5 User in Multiple Groups Matching Same Unit

- `ListUnitsByUserGroups`: The `DISTINCT` keyword prevents duplicate unit rows.
- `IsUnitMember`: `EXISTS` returns `true` on the first match.

### 7.6 Case Sensitivity of Group Names

Group names are compared case-sensitively (standard PostgreSQL text comparison). This is intentional -- IdP group names are case-sensitive identifiers.

## 8. Testing Strategy

### 8.1 Approach: pgxmock

Use `pgxmock` to mock the `DBTX` interface. pgxmock implements the pgx pool/connection interface.

### 8.2 Test File: `internal/store/unit_test.go`

### 8.3 Test Cases

#### ListUnits

| Test | Setup | Expected |
|------|-------|----------|
| Returns all units ordered by name | Mock returns 3 unit rows | Slice of 3 `db.Unit`, correct field mapping |
| Returns empty slice when no units | Mock returns 0 rows | Empty (nil) slice, no error |
| Propagates database error | Mock returns error | nil slice, error returned |

#### GetUnitByID

| Test | Setup | Expected |
|------|-------|----------|
| Returns unit when found | Mock returns 1 row | `*db.Unit` with correct fields |
| Returns ErrNoRows when not found | Mock returns pgx.ErrNoRows | nil, pgx.ErrNoRows |
| Propagates database error | Mock returns error | nil, error |

#### GetUnitBySlug

| Test | Setup | Expected |
|------|-------|----------|
| Returns unit when found | Mock returns 1 row for slug "bar-committee" | `*db.Unit` with correct slug |
| Returns ErrNoRows when not found | Mock returns pgx.ErrNoRows | nil, pgx.ErrNoRows |

#### ListUnitsByUserGroups

| Test | Setup | Expected |
|------|-------|----------|
| Returns matching units | Mock expects `ANY($1)` with groups, returns 2 rows | Slice of 2 units |
| Returns empty for non-matching groups | Mock returns 0 rows | Empty slice |
| Short-circuits for nil groups | No mock expectations | nil, nil (no DB call) |
| Short-circuits for empty groups | No mock expectations | nil, nil (no DB call) |

#### IsUnitMember

| Test | Setup | Expected |
|------|-------|----------|
| Returns true when group matches | Mock returns `true` from EXISTS | true, nil |
| Returns false when no match | Mock returns `false` from EXISTS | false, nil |
| Short-circuits for nil groups | No mock expectations | false, nil |
| Short-circuits for empty groups | No mock expectations | false, nil |

#### IsUnitAdmin

| Test | Setup | Expected |
|------|-------|----------|
| Returns true when isAssocAdmin | No mock expectations | true, nil (short-circuit) |
| Returns true when group matches admin_group | Mock returns `true` from EXISTS | true, nil |
| Returns false when no match | Mock returns `false` from EXISTS | false, nil |
| Returns false when admin_group is NULL | Mock returns `false` (IS NOT NULL filters it) | false, nil |
| Short-circuits for empty groups, not assoc admin | No mock expectations | false, nil |

---

## 9. Dependency on TASK-003

TASK-004 depends on TASK-003 producing:

1. **`internal/db/models.go`** with the `db.Unit` struct (used as return type).
2. **Optionally, sqlc-generated query functions** in `internal/db/` for `ListUnits`, `GetUnitByID`, `GetUnitBySlug`.
3. **The `internal/db/db.go`** file with the sqlc `DBTX` interface and `New()` constructor.

The `store.DBTX` interface and `db.DBTX` interface (sqlc-generated) are structurally identical. No adapter is needed.

---

## 10. Implementation Order

1. **Create `internal/store/unit.go`** with all 6 methods.
2. **Create `internal/store/unit_test.go`** with all test cases.
3. **Run `go test ./internal/store/...`** to verify.
4. **Run `go build ./...`** to verify no compilation errors.

Within `unit.go`, implement in this order:
1. `scanUnit` and `scanUnits` helpers (or column constant)
2. `ListUnits` (simplest, validates pattern)
3. `GetUnitByID`
4. `GetUnitBySlug`
5. `ListUnitsByUserGroups` (first membership query)
6. `IsUnitMember`
7. `IsUnitAdmin`

---

## 11. Open Questions

### Q1: Should `ListUnitsByUserGroups` also return units where the user is an admin (via `admin_group`) but not a member?

**Tentative answer: No.** The task description says "list units whose group bindings overlap with the user's IdP groups." The `admin_group` is separate from `unit_group_bindings`. If a unit admin should also see the unit in navigation, they should be added to a member group binding as well. The fix, if needed, is simple: add `OR u.admin_group = ANY($1::text[])` to the WHERE clause.

### Q2: Type of return value -- pointer vs value for single-entity methods?

**Tentative answer:** `*db.Unit` (pointer) for `GetUnitByID` and `GetUnitBySlug`, matching the existing `GetOrCreateUser` pattern. `[]db.Unit` (value slice) for list methods.

### Q3: Should `ListUnitsByUserGroups` also return the matched group names?

**Tentative answer: No.** The task description asks for a list of units, not a list of (unit, matched_group) pairs.

### Q4: Should the store methods handle `pgx.ErrNoRows` by wrapping it in a domain error?

**Tentative answer: No.** The existing `GetOrCreateUser` does not wrap errors. The handler layer should check `errors.Is(err, pgx.ErrNoRows)` and return `&NotFoundError{...}`.

### Q5: pgxmock version compatibility

The project uses `pgx/v5`. Ensure the test dependency is `pgxmock/v4` or `/v5` as appropriate. Verify with `go get`.
<!-- SECTION:PLAN:END -->
