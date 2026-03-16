---
id: TASK-009
title: Calendar store methods
status: To Do
assignee: []
created_date: '2026-03-16 14:32'
labels:
  - backend
milestone: m-2
dependencies:
  - TASK-003
priority: medium
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement store-layer methods for calendars in internal/store/. Follow the existing DBTX pattern from store.go and user.go.

Required store methods:
- CreateCalendar(ctx, db, calendar) — insert a new calendar
- UpdateCalendar(ctx, db, calendar) — update calendar properties
- DeleteCalendar(ctx, db, id) — delete calendar (entries cascade via FK)
- GetCalendarByID(ctx, db, id) — get a single calendar with its unit info
- ListCalendarsByUnit(ctx, db, unitID) — list calendars for a unit, ordered by sort_order
- ListVisibleCalendars(ctx, db, userGroups, isAdmin) — list calendars visible to a user based on visibility settings (association-wide, unit membership, or custom unit list)

The visibility check is the most nuanced method: it must evaluate the calendar's visibility setting against the user's group memberships. For visibility='association', all authenticated users see it. For visibility='unit', only unit members. For visibility='custom', check against the explicit unit list.

All methods accept DBTX as a parameter (not a receiver method on Store).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 CRUD methods for calendars exist and work correctly
- [ ] #2 ListCalendarsByUnit returns calendars ordered by sort_order
- [ ] #3 ListVisibleCalendars correctly evaluates all three visibility modes (association, unit, custom)
- [ ] #4 Follows existing DBTX pattern in internal/store/
<!-- AC:END -->
