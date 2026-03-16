---
id: TASK-027
title: Participant visibility enforcement
status: To Do
assignee: []
created_date: '2026-03-16 14:35'
labels:
  - fullstack
milestone: m-5
dependencies:
  - TASK-019
  - TASK-016
priority: medium
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement participant visibility rules that control who can see the names of participants on entries (§3.3):

- 'everyone': anyone who can see the entry sees full participant names
- 'unit': non-members see aggregate counts only (e.g., "3/5 filled"), unit members see names
- 'participants_only': only users who have joined an entry see other participants

This affects:
- Entry detail view participant list
- Shift card display (names vs counts)
- Meeting card display (names vs counts)
- iCal feed export attendee fields (future, but design for it now)

Implementation: the view model builder methods must check the calendar's participant_visibility setting and the user's relationship to the unit/entry. The view model struct should include a ParticipantNames field (populated or empty based on visibility) and aggregate counts (always populated). Templates render based on what's available.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 'everyone' mode shows participant names to all viewers
- [ ] #2 'unit' mode shows names to unit members, counts to others
- [ ] #3 'participants_only' mode shows names only to users who have joined
- [ ] #4 Aggregate counts always available regardless of visibility mode
- [ ] #5 View model structs handle both name and count display modes
<!-- AC:END -->
