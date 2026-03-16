---
id: TASK-034
title: Plan and decompose internationalization (i18n)
status: To Do
assignee: []
created_date: '2026-03-16 14:36'
labels:
  - design
  - planning
milestone: m-10
dependencies:
  - TASK-024
documentation:
  - docs/design/requirements.md
priority: low
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Design the internationalization approach and create implementation subtasks. Review requirements §10.4.

Requirements:
- Support German and English
- All user-facing text must be translatable
- Date, time, and number formatting must respect locale settings

Design decisions needed:
- Translation approach for Go templates: message catalogs, gettext, go-i18n, or custom solution
- How locale is determined (user preference? browser Accept-Language? association default?)
- How to handle date/time formatting in templates (the existing formatTime/formatDate helpers need locale awareness)
- Translation workflow: how are translation files maintained?

This is a cross-cutting concern that touches every template and many handlers. It should be planned carefully to minimize the retrofit effort.

Output: subtasks for i18n infrastructure, template translation, date/time formatting, and language switching UI.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Design document covering i18n approach for Go templates
- [ ] #2 Subtasks created for: i18n infrastructure, template translation, date/time locale formatting, language switching
- [ ] #3 Translation file format and workflow documented
- [ ] #4 Locale detection strategy decided
<!-- AC:END -->
