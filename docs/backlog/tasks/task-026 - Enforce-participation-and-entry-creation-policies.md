---
id: TASK-026
title: Enforce participation and entry creation policies
status: To Do
assignee: []
created_date: '2026-03-16 14:35'
labels:
  - backend
milestone: m-5
dependencies:
  - TASK-019
  - TASK-015
priority: medium
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement enforcement of calendar participation and entry creation policies across all relevant actions.

Entry creation policy (§3.3):
- 'admins_only': only unit admins and association admins can create entries
- 'unit_members': any member of the owning unit can create entries
Enforce on: entry creation form access, entry creation POST, entry edit, entry delete

Participation policy (§3.3):
- 'viewers': anyone who can see the calendar can join/RSVP
- 'unit': only members of the owning unit can participate
- 'nobody': no attendance interaction (used for imported calendars)
Enforce on: shift join, meeting RSVP, substitute claim

These checks should be implemented as handler-level helpers that return appropriate errors (ForbiddenError) when the user doesn't have permission. The precomputed boolean flags in view model structs (CanJoin, CanEdit, etc.) must respect these policies.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Entry creation policy enforced on create/edit/delete actions
- [ ] #2 Participation policy enforced on join/RSVP/claim actions
- [ ] #3 View model boolean flags (CanJoin, CanEdit) respect policies
- [ ] #4 ForbiddenError returned for unauthorized actions
- [ ] #5 Calendar with participation='nobody' blocks all attendance actions
<!-- AC:END -->
