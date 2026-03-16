---
id: TASK-016
title: Entry detail view
status: To Do
assignee: []
created_date: '2026-03-16 14:33'
labels:
  - fullstack
milestone: m-3
dependencies:
  - TASK-014
  - TASK-006
documentation:
  - docs/design/design/entry_detail_shift/
priority: medium
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the entry detail view — the full-information page for a single entry. This is where users see complete entry details, participant lists, and take actions (join/leave/RSVP — wired up in the Attendance milestone).

Display:
- Entry name, type badge (shift/meeting), date/time range, location, description
- Calendar name and unit it belongs to (with links)
- For shifts: staffing status bar (filled/required/max), color-coded status
- For meetings: response summary (accepted/declined/pending counts out of total audience)
- Participant list — subject to participant_visibility rules (everyone sees names, unit-only sees names, participants-only sees names; others see aggregate counts). For now, implement the "everyone" mode; access control enforcement comes later.
- Edit/delete buttons (visible to authorized users)
- Warning annotations (placeholder for recurrence rule holiday/weekend warnings)

Route: GET /entries/{id}

View model should use the ShiftCard/MeetingCard patterns from the go-htmx-fullstack skill with precomputed boolean flags (CanJoin, CanLeave, CanEdit, etc.). These flags will be false initially and wired up when attendance features are built.

Reference: docs/design/design/entry_detail_shift/ contains a draft mockup.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Entry detail page shows all entry information (name, type, time, location, description)
- [ ] #2 Shows calendar and unit context with navigation links
- [ ] #3 Shift entries display staffing status bar
- [ ] #4 Meeting entries display response summary
- [ ] #5 Participant list is displayed (everyone mode initially)
- [ ] #6 Edit/delete buttons visible to authorized users
- [ ] #7 Returns 404 for non-existent entry IDs
<!-- AC:END -->
