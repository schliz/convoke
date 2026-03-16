---
id: TASK-032
title: Plan and decompose notification system
status: To Do
assignee: []
created_date: '2026-03-16 14:36'
labels:
  - design
  - planning
milestone: m-9
dependencies:
  - TASK-024
documentation:
  - docs/design/requirements.md
priority: low
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Design the notification system in detail and create implementation subtasks. This is a planning task — review requirements §6.1-6.2 before writing code.

Notifications are described as a "first-class subsystem" in the requirements. Key areas:

**Notification events (§6.1):**
- New entry requiring response
- Entry changed (time/location/description on accepted entries)
- Entry canceled (deleted entry user had accepted)
- Reminder before entry (configurable timing)
- Response deadline approaching (meetings, pending users)
- Non-response escalation (to unit admins after deadline)
- Staffing warning (shifts below minimum near start time)
- Substitute requested / substitute found

**Channels (§6.2):**
- Email (required, default)
- Webhook (optional, for Matrix/Slack/etc.)

**Infrastructure decisions needed:**
- Task queue for async delivery (requirements mention Redis for background tasks)
- Notification scheduling: how to trigger time-based notifications (reminders, deadline warnings)
- Retry and failure logging (system must never silently fail)
- User preference storage (per-channel, per-notification-type opt-in/out)
- Per-calendar notification configuration (which types enabled, timing)

Output: detailed subtasks covering notification event triggers, email delivery, webhook delivery, scheduling, user preferences, admin configuration, and e2e tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Design document covering notification architecture, events, channels, and scheduling
- [ ] #2 Subtasks created for: event triggers, email channel, webhook channel, notification scheduling, user preferences UI, admin configuration, e2e tests
- [ ] #3 Delivery reliability approach documented (retries, failure logging)
- [ ] #4 Task queue / background job approach decided
<!-- AC:END -->
