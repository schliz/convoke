---
id: TASK-024
title: 'E2E tests: Attendance workflows'
status: To Do
assignee: []
created_date: '2026-03-16 14:34'
labels:
  - testing
milestone: m-4
dependencies:
  - TASK-020
  - TASK-021
  - TASK-022
  - TASK-023
priority: low
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Write Playwright e2e tests covering all attendance interactions: shift join/leave, meeting RSVP, personal dashboard, and substitution.

Seed data: units, calendars, entries (shifts and meetings) with various states. Multiple test users with different roles (member, admin, non-member) to test participation policies.

Test scenarios:
- User joins a shift, slot count updates
- User leaves a shift
- Join blocked when shift is full
- User accepts/declines a meeting
- User changes RSVP response
- Personal dashboard shows correct entries for the user
- Substitute request and claim flow
- Participation policy prevents unauthorized join/RSVP

These tests exercise the core value proposition of the app (binding attendance commitments) and should be thorough.

Follow existing Playwright project structure with member and admin test projects.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Tests cover shift join/leave with slot count verification
- [ ] #2 Tests cover meeting RSVP accept/decline
- [ ] #3 Tests verify personal dashboard content
- [ ] #4 Tests cover substitution request and claim
- [ ] #5 Tests verify participation policy enforcement
- [ ] #6 Tests use multiple user roles (member, admin, non-member)
<!-- AC:END -->
