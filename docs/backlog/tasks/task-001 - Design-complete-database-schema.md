---
id: TASK-001
title: Design complete database schema
status: Done
assignee: []
created_date: '2026-03-16 14:31'
updated_date: '2026-03-21 16:03'
labels:
  - design
milestone: m-0
dependencies: []
documentation:
  - docs/design/requirements.md
  - docs/superpowers/specs/2026-03-21-database-schema-design.md
  - docs/backlog/documents/doc-001
priority: medium
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Review the full requirements document (docs/design/requirements.md) and design the complete database schema for Convoke. This is a design task — the goal is to produce a well-reasoned schema document, not to write migrations directly.

The existing codebase has one migration (00001_create_users.sql) creating a `users` table. The schema design must cover all remaining entities from the requirements:

- Units and unit group bindings (IdP group → unit membership)
- Calendars (with access control fields: entry_creation, visibility, participation, participant_visibility)
- Events (calendars with bounded date ranges)
- Entries (shifts and meetings, with type-specific fields)
- Attendance (accepted/declined/pending/needs_substitute/replaced statuses)
- Template groups and templates (reusable entry blueprints)
- Recurrence rules (pattern types, holiday/weekend handling, auto-create horizon)
- External sources (imported iCal feeds)
- Feed tokens (per-user, per-scope authentication for iCal export)
- Notification preferences (per-user, per-channel opt-in/out)

Key design considerations:
- Unit membership is resolved via IdP groups, not a membership table. Units store group bindings.
- Calendar access control has four dimensions (see requirements §3.3).
- Entries have a CHECK constraint: entry_type IN ('shift', 'meeting').
- Attendance has a UNIQUE(entry_id, user_id) constraint.
- Recurrence rules need pattern type storage (nth weekday, nth day, every nth weekday, etc.).
- Template instantiation must be idempotent — duplicate detection via calendar + name + start_at.
- Consider what indexes are needed for the primary query patterns (entries by calendar+date, attendance by entry, attendance by user+status, units by slug, users by email).

Output: A schema design document (can be markdown with SQL DDL blocks) documenting each table, its columns, types, constraints, indexes, and the reasoning behind key decisions. Save to docs/superpowers/specs/.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 All entities from requirements sections 3-7 are represented in the schema design
- [x] #2 Relationships between entities are documented with foreign keys and ON DELETE behavior
- [x] #3 Index strategy is documented with justification for each index
- [x] #4 Check constraints and enum-like columns are specified
- [x] #5 Design decisions and trade-offs are explained (not just raw DDL)
- [x] #6 The existing users table (migration 00001) is accounted for — schema builds on it
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Schema design complete. 24-table PostgreSQL schema covering all entities from requirements sections 3-7. Key decisions documented in DOC-001 (ADR): BIGSERIAL PKs + opaque slugs, CHECK constraints over native enums, events as loose calendar groupings, sqlc for data access, hard delete with CASCADE. Spec reviewed and approved. Implementation plans created as DOC-002 (migrations) and DOC-003 (sqlc model types).
<!-- SECTION:FINAL_SUMMARY:END -->
