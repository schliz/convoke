---
id: TASK-031
title: Plan and decompose calendar integration (iCal)
status: To Do
assignee: []
created_date: '2026-03-16 14:36'
labels:
  - design
  - planning
milestone: m-8
dependencies:
  - TASK-024
documentation:
  - docs/design/requirements.md
priority: low
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Design the calendar integration system in detail and create implementation subtasks. This is a planning task — review requirements §5.1-5.2 before writing code.

Two directions of integration:

**Feed export (§5.1) — iCalendar (.ics) generation:**
- Feed scopes: per calendar, per unit, per user (personal), per user (all visible)
- Feed content: VEVENT with summary, dtstart, dtend, location, description, organizer, attendees (subject to participant visibility)
- Authentication: per-user, per-scope tokens embedded in URL (calendar clients don't support interactive auth). Tokens are revocable and regenerable.
- RFC 5545 compliance
- Feed token management UI (user settings page)

**External source import (§5.2) — iCalendar parsing:**
- External source CRUD (admin): name, feed URL, target calendar, refresh interval, enabled flag
- Background job: periodically fetch and parse external feeds, create/update/delete imported entries
- Imported entries displayed alongside native entries but visually distinguished
- Imported entries have participation='nobody' (read-only)
- Handle common RFC 5545 deviations gracefully

Output: detailed subtasks covering feed generation, token management, external source CRUD, import job, and e2e tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Design document covering iCal export and import
- [ ] #2 Subtasks created for: feed generation, feed token management, external source CRUD, import background job, e2e tests
- [ ] #3 RFC 5545 compliance approach documented
- [ ] #4 Feed authentication (token) design documented
<!-- AC:END -->
