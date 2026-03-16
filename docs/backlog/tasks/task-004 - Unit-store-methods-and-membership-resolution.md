---
id: TASK-004
title: Unit store methods and membership resolution
status: To Do
assignee: []
created_date: '2026-03-16 14:31'
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
