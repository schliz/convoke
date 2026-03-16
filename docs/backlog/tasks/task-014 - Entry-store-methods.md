---
id: TASK-014
title: Entry store methods
status: To Do
assignee: []
created_date: '2026-03-16 14:33'
labels:
  - backend
milestone: m-3
dependencies:
  - TASK-003
priority: medium
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement store-layer methods for entries in internal/store/. Follow the existing DBTX pattern.

Required store methods:
- CreateEntry(ctx, db, entry) — insert a new entry
- UpdateEntry(ctx, db, entry) — update entry properties
- DeleteEntry(ctx, db, id) — delete entry (attendance cascades via FK)
- GetEntryByID(ctx, db, id) — get a single entry with calendar info
- GetEntryForUpdate(ctx, db, id) — get entry with SELECT FOR UPDATE (for transactional attendance operations)
- ListEntriesByCalendar(ctx, db, calendarID, startDate, endDate) — list entries in a date range, ordered by start_at
- ListEntriesByUnit(ctx, db, unitID, startDate, endDate) — list entries across all of a unit's calendars
- ListEntriesByUser(ctx, db, userID, startDate, endDate) — list entries the user has accepted or is pending on (for personal dashboard)

Each method should return model.Entry structs. The ListEntries methods should support both shift and meeting types — callers filter by type if needed.

The date range queries are critical for performance since they power all calendar views. Ensure they use the idx_entries_calendar_start index effectively.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 CRUD methods for entries work correctly for both shift and meeting types
- [ ] #2 Date range queries return entries ordered by start_at
- [ ] #3 GetEntryForUpdate uses SELECT FOR UPDATE for safe concurrent access
- [ ] #4 ListEntriesByUser returns entries based on attendance records
- [ ] #5 Follows existing DBTX pattern
<!-- AC:END -->
