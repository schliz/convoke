---
id: TASK-021
title: Meeting RSVP flow
status: To Do
assignee: []
created_date: '2026-03-16 14:34'
labels:
  - fullstack
milestone: m-4
dependencies:
  - TASK-019
  - TASK-016
  - TASK-015
priority: medium
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement the meeting RSVP flow. Meetings differ from shifts: attendance records are pre-created with 'pending' status for every user in the audience when the meeting is created. Users respond by changing their status to 'accepted' or 'declined'.

Meeting creation integration:
- When a meeting entry is created, automatically create 'pending' attendance records for all users in the audience. The audience defaults to the unit's members (users whose IdP groups match the unit's group bindings). For now, resolve the audience at creation time by querying users whose groups overlap with the unit's bindings.
- When the audience changes (entry edit), reconcile attendance records: add pending for new members, leave existing records unchanged.

RSVP interaction:
- Meeting cards and entry detail view show accept/decline buttons for users with pending status
- Users with accepted/declined status see their current status and can change it
- After RSVP action, re-render the meeting card via HTMX (same pattern as shift join)
- Display: accepted count, declined count, pending count out of total audience

Routes:
- POST /entries/{id}/rsvp — accepts form value 'status' (accepted/declined)

The response should re-render the meeting card component showing updated counts.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Pending attendance records created automatically when meeting is created
- [ ] #2 Users can accept or decline meeting invitations
- [ ] #3 Users can change their response (accepted ↔ declined)
- [ ] #4 Meeting card shows accepted/declined/pending counts
- [ ] #5 RSVP action re-renders meeting card via HTMX
- [ ] #6 Only users in the audience can RSVP
<!-- AC:END -->
