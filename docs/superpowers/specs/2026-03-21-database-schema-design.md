# Database Schema Design — Convoke

> TASK-001: Design complete database schema
> Date: 2026-03-21
> Status: Approved
> References: docs/design/requirements.md, docs/design/database.dbml, docs/backlog/documents/doc-001

---

## Overview

This document specifies the complete PostgreSQL database schema for Convoke. It covers all entities from requirements sections 3-7 and defines columns, types, constraints, indexes, foreign keys, and deletion behavior.

The schema diverges from the DBML reference design in several ways, documented in DOC-001 (Architecture Decision Record). Key decisions:

- **BIGSERIAL** primary keys with opaque text slugs on URL-facing entities (not UUIDs)
- **CHECK constraints** for enum-like columns (not native PostgreSQL ENUMs)
- **Hard delete with CASCADE** (no soft deletes; stats preserved via snapshots — snapshot table deferred to milestone 10)
- **Events as loose calendar groupings** via junction table (not 1:1 calendar extension)
- **sqlc** for Go code generation from SQL queries
- **Normalized IdP groups** in a junction table (not TEXT[] on users)

### Entities requiring opaque URL slug

units (user-chosen slug), calendars, entries, events

### Entities with internal-only integer PK

attendances, substitution_requests, templates, template_groups, recurrence_rules, notification_configs, user_notification_preferences, notifications, webhooks, feed_tokens, external_sources, external_entries, user_idp_groups, unit_group_bindings, calendar_custom_viewers, entry_shift_details, meeting_audience_units, event_calendars, entry_annotations

---

## 1. Identity & Group Sync

### users

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| idp_subject | TEXT | NOT NULL, UNIQUE |
| username | TEXT | NOT NULL |
| display_name | TEXT | NOT NULL |
| email | TEXT | NOT NULL |
| timezone | TEXT | NULL = association default |
| locale | TEXT | NULL = association default |
| is_assoc_admin | BOOLEAN | NOT NULL DEFAULT false |
| last_login_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes:** `idp_subject` (unique, implicit), `email`

**Notes:** User records created on first OIDC login, updated from IdP claims on subsequent logins. `idp_subject` is the stable external identifier (OIDC sub claim). `is_assoc_admin` derived from IdP role on login sync.

### user_idp_groups

| Column | Type | Constraints |
|--------|------|-------------|
| user_id | BIGINT | NOT NULL, FK → users(id) ON DELETE CASCADE |
| group_name | TEXT | NOT NULL |

**Primary key:** (user_id, group_name)
**Indexes:** `group_name`

**Notes:** Fully replaced on each login sync. Represents the user's current IdP group memberships.

---

## 2. Units

### units

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| name | TEXT | NOT NULL |
| slug | TEXT | NOT NULL, UNIQUE |
| description | TEXT | |
| logo_path | TEXT | Relative path or object-storage key |
| contact_email | TEXT | |
| admin_group | TEXT | IdP group for unit admins; NULL = assoc admins only |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes:** `slug` (unique, implicit)

**Notes:** `slug` is user-chosen (e.g., `bar-committee`) and serves as the URL identifier. No separate opaque ID needed.

### unit_group_bindings

| Column | Type | Constraints |
|--------|------|-------------|
| unit_id | BIGINT | NOT NULL, FK → units(id) ON DELETE CASCADE |
| group_name | TEXT | NOT NULL |

**Primary key:** (unit_id, group_name)
**Indexes:** `group_name`

**Membership resolution pattern:**
```sql
SELECT 1 FROM user_idp_groups uig
JOIN unit_group_bindings ugb ON uig.group_name = ugb.group_name
WHERE uig.user_id = $1 AND ugb.unit_id = $2
LIMIT 1
```

---

## 3. Calendars & Events

### calendars

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| slug | TEXT | NOT NULL, UNIQUE |
| unit_id | BIGINT | NOT NULL, FK → units(id) ON DELETE CASCADE |
| name | TEXT | NOT NULL |
| creation_policy | TEXT | NOT NULL DEFAULT 'admins_only', CHECK IN ('admins_only', 'unit_members') |
| visibility | TEXT | NOT NULL DEFAULT 'association', CHECK IN ('association', 'unit', 'custom') |
| participation | TEXT | NOT NULL DEFAULT 'viewers', CHECK IN ('viewers', 'unit', 'nobody') |
| participant_visibility | TEXT | NOT NULL DEFAULT 'everyone', CHECK IN ('everyone', 'unit', 'participants_only') |
| color | TEXT | Hex color, e.g. #3b82f6 |
| sort_order | INT | NOT NULL DEFAULT 0 |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes:** `slug` (unique, implicit), `unit_id`

### calendar_custom_viewers

| Column | Type | Constraints |
|--------|------|-------------|
| calendar_id | BIGINT | NOT NULL, FK → calendars(id) ON DELETE CASCADE |
| unit_id | BIGINT | NOT NULL, FK → units(id) ON DELETE CASCADE |

**Primary key:** (calendar_id, unit_id)

**Notes:** Only populated for calendars with `visibility = 'custom'`. Lists which units can see the calendar.

### events

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| slug | TEXT | NOT NULL, UNIQUE |
| unit_id | BIGINT | NOT NULL, FK → units(id) ON DELETE CASCADE |
| name | TEXT | NOT NULL |
| start_date | DATE | NOT NULL |
| end_date | DATE | NOT NULL |
| website | TEXT | |
| description | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Check:** `end_date >= start_date`
**Indexes:** `slug` (unique, implicit), `unit_id`

**Notes:** `unit_id` identifies the coordinating unit, not an ownership constraint on linked calendars. Events are a loose grouping — see ADR-7. Access control for the event timeline view is derived from the visibility settings of the constituent calendars in `event_calendars`, not from properties on the event itself.

### event_calendars

| Column | Type | Constraints |
|--------|------|-------------|
| event_id | BIGINT | NOT NULL, FK → events(id) ON DELETE CASCADE |
| calendar_id | BIGINT | NOT NULL, FK → calendars(id) ON DELETE CASCADE |
| sort_order | INT | NOT NULL DEFAULT 0 |

**Primary key:** (event_id, calendar_id)

**Notes:** Many-to-many junction. Deleting an event removes junction rows but leaves calendars intact. Deleting a calendar removes it from all events.

---

## 4. Entries

### entries

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| slug | TEXT | NOT NULL, UNIQUE |
| calendar_id | BIGINT | NOT NULL, FK → calendars(id) ON DELETE CASCADE |
| name | TEXT | NOT NULL |
| type | TEXT | NOT NULL, CHECK IN ('shift', 'meeting') |
| starts_at | TIMESTAMPTZ | NOT NULL |
| ends_at | TIMESTAMPTZ | NOT NULL |
| location | TEXT | |
| description | TEXT | |
| response_deadline | TIMESTAMPTZ | |
| recurrence_rule_id | BIGINT | FK → recurrence_rules(id) ON DELETE SET NULL |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Check:** `ends_at > starts_at`
**Indexes:** `slug` (unique, implicit), `(calendar_id, starts_at)` (composite), `starts_at`, `recurrence_rule_id`
**Unique:** (calendar_id, name, starts_at) — idempotency guard for template instantiation

**Notes:** `recurrence_rule_id` uses ON DELETE SET NULL — entries survive rule deletion but lose the back-reference.

### entry_shift_details

| Column | Type | Constraints |
|--------|------|-------------|
| entry_id | BIGINT | PRIMARY KEY, FK → entries(id) ON DELETE CASCADE |
| required_participants | INT | NOT NULL, CHECK >= 1 |
| max_participants | INT | NOT NULL DEFAULT 0 (0 = unlimited) |

### meeting_audience_units

| Column | Type | Constraints |
|--------|------|-------------|
| entry_id | BIGINT | NOT NULL, FK → entries(id) ON DELETE CASCADE |
| unit_id | BIGINT | NOT NULL, FK → units(id) ON DELETE CASCADE |

**Primary key:** (entry_id, unit_id)

**Notes:** Overrides the default audience (calendar's owning unit) for cross-unit meetings. When no rows exist, the audience defaults to the calendar's unit.

### entry_annotations

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| entry_id | BIGINT | NOT NULL, FK → entries(id) ON DELETE CASCADE |
| kind | TEXT | NOT NULL (e.g. 'weekend_warning', 'holiday_warning') |
| message | TEXT | NOT NULL |

**Indexes:** `entry_id`

---

## 5. Attendance & Substitution

### attendances

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| entry_id | BIGINT | NOT NULL, FK → entries(id) ON DELETE CASCADE |
| user_id | BIGINT | NOT NULL, FK → users(id) ON DELETE CASCADE |
| status | TEXT | NOT NULL DEFAULT 'pending', CHECK IN ('pending', 'accepted', 'declined', 'needs_substitute', 'replaced') |
| confirmed | BOOLEAN | NOT NULL DEFAULT false |
| note | TEXT | |
| responded_at | TIMESTAMPTZ | When status last changed from pending |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Unique:** (entry_id, user_id)
**Indexes:** `entry_id`, `user_id`, (user_id, status)

**Notes:** `confirmed` is for admin post-hoc attendance confirmation (UI deferred per §11, data model included for statistics). Stats snapshots capture both `accepted` and `confirmed` counts before data is deleted.

### substitution_requests

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| attendance_id | BIGINT | NOT NULL, UNIQUE, FK → attendances(id) ON DELETE CASCADE |
| claimed_by_user_id | BIGINT | FK → users(id) ON DELETE SET NULL |
| claimed_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Notes:** `claimed_by_user_id` uses SET NULL — if the claiming user is deleted, the request reverts to unclaimed. One substitution request per attendance (UNIQUE constraint).

---

## 6. Templates & Recurrence

### template_groups

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| unit_id | BIGINT | NOT NULL, FK → units(id) ON DELETE CASCADE |
| calendar_id | BIGINT | NOT NULL, FK → calendars(id) ON DELETE CASCADE |
| name | TEXT | NOT NULL |
| base_start_time | TIME | NOT NULL |
| location | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes:** `unit_id`, `calendar_id`

### templates

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| template_group_id | BIGINT | NOT NULL, FK → template_groups(id) ON DELETE CASCADE |
| name | TEXT | NOT NULL |
| type | TEXT | NOT NULL, CHECK IN ('shift', 'meeting') |
| start_offset | INTERVAL | NOT NULL |
| duration | INTERVAL | NOT NULL |
| required_participants | INT | Shift only |
| max_participants | INT | Shift only; 0 = unlimited |
| description | TEXT | |
| response_deadline_offset | INTERVAL | Duration before entry start |
| sort_order | INT | NOT NULL DEFAULT 0 |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes:** `template_group_id`

### recurrence_rules

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| template_group_id | BIGINT | NOT NULL, FK → template_groups(id) ON DELETE CASCADE |
| pattern_type | TEXT | NOT NULL, CHECK IN ('nth_weekday_of_month', 'nth_day_of_month', 'every_nth_weekday', 'nth_workday_of_month', 'nth_day_of_year', 'nth_workday_of_year') |
| pattern_params | JSONB | NOT NULL |
| first_occurrence | DATE | NOT NULL |
| auto_create_horizon | INT | NOT NULL DEFAULT 14 |
| enabled | BOOLEAN | NOT NULL DEFAULT true |
| weekend_action | TEXT | NOT NULL DEFAULT 'ignore', CHECK IN ('ignore', 'skip', 'warn') |
| weekend_warning_text | TEXT | |
| holiday_action | TEXT | NOT NULL DEFAULT 'ignore', CHECK IN ('ignore', 'skip', 'warn') |
| holiday_warning_text | TEXT | |
| last_evaluated_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes:** `template_group_id`, `enabled`

**Notes:** `pattern_params` shape depends on `pattern_type`. Validated at the application level. Deleting a template group cascades to its templates and recurrence rules. Entries already created survive via ON DELETE SET NULL on `entries.recurrence_rule_id`.

---

## 7. Calendar Integration

### feed_tokens

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| user_id | BIGINT | NOT NULL, FK → users(id) ON DELETE CASCADE |
| scope | TEXT | NOT NULL, CHECK IN ('calendar', 'unit', 'personal', 'all_visible') |
| scope_id | BIGINT | calendars.id or units.id; NULL for personal/all_visible |
| token | TEXT | NOT NULL, UNIQUE |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| revoked_at | TIMESTAMPTZ | |

**Indexes:** `token` (unique, implicit), `user_id`, (user_id, scope, scope_id)

**Notes:** `scope_id` is polymorphic — references calendars or units depending on `scope`. No FK constraint; validated at the application level. `token` is the opaque URL identifier by nature.

### external_sources

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| name | TEXT | NOT NULL |
| feed_url | TEXT | NOT NULL |
| calendar_id | BIGINT | NOT NULL, FK → calendars(id) ON DELETE CASCADE |
| refresh_interval | INTERVAL | NOT NULL DEFAULT '1 hour' |
| enabled | BOOLEAN | NOT NULL DEFAULT true |
| last_fetched_at | TIMESTAMPTZ | |
| last_error | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes:** `calendar_id`

### external_entries

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| external_source_id | BIGINT | NOT NULL, FK → external_sources(id) ON DELETE CASCADE |
| uid | TEXT | NOT NULL (iCal UID for dedup) |
| summary | TEXT | |
| starts_at | TIMESTAMPTZ | NOT NULL |
| ends_at | TIMESTAMPTZ | |
| location | TEXT | |
| description | TEXT | |
| raw_ical | TEXT | Original VEVENT blob |
| fetched_at | TIMESTAMPTZ | NOT NULL |

**Unique:** (external_source_id, uid)
**Indexes:** `external_source_id`, `starts_at`

---

## 8. Notifications & Webhooks

### notification_configs

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| calendar_id | BIGINT | NOT NULL, FK → calendars(id) ON DELETE CASCADE |
| event_type | TEXT | NOT NULL, CHECK IN ('new_entry', 'entry_changed', 'entry_canceled', 'reminder_before_entry', 'response_deadline_approaching', 'non_response_escalation', 'staffing_warning', 'substitute_requested', 'substitute_found') |
| enabled | BOOLEAN | NOT NULL DEFAULT true |
| lead_time | INTERVAL | For timed notifications |

**Unique:** (calendar_id, event_type)

### user_notification_preferences

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| user_id | BIGINT | NOT NULL, FK → users(id) ON DELETE CASCADE |
| event_type | TEXT | NOT NULL, CHECK IN ('new_entry', 'entry_changed', 'entry_canceled', 'reminder_before_entry', 'response_deadline_approaching', 'non_response_escalation', 'staffing_warning', 'substitute_requested', 'substitute_found') |
| channel | TEXT | NOT NULL, CHECK IN ('email', 'webhook') |
| enabled | BOOLEAN | NOT NULL DEFAULT true |

**Unique:** (user_id, event_type, channel)

### notifications

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| user_id | BIGINT | NOT NULL, FK → users(id) ON DELETE CASCADE |
| entry_id | BIGINT | FK → entries(id) ON DELETE SET NULL |
| event_type | TEXT | NOT NULL, CHECK IN ('new_entry', 'entry_changed', 'entry_canceled', 'reminder_before_entry', 'response_deadline_approaching', 'non_response_escalation', 'staffing_warning', 'substitute_requested', 'substitute_found') |
| channel | TEXT | NOT NULL, CHECK IN ('email', 'webhook') |
| status | TEXT | NOT NULL DEFAULT 'pending', CHECK IN ('pending', 'sent', 'failed', 'retrying') |
| payload | JSONB | Rendered content / template data |
| error | TEXT | |
| retry_count | INT | NOT NULL DEFAULT 0 |
| sent_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes:** `user_id`, `entry_id`, (status, created_at), (user_id, event_type, entry_id)

**Notes:** `entry_id` uses ON DELETE SET NULL — delivery log survives entry deletion for auditing.

### webhooks

| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| unit_id | BIGINT | FK → units(id) ON DELETE CASCADE; NULL = association-wide |
| name | TEXT | NOT NULL |
| url | TEXT | NOT NULL |
| secret | TEXT | HMAC signing secret |
| enabled | BOOLEAN | NOT NULL DEFAULT true |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes:** `unit_id`

---

## Index Strategy Summary

| Pattern | Index | Justification |
|---------|-------|---------------|
| Entries by calendar + date range | `entries(calendar_id, starts_at)` | Primary calendar view query — composite index serves the most common query with a single index scan |
| Entries by date range (cross-calendar) | `entries(starts_at)` | Notification scheduling: "all entries in next 24h" |
| Events by coordinating unit | `events(unit_id)` | Unit dashboard: show unit's events |
| Template idempotency | `entries(calendar_id, name, starts_at)` UNIQUE | Prevents duplicate entries from template instantiation |
| Attendance by entry | `attendances(entry_id)` | Entry detail view: list participants |
| Attendance by user + status | `attendances(user_id, status)` | Personal dashboard: "my upcoming shifts" |
| Membership resolution | `user_idp_groups(group_name)`, `unit_group_bindings(group_name)` | JOIN-based membership check |
| Unit by slug | `units(slug)` UNIQUE | URL routing |
| Calendar by slug | `calendars(slug)` UNIQUE | URL routing |
| Entry by slug | `entries(slug)` UNIQUE | URL routing |
| Event by slug | `events(slug)` UNIQUE | URL routing |
| Feed token lookup | `feed_tokens(token)` UNIQUE | iCal feed authentication |
| Notification queue processing | `notifications(status, created_at)` | Background worker: pick pending/retrying |
| External entry dedup | `external_entries(external_source_id, uid)` UNIQUE | iCal import dedup |
| Recurrence cron | `recurrence_rules(enabled)` | Daily cron: find active rules |

---

## Deletion Cascade Summary

| When deleted | Cascades to |
|-------------|-------------|
| user | user_idp_groups, attendances (→ substitution_requests), feed_tokens, user_notification_preferences, notifications; substitution_requests.claimed_by_user_id SET NULL |
| unit | unit_group_bindings, calendars (→ full subtree), events (→ event_calendars), template_groups (→ templates, recurrence_rules), webhooks, calendar_custom_viewers (via unit_id FK), meeting_audience_units (via unit_id FK — removes unit from cross-unit meeting audiences) |
| calendar | calendar_custom_viewers, entries (→ full subtree), event_calendars, external_sources (→ external_entries), notification_configs, template_groups (→ templates, recurrence_rules) |
| entry | entry_shift_details, meeting_audience_units, attendances (→ substitution_requests), entry_annotations; notifications.entry_id SET NULL |
| event | event_calendars (calendars remain) |
| template_group | templates, recurrence_rules; entries.recurrence_rule_id SET NULL |
| external_source | external_entries |

---

## Future Extension: Guest Participation

Not included in this schema. Validated as addable without breaking changes:

1. Add `guests` table (name, email, token for session)
2. Make `attendances.user_id` nullable
3. Add `attendances.guest_id` nullable FK → guests
4. Add CHECK: exactly one of user_id/guest_id is non-null

No existing data requires migration.

---

## Table Count

24 tables:

1. users
2. user_idp_groups
3. units
4. unit_group_bindings
5. calendars
6. calendar_custom_viewers
7. events
8. event_calendars
9. entries
10. entry_shift_details
11. meeting_audience_units
12. entry_annotations
13. attendances
14. substitution_requests
15. template_groups
16. templates
17. recurrence_rules
18. feed_tokens
19. external_sources
20. external_entries
21. notification_configs
22. user_notification_preferences
23. notifications
24. webhooks
