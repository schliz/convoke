---
id: TASK-013
title: 'E2E tests: Calendar management and views'
status: To Do
assignee: []
created_date: '2026-03-16 14:33'
labels:
  - testing
milestone: m-2
dependencies:
  - TASK-010
  - TASK-011
  - TASK-012
priority: low
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Write Playwright e2e tests covering calendar admin CRUD and calendar views (month and day).

Seed data needs: units with calendars containing entries (both shifts and meetings) spread across dates for meaningful month/day view testing.

Test scenarios:
- Admin can create a calendar with access control settings
- Admin can edit calendar properties
- Admin can delete a calendar
- Month view displays entries on correct days
- Month view navigation (prev/next month) works
- Day view shows entries for selected day
- Day view navigation (prev/next day) works
- Non-admin cannot access calendar management

Follow existing Playwright project structure.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Seed data includes calendars with entries for view testing
- [ ] #2 Tests cover calendar CRUD by admin
- [ ] #3 Tests cover month view rendering and navigation
- [ ] #4 Tests cover day view rendering and navigation
- [ ] #5 Tests verify non-admin cannot manage calendars
<!-- AC:END -->
