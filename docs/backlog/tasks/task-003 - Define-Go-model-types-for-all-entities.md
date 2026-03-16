---
id: TASK-003
title: Define Go model types for all entities
status: To Do
assignee: []
created_date: '2026-03-16 14:31'
updated_date: '2026-03-16 14:36'
labels:
  - backend
milestone: m-0
dependencies:
  - TASK-002
priority: medium
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create Go structs in internal/model/ for all database entities defined in the migrations. The existing codebase has internal/model/user.go with the User struct as a reference pattern.

Each model struct should:
- Match the database column names and types
- Use appropriate Go types (time.Time for timestamps, []string for text arrays, *string for nullable text, etc.)
- Use int64 for IDs (matching BIGSERIAL)
- Follow the existing pattern in model/user.go

This is a mechanical translation of the schema into Go types. No business logic in model types — they are pure data containers.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Model structs exist for all database entities (units, calendars, entries, attendance, template groups, templates, recurrence rules, external sources, feed tokens)
- [ ] #2 Field types match database column types correctly
- [ ] #3 Follows existing pattern in internal/model/user.go
- [ ] #4 No business logic in model types
<!-- AC:END -->
