---
id: TASK-012
title: Calendar day view
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
  - docs/design/design/entry_detail_shift/
priority: medium
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the calendar day view — a detailed view showing all entries for a single day within a calendar.

The day view shows entries in chronological order with expanded information:
- Entry name, type (shift/meeting), time range, location
- Staffing status bar for shifts (filled/required/max with progress indicator)
- Response summary for meetings (accepted/declined/pending counts)
- Quick-action buttons (join shift, RSVP to meeting) — these will be wired up in the Attendance milestone, but the UI slots should exist
- Link to full entry detail view

Navigation:
- Previous/next day buttons (HTMX)
- Back to month view link
- Date in the header

Route: GET /units/{slug}/calendars/{id}/day?date=2026-03-16

View model should compose entry card components (shift-card, meeting-card) that will be reused in other views (unit dashboard, personal dashboard).

Reference: docs/design/design/entry_detail_shift/ contains a draft mockup for entry cards.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Day view shows all entries for the selected day in chronological order
- [ ] #2 Shift entries display staffing status with progress indicator
- [ ] #3 Meeting entries display response summary counts
- [ ] #4 Previous/next day navigation with HTMX
- [ ] #5 Each entry links to its full detail view
- [ ] #6 Entry card components are reusable (shift-card, meeting-card)
<!-- AC:END -->
