---
id: TASK-002
title: Implement database migrations
status: To Do
assignee: []
created_date: '2026-03-16 14:31'
updated_date: '2026-03-16 14:36'
labels:
  - backend
milestone: m-0
dependencies:
  - TASK-001
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
- [ ] #1 All tables from the approved schema design are created via Goose migrations
- [ ] #2 Migrations run cleanly: goose up from scratch and goose down back to zero
- [ ] #3 Foreign key relationships match the design with correct ON DELETE behavior
- [ ] #4 Indexes and constraints match the design
- [ ] #5 Migrations are embedded and auto-apply at startup (existing pattern in cmd/server/main.go)
<!-- AC:END -->
