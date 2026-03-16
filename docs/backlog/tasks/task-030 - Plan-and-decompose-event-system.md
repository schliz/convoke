---
id: TASK-030
title: Plan and decompose event system
status: To Do
assignee: []
created_date: '2026-03-16 14:36'
labels:
  - design
  - planning
milestone: m-7
dependencies:
  - TASK-024
documentation:
  - docs/design/requirements.md
  - docs/design/design/event_timeline_view_summer_festival/
priority: low
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Design the event system in detail and create implementation subtasks. This is a planning task — review requirements §3.4 and §9.2 before writing code.

Events are calendars with bounded date ranges (start date, end date) and a specialized timeline UI. Key areas to design:

- Event as a calendar variant: how to model events (separate table? boolean flag on calendar? additional fields on calendar?). The schema design task (TASK-001) should have established the data model — this task designs the feature layer.
- Event CRUD: creating/editing events within a unit (admin), setting date range, website, description
- Event timeline view (§9.2): horizontal time axis spanning the event period, entries grouped by day, staffing status prominently displayed, aggregate status bar (overall fill rate, understaffed entries count), quick-join from timeline
- Cross-unit event coordination: how multiple units contribute entries to a shared event
- Event-specific metrics: total entries, total slots, filled slots, entries below minimum staffing

The timeline view is the most complex UI in the app — it needs careful design for usability on both desktop and mobile.

Output: detailed subtasks covering event CRUD, timeline view, aggregate metrics, cross-unit coordination, and e2e tests.

Reference: docs/design/design/event_timeline_view_summer_festival/ contains a draft mockup.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Design document covering event data model, CRUD, timeline view, and cross-unit coordination
- [ ] #2 Subtasks created for: event CRUD, timeline view, aggregate metrics, e2e tests
- [ ] #3 Timeline view design addresses both desktop and mobile layouts
- [ ] #4 Each subtask has clear acceptance criteria
<!-- AC:END -->
