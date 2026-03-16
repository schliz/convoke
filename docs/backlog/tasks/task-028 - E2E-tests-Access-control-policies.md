---
id: TASK-028
title: 'E2E tests: Access control policies'
status: To Do
assignee: []
created_date: '2026-03-16 14:35'
labels:
  - testing
milestone: m-5
dependencies:
  - TASK-025
  - TASK-026
  - TASK-027
priority: low
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Write Playwright e2e tests verifying all access control policies are enforced correctly. These tests require multiple user accounts with different roles and group memberships.

Seed data: at least 3 test users (association admin, unit member, non-member), units with various calendar configurations (different visibility, participation, creation, and participant visibility settings).

Test scenarios:
- Unit-scoped calendar invisible to non-members
- Custom visibility calendar accessible only to designated units
- Entry creation blocked for non-authorized users
- Participation blocked by participation policy
- Participant names hidden from non-members (unit visibility mode)
- Participant names hidden from non-participants (participants_only mode)
- Admin bypasses all restrictions

These tests are critical for security — access control bugs could expose private scheduling data.

Follow existing Playwright project structure. May need a third test project for non-member users.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Tests verify calendar visibility enforcement across all modes
- [ ] #2 Tests verify entry creation policy enforcement
- [ ] #3 Tests verify participation policy enforcement
- [ ] #4 Tests verify participant visibility rules
- [ ] #5 Tests use at least 3 user roles (admin, member, non-member)
- [ ] #6 All access control tests pass consistently
<!-- AC:END -->
