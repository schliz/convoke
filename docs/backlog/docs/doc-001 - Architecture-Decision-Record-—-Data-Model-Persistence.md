---
id: doc-001
title: Architecture Decision Record — Data Model & Persistence
type: other
created_date: '2026-03-21 15:10'
---
# Architecture Decision Record — Data Model & Persistence

> Context: TASK-001 (Design complete database schema)
> Date: 2026-03-21
> Status: Decided

---

## ADR-1: PostgreSQL as sole source of truth

**Decision:** PostgreSQL stores all application state. Redis is used only as a task queue broker (recurrence evaluation, notification delivery). No application state lives in Redis.

**Rationale:** The query patterns (entries by calendar+date, attendance counts, membership resolution) are well-served by indexes at single-association scale. Keeping state in one place simplifies operations and means losing Redis doesn't lose data.

---

## ADR-2: sqlc for data access

**Decision:** Use sqlc (SQL-to-Go code generation) for the data access layer. Hand-write SQL queries in `.sql` files, sqlc generates type-safe Go functions with row scanning.

**Rationale:** Keeps us writing real SQL (no ORM abstraction), eliminates boilerplate row scanning across 20+ tables, and catches query/schema mismatches at build time. Pairs naturally with goose migrations since both work with raw SQL.

---

## ADR-3: BIGSERIAL primary keys + opaque slugs for URLs

**Decision:** Use `BIGSERIAL` for all internal primary keys and foreign keys. Add a short opaque identifier column (nanoid or similar) on entities that appear in URLs.

**Entities with opaque URL identifier:** units (already have `slug`), calendars, entries, events.

**Entities without (internal-only integer PK):** attendances, substitution_requests, templates, recurrence_rules, notification_configs, webhooks, user_idp_groups, unit_group_bindings.

**Rationale:** Compact integer PKs give fast joins and sequential index inserts. Opaque slugs prevent enumeration in URLs. UUIDs are overkill for a single-database system. The split avoids adding slug overhead to tables that are only accessed through their parent entity or admin-only settings.

---

## ADR-4: Normalized IdP group storage

**Decision:** Store IdP group memberships in a `user_idp_groups` junction table (user_id, group_name), not as a `TEXT[]` array on the users table. Fully replaced on each login sync.

**Rationale:** Better queryability for membership resolution ("which users are in group X?"), cleaner joins with `unit_group_bindings` for determining unit membership. The existing migration (00001) used TEXT[] and will be discarded.

---

## ADR-5: CHECK constraints for enums, not CREATE TYPE

**Decision:** Use `TEXT` columns with `CHECK (column IN (...))` constraints instead of native PostgreSQL `CREATE TYPE ... AS ENUM`.

**Rationale:** Easier migration path as the app evolves — adding/removing values is a simple `ALTER TABLE` with no transaction restrictions. Same data integrity guarantees. sqlc can still generate typed Go constants. (PostgreSQL 18 improves native enum ergonomics but CHECK constraints remain simpler.)

---

## ADR-6: Hard delete with CASCADE + stats snapshots

**Decision:** Use hard deletes with `ON DELETE CASCADE` for all entities. Preserve attendance history for participation statistics via periodic stats snapshots, not soft deletes.

**Rationale:** Keeps the operational data model clean — no `deleted_at` filtering on every query. Statistics (§7 of requirements) are served from pre-aggregated snapshot data rather than querying raw attendance records, so deleting an entry doesn't destroy historical stats.

---

## ADR-7: Events as loose calendar groupings (not owners)

**Decision:** Model events as a separate entity with a many-to-many junction to calendars. An event does NOT own calendars — it groups them.

```
events (id, name, slug, unit_id, start_date, end_date, website, description)
event_calendars (event_id, calendar_id, sort_order)
```

**Rejected alternative:** Event as an owning container (calendars.event_id FK). Rejected because it creates ambiguous ownership (unit vs event), prevents calendar reuse across events, and complicates deletion semantics.

**Rationale:** A summer festival has a bar calendar (open signup) and a tech crew calendar (restricted). These are normal calendars owned by their respective units, grouped under the event. The event timeline view is a read-only aggregation. Cross-unit events work naturally — each unit creates their own calendar, the event coordinator links them. "Create calendar for event" is handled as an atomic UI action, not a schema constraint.

---

## ADR-8: Guest participation deferred

**Decision:** Guest participation (external users joining events via email link) is not included in the initial schema. The design has been validated as addable later.

**Migration path when ready:** Add a `guests` table, add nullable `guest_id` FK to attendances, make `user_id` nullable, add `CHECK` ensuring exactly one of user_id/guest_id is set. No existing data needs migration.

---

## ADR-9: Existing migration to be discarded

**Decision:** The existing `00001_create_users.sql` (BIGSERIAL PK, TEXT[] groups, no IdP subject) will be replaced with migrations matching the new schema design.

**Rationale:** The initial migration was scaffolding. The new schema differs in PK type, group storage approach, and column set (adds idp_subject, username, timezone, locale).
