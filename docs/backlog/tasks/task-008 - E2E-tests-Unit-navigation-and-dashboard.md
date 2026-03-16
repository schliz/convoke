---
id: TASK-008
title: 'E2E tests: Unit navigation and dashboard'
status: To Do
assignee: []
created_date: '2026-03-16 14:32'
labels:
  - testing
milestone: m-1
dependencies:
  - TASK-005
  - TASK-006
  - TASK-007
priority: low
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Write Playwright e2e tests covering unit navigation and dashboard functionality. The test infrastructure is scaffolded in e2e/ with playwright.config.ts, auth.setup.ts, and a health check test as reference.

Tests need seed data: create test units with group bindings in test/seed.sql (or a dedicated seed file). The seed should include at least two units — one the test user is a member of, one they are not.

Test scenarios:
- Navigation shows only units the user belongs to
- Clicking a unit navigates to its dashboard
- Unit dashboard displays the unit's name and calendars
- Non-existent unit slug shows 404
- Admin user sees admin link in navigation

Follow the existing Playwright config pattern: member-tests project uses member auth state, admin-tests project uses admin auth state.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Seed data creates test units with group bindings
- [ ] #2 Test verifies navigation shows correct units for user
- [ ] #3 Test verifies unit dashboard displays expected content
- [ ] #4 Test verifies 404 for non-existent unit
- [ ] #5 Tests follow existing Playwright project structure (member vs admin)
<!-- AC:END -->
