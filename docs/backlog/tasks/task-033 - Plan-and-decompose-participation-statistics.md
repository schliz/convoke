---
id: TASK-033
title: Plan and decompose participation statistics
status: To Do
assignee: []
created_date: '2026-03-16 14:36'
labels:
  - design
  - planning
milestone: m-10
dependencies:
  - TASK-024
documentation:
  - docs/design/requirements.md
priority: low
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Design the participation statistics feature and create implementation subtasks. Review requirements §7.

Statistics provide a table view of participation history per unit:
- Rows: unit members
- Columns: calendars within the unit
- Cells: count of attended entries in the selected time range
- Totals: per member (row sum) and per calendar (column sum)
- Time ranges: month, quarter, half-year, year (selectable)

Design decisions needed:
- Query strategy: real-time aggregation vs materialized/cached stats
- How to resolve "unit members" for the row list (IdP groups → users query)
- Access control: visible to unit admins by default, configurable for regular members
- Default time range configurable per unit

Output: subtasks for statistics queries, statistics page UI, access control, and e2e tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Design document covering statistics queries, UI, and access control
- [ ] #2 Subtasks created for: statistics queries, statistics page, access control, e2e tests
- [ ] #3 Query performance approach documented (real-time vs cached)
<!-- AC:END -->
