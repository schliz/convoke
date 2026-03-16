---
id: TASK-005
title: Unit management (association admin)
status: To Do
assignee: []
created_date: '2026-03-16 14:31'
labels:
  - fullstack
milestone: m-1
dependencies:
  - TASK-004
priority: medium
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the association-admin UI for creating and managing units. Association admins (identified by the configured admin IdP group) need to be able to:

- List all units
- Create a new unit (name, slug, description, group bindings, admin group binding)
- Edit unit properties
- Delete a unit (with confirmation — cascades to calendars/entries)

This is an admin-only feature behind auth.RequireAdmin middleware. The UI should be simple forms — this is a management interface, not a public-facing page.

The existing codebase has handler patterns in internal/handler/ (error-returning handlers with Wrap()), template patterns in templates/, and the RequireAdmin middleware concept in internal/auth/. Follow these patterns.

Routes should be under /admin/units/.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Association admin can create a new unit with name, slug, and group bindings
- [ ] #2 Association admin can edit unit properties
- [ ] #3 Association admin can delete a unit with confirmation
- [ ] #4 Non-admin users cannot access unit management
- [ ] #5 Group bindings are stored in the unit_group_bindings table
<!-- AC:END -->
