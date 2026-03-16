---
id: TASK-015
title: Entry creation UI
status: To Do
assignee: []
created_date: '2026-03-16 14:33'
labels:
  - fullstack
milestone: m-3
dependencies:
  - TASK-014
  - TASK-010
priority: medium
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the UI for creating new entries (shifts and meetings) within a calendar. Entry creation is governed by the calendar's entry_creation policy (admins_only or unit_members).

The form should:
- Let the user select entry type (shift or meeting) — this changes which fields are shown
- Common fields: name, start date/time, end date/time, location, description, response deadline
- Shift-specific fields: required participants (min 1), maximum participants (0 = unlimited)
- Meeting-specific fields: audience selection (default: unit members, or select specific units for cross-unit meetings) — audience selection can be simplified initially to just "unit members"
- Validate: start < end, required_participants >= 1 for shifts, max >= required if max > 0

Routes:
- GET /units/{slug}/calendars/{id}/entries/new — creation form
- POST /units/{slug}/calendars/{id}/entries — create action

After successful creation, redirect to the entry detail view or back to the calendar day view for the entry's date.

Authorization: check calendar's entry_creation policy. If admins_only, require unit/association admin. If unit_members, require unit membership.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Can create a shift entry with required/max participants
- [ ] #2 Can create a meeting entry
- [ ] #3 Form shows type-specific fields based on shift/meeting selection
- [ ] #4 Validation enforces start < end and participant constraints
- [ ] #5 Entry creation policy is enforced (admins_only vs unit_members)
- [ ] #6 Successful creation redirects to entry detail or calendar day view
<!-- AC:END -->
