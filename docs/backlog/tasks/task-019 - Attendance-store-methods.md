---
id: TASK-019
title: Attendance store methods
status: To Do
assignee: []
created_date: '2026-03-16 14:34'
labels:
  - backend
milestone: m-4
dependencies:
  - TASK-003
priority: medium
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement store-layer methods for attendance records in internal/store/. Follow the existing DBTX pattern.

Required store methods:
- CreateAttendance(ctx, db, entryID, userID, status) — create an attendance record
- UpdateAttendanceStatus(ctx, db, entryID, userID, status) — change attendance status
- DeleteAttendance(ctx, db, entryID, userID) — remove attendance record
- GetAttendance(ctx, db, entryID, userID) — get a user's attendance for an entry
- ListAttendeesByEntry(ctx, db, entryID) — list all attendance records for an entry (with user info via JOIN)
- CountAttendeesByStatus(ctx, db, entryID, status) — count attendees with a given status (accepted, declined, pending)
- CountAcceptedAttendees(ctx, db, entryID) — shorthand for counting accepted (used for staffing checks)
- IsAttendee(ctx, db, entryID, userID) — check if user has any attendance record
- BulkCreatePendingAttendance(ctx, db, entryID, userIDs) — create pending records for a list of users (for meeting audience)
- ListEntriesWithUserAttendance(ctx, db, userID, startDate, endDate) — list entries with the user's attendance status attached (for personal dashboard)

The UNIQUE(entry_id, user_id) constraint on the attendance table means CreateAttendance should handle the conflict case (either error or upsert, depending on the caller's needs).

These methods are performance-critical since they're called on every entry card render. CountAcceptedAttendees in particular should be efficient.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 CRUD methods for attendance records work correctly
- [ ] #2 Can count attendees by status efficiently
- [ ] #3 Can bulk-create pending attendance for meeting audiences
- [ ] #4 Can list entries with user's attendance status for personal dashboard
- [ ] #5 UNIQUE constraint is handled properly on create
- [ ] #6 Follows existing DBTX pattern
<!-- AC:END -->
