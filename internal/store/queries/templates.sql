-- name: GetTemplateGroupByID :one
SELECT * FROM template_groups WHERE id = $1;

-- name: ListTemplateGroupsByCalendar :many
SELECT * FROM template_groups WHERE calendar_id = $1 ORDER BY name;

-- name: CreateTemplateGroup :one
INSERT INTO template_groups (unit_id, calendar_id, name, base_start_time, location)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateTemplateGroup :one
UPDATE template_groups SET
    name = $2,
    base_start_time = $3,
    location = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTemplateGroup :exec
DELETE FROM template_groups WHERE id = $1;

-- name: ListTemplatesByGroup :many
SELECT * FROM templates WHERE template_group_id = $1 ORDER BY sort_order, name;

-- name: CreateTemplate :one
INSERT INTO templates (template_group_id, name, type, start_offset, duration, required_participants, max_participants, description, response_deadline_offset, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateTemplate :one
UPDATE templates SET
    name = $2,
    type = $3,
    start_offset = $4,
    duration = $5,
    required_participants = $6,
    max_participants = $7,
    description = $8,
    response_deadline_offset = $9,
    sort_order = $10,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTemplate :exec
DELETE FROM templates WHERE id = $1;

-- name: GetRecurrenceRuleByID :one
SELECT * FROM recurrence_rules WHERE id = $1;

-- name: ListRecurrenceRulesByTemplateGroup :many
SELECT * FROM recurrence_rules WHERE template_group_id = $1;

-- name: ListEnabledRecurrenceRules :many
SELECT * FROM recurrence_rules WHERE enabled = true;

-- name: CreateRecurrenceRule :one
INSERT INTO recurrence_rules (template_group_id, pattern_type, pattern_params, first_occurrence, auto_create_horizon, enabled, weekend_action, weekend_warning_text, holiday_action, holiday_warning_text)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateRecurrenceRule :one
UPDATE recurrence_rules SET
    pattern_type = $2,
    pattern_params = $3,
    first_occurrence = $4,
    auto_create_horizon = $5,
    enabled = $6,
    weekend_action = $7,
    weekend_warning_text = $8,
    holiday_action = $9,
    holiday_warning_text = $10,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateRecurrenceRuleLastEvaluated :exec
UPDATE recurrence_rules SET last_evaluated_at = now() WHERE id = $1;
