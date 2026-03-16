---
id: TASK-007
title: Unit dashboard page
status: To Do
assignee: []
created_date: '2026-03-16 14:32'
labels:
  - fullstack
milestone: m-1
dependencies:
  - TASK-006
priority: medium
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the unit dashboard page — the landing page when a user navigates to a specific unit. This page provides an overview of the unit's activity and is the entry point to its calendars.

The dashboard should display:
- Unit name and description
- List of the unit's calendars (with links to their views)
- Upcoming entries across all the unit's calendars (next 7-14 days)
- Entries needing attention: understaffed shifts approaching their start time, meetings with pending responses near deadline (this can be simplified for now — just show upcoming entries sorted by date)

Route: GET /units/{slug}

The page needs:
- A page template in templates/pages/
- A view model struct in internal/viewmodel/
- A handler method building the view model from store queries
- Store methods for listing calendars by unit and listing upcoming entries by unit

For now, the "entries needing attention" section can be a simplified list. It will be refined when attendance features are built.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Unit dashboard accessible at /units/{slug}
- [ ] #2 Displays unit name and description
- [ ] #3 Lists the unit's calendars with links
- [ ] #4 Shows upcoming entries from the unit's calendars
- [ ] #5 Returns 404 for non-existent unit slugs
- [ ] #6 View model struct follows typed component pattern
<!-- AC:END -->
