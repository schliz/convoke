---
id: TASK-020
title: Shift join and leave
status: To Do
assignee: []
created_date: '2026-03-16 14:34'
labels:
  - fullstack
milestone: m-4
dependencies:
  - TASK-019
  - TASK-016
priority: medium
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement the shift join/leave flow — the core interaction for volunteer shift scheduling. When a user joins a shift, an attendance record with status 'accepted' is created. When they leave, the record is deleted (or set to 'declined' if past the response deadline).

Join logic:
- User must be eligible to participate (calendar's participation policy)
- Shift must not be in the past
- Shift must not be full (if max_participants > 0, check count < max)
- User must not already be attending
- Use a transaction with GetEntryForUpdate to prevent race conditions on the max check

Leave logic:
- User must have an accepted attendance record
- If entry is in the past, cannot leave
- If past response deadline but before entry start, set status to 'needs_substitute' instead of removing (or allow direct leave — this is a UX decision; implement direct leave for now, substitution comes later)

HTMX interaction: join/leave buttons on shift cards and entry detail view. After the action, re-render the shift card component in place (hx-target="#shift-{id}", hx-swap="outerHTML"). The re-rendered card shows updated slot count and toggled button state.

Routes:
- POST /entries/{id}/join — join action
- POST /entries/{id}/leave — leave action

Both return the re-rendered shift card component for HTMX swap.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 User can join a shift and see the card update in place via HTMX
- [ ] #2 User can leave a shift and see the card update
- [ ] #3 Join is blocked when shift is full (max_participants enforced)
- [ ] #4 Join is blocked for past shifts
- [ ] #5 Join uses transaction to prevent race conditions on slot count
- [ ] #6 Leave is blocked for past shifts
- [ ] #7 Participation policy is checked before allowing join
<!-- AC:END -->
