---
id: TASK-011
title: Calendar month view
status: To Do
assignee: []
created_date: '2026-03-16 14:32'
labels:
  - fullstack
milestone: m-2
dependencies:
  - TASK-009
  - TASK-006
documentation:
  - docs/design/design/month_view_calendar/
priority: medium
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the calendar month view — the default view for browsing a calendar's entries. This is one of the most important user-facing pages.

The month view displays a grid of days for a calendar month. Each day cell shows the entries for that day with compact indicators:
- Shift entries: name + compact staffing indicator (e.g., "2/4") color-coded by staffing status (understaffed=error, minimum-met=warning, full=success)
- Meeting entries: name + the viewing user's response status icon (accepted/declined/pending)
- Entries are color-coded by calendar color

Navigation:
- Previous/next month buttons (HTMX partial updates, hx-push-url for URL state)
- Today button to jump to current month
- Clicking a day navigates to the day view

Route: GET /units/{slug}/calendars/{id}/month?date=2026-03

The view needs:
- Entry store methods for listing entries by calendar and date range (month boundaries)
- Attendance store methods for getting the current user's status on each entry (for meeting indicators)
- A page template with a responsive month grid
- View model struct with pre-computed display data for each day/entry

The month grid should be responsive — on mobile, consider a list layout instead of a 7-column grid.

Reference: docs/design/design/month_view_calendar/ contains a draft mockup (AI-generated, treat as inspiration only).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Month grid displays entries for each day of the month
- [ ] #2 Shift entries show compact staffing indicator with color coding
- [ ] #3 Meeting entries show user's response status
- [ ] #4 Previous/next month navigation works with HTMX partial updates
- [ ] #5 URL updates via hx-push-url when navigating months
- [ ] #6 Clicking a day navigates to the day view
- [ ] #7 Responsive layout works on mobile
<!-- AC:END -->
