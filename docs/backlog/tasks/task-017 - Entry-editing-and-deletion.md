---
id: TASK-017
title: Entry editing and deletion
status: To Do
assignee: []
created_date: '2026-03-16 14:33'
labels:
  - fullstack
milestone: m-3
dependencies:
  - TASK-015
  - TASK-016
priority: medium
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the UI for editing and deleting existing entries. Only users authorized by the calendar's entry_creation policy (or unit/association admins) can edit/delete entries.

Edit form:
- Pre-populated with current entry values
- Same fields and validation as creation form
- Type cannot be changed after creation (shift stays shift, meeting stays meeting)

Delete:
- Confirmation modal (loaded via HTMX into the modal container)
- Cascades to attendance records (via FK)
- After deletion, redirect to the calendar day view

Routes:
- GET /entries/{id}/edit — edit form
- POST /entries/{id} — update action
- GET /entries/{id}/confirm-delete — confirmation modal fragment
- POST /entries/{id}/delete — delete action

HTMX patterns: edit form can be a full page or a modal. Delete confirmation should use the modal pattern from the go-htmx-fullstack skill (load into #modal container, close on success).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Edit form is pre-populated with current entry values
- [ ] #2 Entry type cannot be changed during editing
- [ ] #3 Validation is enforced on edit (same rules as creation)
- [ ] #4 Delete shows confirmation before proceeding
- [ ] #5 Successful deletion redirects to calendar day view
- [ ] #6 Only authorized users can edit/delete (entry creation policy or admin)
<!-- AC:END -->
