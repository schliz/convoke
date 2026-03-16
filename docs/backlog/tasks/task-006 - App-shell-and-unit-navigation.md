---
id: TASK-006
title: App shell and unit navigation
status: To Do
assignee: []
created_date: '2026-03-16 14:31'
labels:
  - fullstack
milestone: m-1
dependencies:
  - TASK-004
priority: medium
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the top-level app shell: the navigation bar, unit listing/switcher, and overall page layout that all other pages will live inside.

The existing codebase has templates/layouts/base.html with a basic layout and templates/components/nav.html with a starter nav bar. These need to be extended to include:

- Navigation showing units the current user belongs to (resolved via their IdP groups)
- A way to switch between units (sidebar, dropdown, or nav links)
- Link to the personal dashboard (future, but reserve the nav slot)
- Admin link (visible only to association admins)
- The existing theme toggle should be preserved

The root route (/) currently redirects — it should redirect to the personal dashboard (or a unit listing if dashboard isn't built yet).

The nav component's view model (internal/viewmodel/layout.go LayoutData) needs to be extended with the user's units list and admin status.

Follow the component-oriented template architecture: typed view model structs, precomputed booleans for conditional rendering.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Navigation displays units the current user belongs to
- [ ] #2 User can navigate between different units
- [ ] #3 Admin link visible only to association admins
- [ ] #4 Theme toggle preserved from existing nav
- [ ] #5 Root route redirects to a sensible default page
- [ ] #6 LayoutData view model extended with units and admin status
<!-- AC:END -->
