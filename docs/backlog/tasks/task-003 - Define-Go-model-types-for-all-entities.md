---
id: TASK-003
title: Define Go model types for all entities
status: Done
assignee: []
created_date: '2026-03-16 14:31'
updated_date: '2026-03-21 16:13'
labels:
  - backend
milestone: m-0
dependencies:
  - TASK-002
documentation:
  - docs/backlog/documents/doc-003
priority: medium
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create Go structs in internal/model/ for all database entities defined in the migrations. The existing codebase has internal/model/user.go with the User struct as a reference pattern.

Each model struct should:
- Match the database column names and types
- Use appropriate Go types (time.Time for timestamps, []string for text arrays, *string for nullable text, etc.)
- Use int64 for IDs (matching BIGSERIAL)
- Follow the existing pattern in model/user.go

This is a mechanical translation of the schema into Go types. No business logic in model types — they are pure data containers.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 sqlc-generated model structs exist in internal/db/models.go for all 24 database tables
- [x] #2 Field types match database column types correctly (pgx/v5 types for nullable fields, native Go types for non-nullable)
- [x] #3 13 SQL query files cover all entities: users, user_idp_groups, units, unit_group_bindings, calendars, entries, attendances, events, templates, feed_tokens, external_sources, notifications, webhooks
- [x] #4 store.Store integrates with sqlc-generated db.Queries for type-safe database access
- [x] #5 Auth middleware rewired to upsert users and sync IdP groups via sqlc queries
- [x] #6 go build ./... compiles cleanly
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
## Summary

Replaced the planned manual Go model structs with sqlc-generated types, per ADR-2. sqlc reads the 9 migration files as schema and produces type-safe Go code in `internal/db/`.

### Changes

**New SQL query files** (13 files in `internal/store/queries/`):
- `users.sql` -- GetByID, GetByIDPSubject, GetByEmail, UpsertUser, UpdatePreferences
- `user_idp_groups.sql` -- Delete/Insert/Get groups, IsUserInGroup
- `units.sql` -- CRUD operations
- `unit_group_bindings.sql` -- Membership resolution (IsUserMemberOfUnit, IsUserAdminOfUnit, ListUnitsForUser), binding management
- `calendars.sql` -- CRUD, ListVisibleCalendarsForUser (visibility-aware query), custom viewer management
- `entries.sql` -- CRUD, date range queries (per-calendar and cross-calendar), recurrence rule listing, shift details, audience units, annotations
- `attendances.sql` -- CRUD, status counts, substitution requests (create/claim/list open)
- `events.sql` -- CRUD, event-calendar associations
- `templates.sql` -- Template groups, templates, recurrence rules (CRUD + update + enabled listing)
- `feed_tokens.sql` -- Create/revoke/list feed tokens
- `external_sources.sql` -- CRUD, fetch status updates, external entry upsert
- `notifications.sql` -- Configs, user preferences, notification creation and status management
- `webhooks.sql` -- CRUD for webhooks

**Generated code** (`internal/db/`): 15 Go files with model structs for all 24 tables and typed query functions.

**Store integration** (`internal/store/store.go`): Store now embeds `*db.Queries` with a `Queries()` accessor. `WithTx` signature updated to pass both `pgx.Tx` and a transaction-scoped `*db.Queries` to callbacks. Removed the manual `DBTX` interface (sqlc generates its own). `DB()` replaced by `Pool()`.

**Auth middleware** (`internal/auth/auth.go`): Expanded header mapping to read X-Forwarded-User (idp_subject), X-Forwarded-Email, X-Forwarded-Preferred-Username, and X-Forwarded-Groups. Derives display_name from username (falls back to email prefix). Determines is_assoc_admin from groups BEFORE calling UpsertUser. Syncs IdP groups on every request (delete-all + re-insert). Dev bypass uses adminGroup config value.

**Review fixes applied**: I1 (cross-calendar entry listing), I2 (entries by recurrence rule), I3 (open substitution requests), I4 (template update queries), I5 (recurrence rule update), I6 (webhook/external source updates), I7 (explicit column names for ListVisibleCalendarsForUser), I8 (renamed delete queries: DeleteUnitGroupBindings, DeleteCalendarCustomViewers, DeleteMeetingAudienceUnits), I9 (store/queries integration pattern), C1 (auth header mapping).

### Commits
1. `feat: add sqlc query files and generate Go types for all entities`
2. `feat: rewire auth and store to use sqlc-generated types`

### Verification
- `~/go/bin/sqlc generate` produces no errors
- `go build ./...` compiles cleanly
- `internal/model/` directory removed (was already empty)
<!-- SECTION:FINAL_SUMMARY:END -->
