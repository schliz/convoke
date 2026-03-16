---
id: TASK-018
title: 'E2E tests: Entry lifecycle'
status: To Do
assignee: []
created_date: '2026-03-16 14:33'
labels:
  - testing
milestone: m-3
dependencies:
  - TASK-015
  - TASK-016
  - TASK-017
priority: low
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Write Playwright e2e tests covering entry creation, viewing, editing, and deletion for both shift and meeting types.

Seed data: units with calendars. Tests will create entries through the UI.

Test scenarios:
- Create a shift entry with required/max participants
- Create a meeting entry
- View entry detail page with correct information
- Edit an entry and verify changes are saved
- Delete an entry with confirmation
- Verify validation errors (end before start, missing required fields)
- Verify non-authorized user cannot create/edit/delete entries

Follow existing Playwright project structure.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Tests cover shift and meeting entry creation
- [ ] #2 Tests verify entry detail view displays correct data
- [ ] #3 Tests cover entry editing
- [ ] #4 Tests cover entry deletion with confirmation
- [ ] #5 Tests verify form validation
- [ ] #6 Tests verify authorization enforcement
<!-- AC:END -->
