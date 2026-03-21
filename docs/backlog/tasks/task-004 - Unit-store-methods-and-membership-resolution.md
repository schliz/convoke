---
id: TASK-004
title: Unit store methods and membership resolution
status: In Progress
assignee: []
created_date: '2026-03-16 14:31'
updated_date: '2026-03-21 22:23'
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
<!-- SECTION:PLAN:END -->
