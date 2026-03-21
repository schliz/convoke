---
id: TASK-002
title: Implement database migrations
status: Done
assignee: []
created_date: '2026-03-16 14:31'
updated_date: '2026-03-21 16:07'
labels:
  - backend
milestone: m-0
dependencies:
  - TASK-001
documentation:
  - docs/backlog/documents/doc-002
priority: medium
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Take the approved schema design and implement it as Goose SQL migration files in the migrations/ directory. The project uses embedded Goose migrations (see migrations/embed.go) that auto-apply at startup.

The existing migration 00001_create_users.sql creates the users table. New migrations should be numbered sequentially (00002, 00003, etc.) and may be split into logical groups if that aids clarity and reviewability.

Each migration must have proper +goose Up and +goose Down directives. Down migrations must cleanly reverse the up migration (DROP TABLE IF EXISTS in reverse dependency order).

Follow the schema design document produced by the design task.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 All tables from the approved schema design are created via Goose migrations
- [x] #2 Migrations run cleanly: goose up from scratch and goose down back to zero
- [x] #3 Foreign key relationships match the design with correct ON DELETE behavior
- [x] #4 Indexes and constraints match the design
- [x] #5 Migrations are embedded and auto-apply at startup (existing pattern in cmd/server/main.go)
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
9 goose migration files created covering all 24 tables from the approved schema spec. Migrations are grouped by domain area and ordered to respect FK dependencies. sqlc.yaml configured to read migrations as schema. Makefile updated with `sqlc` target. Auth middleware stubbed to compile without old store functions. All migrations committed individually.
<!-- SECTION:FINAL_SUMMARY:END -->
