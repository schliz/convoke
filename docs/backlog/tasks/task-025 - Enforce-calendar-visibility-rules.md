---
id: TASK-025
title: Enforce calendar visibility rules
status: To Do
assignee: []
created_date: '2026-03-16 14:35'
labels:
  - backend
milestone: m-5
dependencies:
  - TASK-009
  - TASK-014
priority: medium
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement visibility enforcement across all views and queries. Calendar visibility (§3.3) controls who can see a calendar and its entries:
- 'association': all authenticated users
- 'unit': only members of the owning unit
- 'custom': an explicit list of units whose members can see it

This must be enforced at the store/handler level, not just the template level. Every query that returns calendars or entries must filter by the user's visibility access.

Areas to enforce:
- Calendar listing (unit dashboard, navigation) — already partially handled by ListVisibleCalendars
- Entry listing (month view, day view, personal dashboard) — entries from invisible calendars must be excluded
- Entry detail view — 404 if user cannot see the calendar
- Direct URL access — users cannot access a calendar or entry by guessing the URL

Implementation approach: add a visibility check helper to the handler that takes a calendar and user, returns bool. Use it consistently in all handlers that display calendar/entry data. The store methods for listing entries should accept the user's accessible calendar IDs as a filter.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Users only see calendars matching their visibility access in all views
- [ ] #2 Entries from invisible calendars are excluded from all listings
- [ ] #3 Direct URL access to invisible calendar/entry returns 404
- [ ] #4 Association-wide calendars visible to all authenticated users
- [ ] #5 Unit-scoped calendars visible only to unit members
- [ ] #6 Custom visibility checks against explicit unit list
<!-- AC:END -->
