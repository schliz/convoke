---
id: TASK-022
title: Personal dashboard
status: To Do
assignee: []
created_date: '2026-03-16 14:34'
labels:
  - fullstack
milestone: m-4
dependencies:
  - TASK-020
  - TASK-021
documentation:
  - docs/design/design/personal_dashboard/
priority: medium
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the personal dashboard — the default landing page for authenticated users. This view shows entries relevant to the current user across all their units and calendars.

Display sections:
- Upcoming accepted shifts (next 14 days, sorted by start time)
- Meetings requiring response (pending status, sorted by response deadline then start time)
- Recently responded meetings (last 7 days of accepted/declined, for reference)

Each entry should use the existing shift-card or meeting-card components, showing the calendar name and unit for context since entries come from multiple sources.

The view should clearly separate "action needed" (pending meetings, understaffed shifts the user could join) from "confirmed" (accepted entries).

Route: GET /dashboard (or GET / after login)

This page uses ListEntriesWithUserAttendance from the attendance store to efficiently fetch entries with the user's status in a single query.

Reference: docs/design/design/personal_dashboard/ contains a draft mockup.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Dashboard is the default landing page after login
- [ ] #2 Shows upcoming accepted shifts sorted by start time
- [ ] #3 Shows meetings requiring response (pending status)
- [ ] #4 Uses existing shift-card and meeting-card components
- [ ] #5 Entries show calendar and unit context
- [ ] #6 Action-needed items are visually distinguished from confirmed items
<!-- AC:END -->
