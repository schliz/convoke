---
id: TASK-010
title: Calendar admin UI
status: To Do
assignee: []
created_date: '2026-03-16 14:32'
labels:
  - fullstack
milestone: m-2
dependencies:
  - TASK-009
  - TASK-006
priority: medium
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the admin interface for managing calendars within a unit. Unit admins and association admins can create, edit, and delete calendars belonging to a unit.

UI needs:
- Calendar list within the unit dashboard or a dedicated unit settings page
- Create calendar form: name, color, sort order, and the four access control settings (entry_creation, visibility, participation, participant_visibility) — each as a select/dropdown with the options from requirements §3.3
- Edit calendar form (pre-populated)
- Delete calendar with confirmation modal

Routes (scoped under the unit):
- GET /units/{slug}/calendars/new — create form
- POST /units/{slug}/calendars — create action
- GET /units/{slug}/calendars/{id}/edit — edit form
- POST /units/{slug}/calendars/{id} — update action
- POST /units/{slug}/calendars/{id}/delete — delete action

Authorization: unit admin (via unit's admin group binding) or association admin. Use the handler pattern from internal/handler/ with appropriate admin checks.

The access control settings should have clear labels and help text explaining each option, since these are the primary authorization surface for the app.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Admin can create a calendar with all properties including access control settings
- [ ] #2 Admin can edit calendar properties
- [ ] #3 Admin can delete a calendar with confirmation
- [ ] #4 All four access control settings are configurable with clear labels
- [ ] #5 Non-admin users cannot access calendar management
- [ ] #6 Calendar list is visible on the unit dashboard or settings page
<!-- AC:END -->
