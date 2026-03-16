---
id: TASK-023
title: Substitution flow
status: To Do
assignee: []
created_date: '2026-03-16 14:34'
labels:
  - fullstack
milestone: m-4
dependencies:
  - TASK-020
priority: medium
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement the substitution mechanism for shifts. When a user who has accepted a shift can no longer attend, they can request a substitute instead of simply leaving.

Flow (from requirements §3.7):
1. User marks their attendance as 'needs_substitute'
2. The shift card shows a "substitute needed" indicator visible to eligible users
3. Another eligible user claims the slot — this creates their 'accepted' attendance
4. The original user's attendance is changed to 'replaced' (terminal status for record-keeping)

UI interactions:
- On the shift card/detail view: if the user is attending and the shift is in the future, show a "Find substitute" button (in addition to the regular "Leave" button)
- Shift cards for entries with needs_substitute attendees should show a visual indicator (e.g., warning badge)
- Eligible users (can participate, not already attending) see a "Claim" button on shifts needing substitutes

Routes:
- POST /entries/{id}/substitute — request substitute (sets current user to needs_substitute)
- POST /entries/{id}/claim — claim a substitute slot (new user accepts, original user set to replaced)

Both return re-rendered shift card via HTMX.

Note: Notification of eligible users when a substitute is requested is deferred to the Notifications milestone.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 User can request a substitute for an accepted shift
- [ ] #2 Shift card shows substitute-needed indicator
- [ ] #3 Eligible users can claim a substitute slot
- [ ] #4 Claiming sets new user to accepted and original to replaced
- [ ] #5 Only future shifts allow substitute requests
- [ ] #6 HTMX in-place updates for all actions
<!-- AC:END -->
