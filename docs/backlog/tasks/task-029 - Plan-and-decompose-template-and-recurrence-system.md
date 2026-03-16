---
id: TASK-029
title: Plan and decompose template and recurrence system
status: To Do
assignee: []
created_date: '2026-03-16 14:35'
labels:
  - design
  - planning
milestone: m-6
dependencies:
  - TASK-017
documentation:
  - docs/design/requirements.md
  - docs/design/architecture-learnings.md
priority: low
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Design the template and recurrence system in detail and create implementation subtasks. This is a planning task — review requirements §4.1-4.4 and design the implementation approach before writing code.

Key areas to design:
- Template group and template CRUD (admin UI within unit settings)
- Template instantiation: applying a template group to a date to produce entries. Must be idempotent (duplicate detection via calendar + name + start_at).
- Recurrence rules: storing pattern types (nth weekday of month, nth day of month, every nth weekday, nth workday, annual patterns). Pattern evaluation logic.
- Holiday and weekend handling per recurrence rule (ignore/skip/warn actions). Holiday region configuration at association level.
- Auto-create horizon: a background job that runs daily, evaluates enabled recurrence rules, and instantiates entries up to N days in advance.
- Manual instantiation: admin can apply a template group to a specific date on demand.

Considerations from architecture-learnings.md:
- The three-level template chain (RecurringShift → ShiftTemplateGroup → ShiftTemplate) from the predecessor app (Shiftings) is a good pattern worth following.
- Instantiation must be a factory that creates but doesn't persist — the caller handles persistence and idempotency checks.
- The background job needs to be reliable and observable (logging, error handling).

Output: detailed subtasks created under this parent task covering template CRUD, instantiation logic, recurrence patterns, holiday handling, background job, and e2e tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Design document or detailed plan covering all aspects of templates and recurrence
- [ ] #2 Subtasks created for: template group/template CRUD, instantiation logic, recurrence pattern evaluation, holiday/weekend handling, background job, admin UI, e2e tests
- [ ] #3 Each subtask has clear acceptance criteria
- [ ] #4 Dependencies between subtasks are documented
<!-- AC:END -->
