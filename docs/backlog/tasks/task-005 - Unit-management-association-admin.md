---
id: TASK-005
title: Unit management (association admin)
status: In Progress
assignee: []
created_date: '2026-03-16 14:31'
updated_date: '2026-03-22 14:09'
labels:
  - fullstack
milestone: m-1
dependencies:
  - TASK-004
priority: medium
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the association-admin UI for creating and managing units. Association admins (identified by the configured admin IdP group) need to be able to:

- List all units
- Create a new unit (name, slug, description, group bindings, admin group binding)
- Edit unit properties
- Delete a unit (with confirmation — cascades to calendars/entries)

This is an admin-only feature behind auth.RequireAdmin middleware. The UI should be simple forms — this is a management interface, not a public-facing page.

The existing codebase has handler patterns in internal/handler/ (error-returning handlers with Wrap()), template patterns in templates/, and the RequireAdmin middleware concept in internal/auth/. Follow these patterns.

Routes should be under /admin/units/.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Association admin can create a new unit with name, slug, and group bindings
- [x] #2 Association admin can edit unit properties
- [x] #3 Association admin can delete a unit with confirmation
- [x] #4 Non-admin users cannot access unit management
- [x] #5 Group bindings are stored in the unit_group_bindings table
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
# TASK-005 Implementation Plan: Unit Management (Association Admin)

## 1. Context Summary

### Architecture
The application is a Go + HTMX server-rendered web app using:
- **Go stdlib `net/http`** for routing (ServeMux with method+pattern matching)
- **HTMX** for partial page updates (with DaisyUI v5 / Tailwind CSS v4 for styling)
- **PostgreSQL + pgx** for persistence, with **sqlc** for generated query code
- **Goose** for migrations (already applied)
- **Go `html/template`** for server-side rendering

### Existing Patterns
- **Error-returning handlers**: `AppHandler func(w, r) error` wrapped via `h.Wrap()` in `internal/handler/handler.go`. Custom error types: `NotFoundError`, `ForbiddenError`, `ValidationError`.
- **Handler struct**: `Handler{Store, Renderer, Config}` in `internal/handler/handler.go` -- all handlers are receiver methods on this struct.
- **Middleware chains**: `base` (logging + recovery + auth) and `withCSRF` (base + CSRF). Defined in `cmd/server/main.go`. Role-restricted routes add a `RequireAdmin` middleware on top of `withCSRF`.
- **Auth**: `auth.Middleware()` in `internal/auth/auth.go` extracts user from reverse-proxy headers, upserts to DB, stores `RequestUser{ID, Email, IsAdmin, Groups}` in context. `auth.UserFromContext()` retrieves it.
- **Store pattern**: Package-level functions accepting `db.DBTX` for simple queries. Receiver methods on `*Store` for transactional operations using `s.WithTx()`. See `internal/store/calendar.go` `CreateCalendarWithViewers` for the transaction pattern with group bindings.
- **View models**: Typed structs in `internal/viewmodel/` -- `LayoutData{Title, CSRFToken, UserEmail, IsAdmin}` for layout, page-level structs compose `Layout LayoutData` plus page-specific fields.
- **Templates**: `templates/layouts/base.html` (defines `"layout"` block), `templates/components/*.html` (one `define` block each), `templates/pages/*.html` (define `"content"` block). The Renderer keys templates by filename (without extension) from `templates/pages/`.
- **CSRF**: Token from cookie, validated via `X-CSRF-Token` header. HTMX auto-injects it via `htmx:configRequest` event listener in base layout.

### What Already Exists (TASK-004)
- **DB schema**: `units` table (id, name, slug UNIQUE, description, logo_path, contact_email, admin_group, timestamps) + `unit_group_bindings` table (unit_id FK CASCADE, group_name, PK on both).
- **sqlc queries**: `units.sql` has `CreateUnit`, `UpdateUnit` (does NOT update slug), `DeleteUnit`, `GetUnitByID`, `GetUnitBySlug`, `ListUnits`. `unit_group_bindings.sql` has `InsertUnitGroupBinding`, `DeleteUnitGroupBindings`, `GetUnitGroupBindings`, plus membership queries.
- **Generated Go code**: `db.CreateUnitParams`, `db.UpdateUnitParams`, `db.InsertUnitGroupBindingParams`, etc. in `internal/db/`.
- **Store methods**: `ListUnits`, `GetUnitByID`, `GetUnitBySlug`, `ListUnitsByUserGroups`, `IsUnitMember`, `IsUnitAdmin` in `internal/store/unit.go`.

### What Does NOT Exist Yet
- **`auth.RequireAdmin` middleware**: Referenced in the SKILL.md pattern docs but not implemented. Must be created.
- **Admin handlers and routes**: No handler file for unit CRUD.
- **Admin templates**: No admin page templates exist.
- **Store methods for creating/updating units with group bindings**: The sqlc queries exist, but no transactional store-level `CreateUnitWithBindings` or `UpdateUnitWithBindings` method exists (analogous to `CreateCalendarWithViewers`).

## 2. File Changes

### New Files
| File | Purpose |
|------|---------|
| `internal/auth/require_admin.go` | `RequireAdmin` middleware function |
| `internal/handler/admin_units.go` | Handler methods for unit CRUD (list, create, edit, delete) |
| `internal/viewmodel/admin_units.go` | View model structs for admin unit pages |
| `templates/pages/admin_units.html` | Unit list page |
| `templates/pages/admin_unit_form.html` | Create/edit unit form page (shared template for both create and edit) |
| `templates/components/admin_unit_row.html` | Unit row component for the list (supports HTMX deletion) |
| `templates/components/admin_group_binding_input.html` | Reusable component for the dynamic group binding input field row |

### Modified Files
| File | Change |
|------|--------|
| `cmd/server/main.go` | Add admin unit routes under `/admin/units/` |
| `internal/store/unit.go` | Add `CreateUnitWithBindings` and `UpdateUnitWithBindings` receiver methods (transactional) |

## 3. Routes and Handlers

### Route Registration (in `cmd/server/main.go`)

```
GET  /admin/units/             -> withCSRF(auth.RequireAdmin(h.Wrap(h.AdminListUnits)))
GET  /admin/units/new          -> withCSRF(auth.RequireAdmin(h.Wrap(h.AdminNewUnit)))
POST /admin/units/             -> withCSRF(auth.RequireAdmin(h.Wrap(h.AdminCreateUnit)))
GET  /admin/units/{id}/edit    -> withCSRF(auth.RequireAdmin(h.Wrap(h.AdminEditUnit)))
POST /admin/units/{id}         -> withCSRF(auth.RequireAdmin(h.Wrap(h.AdminUpdateUnit)))
DELETE /admin/units/{id}       -> withCSRF(auth.RequireAdmin(h.Wrap(h.AdminDeleteUnit)))
```

Note: Using `DELETE /admin/units/{id}` with `hx-delete` for the delete action. HTMX sends the CSRF header automatically via the `htmx:configRequest` listener.

### Handler Signatures (in `internal/handler/admin_units.go`)

```go
func (h *Handler) AdminListUnits(w http.ResponseWriter, r *http.Request) error
func (h *Handler) AdminNewUnit(w http.ResponseWriter, r *http.Request) error
func (h *Handler) AdminCreateUnit(w http.ResponseWriter, r *http.Request) error
func (h *Handler) AdminEditUnit(w http.ResponseWriter, r *http.Request) error
func (h *Handler) AdminUpdateUnit(w http.ResponseWriter, r *http.Request) error
func (h *Handler) AdminDeleteUnit(w http.ResponseWriter, r *http.Request) error
```

### Handler Logic

**AdminListUnits**: Call `store.ListUnits()`, for each unit call `q.GetUnitGroupBindings()` to get bindings. Build `AdminUnitsPage` view model. Render `admin_units` page.

**AdminNewUnit**: Render the `admin_unit_form` page with an empty form (no unit data, `IsNew: true`).

**AdminCreateUnit**: Parse form (`r.ParseForm()`), validate, call `s.CreateUnitWithBindings()`, redirect to list with success toast (via `HX-Redirect` header for HTMX or `http.Redirect` for full-page).

**AdminEditUnit**: Parse `{id}` from path, load unit + bindings, render `admin_unit_form` page with existing data (`IsNew: false`).

**AdminUpdateUnit**: Parse `{id}` and form, validate, call `s.UpdateUnitWithBindings()`, redirect to list.

**AdminDeleteUnit**: Parse `{id}`, call `store.DeleteUnit()` (cascading FK handles bindings, calendars, entries). For HTMX requests: return empty body + success toast via OOB swap. For full-page: redirect to list.

## 4. Templates

### Template Naming
Since the renderer uses `filepath.Glob("pages/*.html")` and keys by filename (no subdirectories), use flat filenames with an `admin_` prefix:
- `templates/pages/admin_units.html` -> key `"admin_units"`
- `templates/pages/admin_unit_form.html` -> key `"admin_unit_form"`

### `templates/pages/admin_units.html`
- Defines `"content"` block
- Page title "Unit Management"
- "Create Unit" button linking to `/admin/units/new`
- Table of units with columns: Name, Slug, Admin Group, Group Bindings (count), Actions (Edit, Delete)
- Each row uses the `admin_unit_row` component
- Delete button: `hx-delete="/admin/units/{id}" hx-confirm="Delete unit '{name}'? This will cascade-delete all calendars and entries." hx-target="closest tr" hx-swap="outerHTML swap:0.3s"`
- Note from CLAUDE.md: OOB swaps cannot replace `<tr>` elements. Instead, use `hx-target` on the delete button pointing to a wrapping `<div>` instead of a `<tr>`, or use a div-based list layout rather than a semantic `<table>`. The plan will use a div-based list (card or flex layout) to avoid the HTMX OOB table-row limitation.

### `templates/pages/admin_unit_form.html`
- Defines `"content"` block
- Shared for create and edit: uses `{{ if .IsNew }}` for conditional title/action
- Form fields:
  - Name (text input, required)
  - Slug (text input, required, only on create -- immutable after creation)
  - Description (textarea, optional)
  - Contact Email (email input, optional)
  - Admin Group (text input, optional -- the IdP group that gets admin rights for this unit)
  - Group Bindings (dynamic list: each is a text input with a remove button; "Add Group" button appends a new input)
- Form action: POST to `/admin/units/` (create) or POST to `/admin/units/{id}` (edit)
- HTMX not strictly needed for the form submit itself -- use standard form submission with redirect. But the dynamic group binding inputs require a small amount of inline JS or HTMX to add/remove rows.

### Group Binding Dynamic Inputs
Since group bindings are a variable-length list, use a simple JavaScript pattern:
- Render existing bindings as `<input name="group_bindings" value="...">` elements inside a container `<div id="group-bindings">`.
- "Add Group" button: vanilla JS that clones a hidden template input and appends it.
- Each input has a "Remove" button that removes its parent element.
- On form submit, all `group_bindings` values are sent as repeated form fields, parsed via `r.Form["group_bindings"]`.

No HTMX round-trip needed for add/remove -- this is purely client-side DOM manipulation (simpler and faster).

### `templates/components/admin_unit_row.html` (optional)
- Defines `"admin-unit-row"` block
- Used in the list page, renders one unit card/row.
- Contains edit link and delete button with hx-delete + hx-confirm.

## 5. View Models

### `internal/viewmodel/admin_units.go`

```go
// AdminUnitListItem holds data for one row in the unit list.
type AdminUnitListItem struct {
    ID            int64
    Name          string
    Slug          string
    Description   string
    AdminGroup    string
    ContactEmail  string
    GroupBindings []string
}

// AdminUnitsPage is the page-level struct for the unit list.
type AdminUnitsPage struct {
    Layout LayoutData
    Units  []AdminUnitListItem
}

// AdminUnitFormPage is the page-level struct for the create/edit form.
type AdminUnitFormPage struct {
    Layout        LayoutData
    IsNew         bool
    Unit          AdminUnitFormData
    Errors        map[string]string  // field-name -> error message
}

// AdminUnitFormData holds the form field values (for both create and edit).
type AdminUnitFormData struct {
    ID            int64
    Name          string
    Slug          string
    Description   string
    ContactEmail  string
    AdminGroup    string
    GroupBindings []string
}
```

## 6. Store/Query Changes

### No New sqlc Queries Needed
All required sqlc queries already exist:
- `CreateUnit` (units.sql)
- `UpdateUnit` (units.sql) -- note: does NOT update slug (immutable by design)
- `DeleteUnit` (units.sql)
- `GetUnitByID`, `GetUnitBySlug`, `ListUnits` (units.sql)
- `InsertUnitGroupBinding`, `DeleteUnitGroupBindings`, `GetUnitGroupBindings` (unit_group_bindings.sql)

### New Store Methods (in `internal/store/unit.go`)
Two new receiver methods on `*Store` for transactional unit+binding operations:

**`CreateUnitWithBindings`**: Runs in a transaction via `s.WithTx()`. Creates the unit, then inserts each group binding. Returns the created `db.Unit`. Follows the same pattern as `CreateCalendarWithViewers`.

```go
func (s *Store) CreateUnitWithBindings(
    ctx context.Context,
    params db.CreateUnitParams,
    groupBindings []string,
) (db.Unit, error)
```

**`UpdateUnitWithBindings`**: Runs in a transaction. Updates the unit, deletes all existing bindings, re-inserts the new set. Returns updated `db.Unit`. Follows the same pattern as `UpdateCalendarWithViewers`.

```go
func (s *Store) UpdateUnitWithBindings(
    ctx context.Context,
    params db.UpdateUnitParams,
    groupBindings []string,
) (db.Unit, error)
```

## 7. Form Handling

### Create Form
- POST to `/admin/units/`
- Fields: `name`, `slug`, `description`, `contact_email`, `admin_group`, `group_bindings` (repeated)
- Parse with `r.ParseForm()`, read values with `r.FormValue("name")`, `r.Form["group_bindings"]`
- On validation error: re-render the form with the submitted values and error messages (no redirect)
- On success: `HX-Redirect` to `/admin/units/` (for HTMX) or `http.Redirect` (for non-HTMX)

### Edit Form
- POST to `/admin/units/{id}`
- Fields: same as create except `slug` is not editable (displayed as read-only text or disabled input)
- Same parse/validate/re-render-on-error pattern

### Group Bindings in HTML
- Multiple `<input type="text" name="group_bindings">` elements
- "Add" button uses JS: `document.getElementById('group-bindings').insertAdjacentHTML('beforeend', '<div>...<input name="group_bindings">...<button onclick="this.parentElement.remove()">Remove</button></div>')`
- Empty inputs are filtered out server-side (skip blank strings from `r.Form["group_bindings"]`)

### Slug Auto-Generation
- Optionally add a small JS snippet on the create form that auto-generates slug from name (lowercased, spaces to hyphens, non-alphanum stripped)
- The user can still manually edit the slug field

## 8. Validation

### Required Fields
- `name`: non-empty, trimmed
- `slug`: non-empty, trimmed, lowercase, alphanumeric + hyphens only (regex: `^[a-z0-9]+(-[a-z0-9]+)*$`), only on create

### Uniqueness Checks
- `slug`: Before creating, call `GetUnitBySlug()`. If it returns without error, the slug is taken -> validation error. Use `errors.Is(err, pgx.ErrNoRows)` to confirm it's available.

### Optional Fields
- `description`, `contact_email`, `admin_group`: can be empty (stored as pgtype.Text with Valid=false)
- `group_bindings`: can be empty (unit with no group bindings has no members)

### Validation Implementation
Validation logic lives in the handler. Collect errors into a `map[string]string`. If any errors, re-render the form with the errors map and submitted values. Return `nil` (the form is rendered, no error to propagate to `Wrap()`).

## 9. Edge Cases

### Deletion Confirmation and Cascading
- The `hx-confirm` attribute shows a browser dialog: "Delete unit '{name}'? This will permanently delete all calendars and entries belonging to this unit."
- Database cascading: `unit_group_bindings` has `ON DELETE CASCADE` on `unit_id` FK. Calendars reference `unit_id` with CASCADE (confirmed in migration 00003). This means deleting a unit cascades to calendars, and deleting calendars cascades to entries, attendances, etc.
- After successful deletion: for HTMX requests, return a toast (OOB) + empty response so `hx-swap="delete"` removes the row from the DOM. For full-page: redirect to list.

### Slug Collisions
- Check slug uniqueness before INSERT. On collision, return a validation error: "A unit with this slug already exists."
- Since slug is not updatable in UpdateUnit (by design), no collision check is needed on edit.

### Empty Group Bindings
- A unit with no group bindings is valid (it just has no members via group mapping). The admin can still manage it.
- Filter out empty strings from the group_bindings form field before storing.

### Duplicate Group Bindings
- Filter out duplicate group names from the submitted list before inserting (deduplicate server-side).
- The PK constraint on (unit_id, group_name) also prevents duplicates at the DB level.

### HTMX Table Row Limitation
- Per CLAUDE.md: "The oob content replace functionality in HTMX cannot replace table rows or semantic HTML objects." Use a `div`-based card/list layout for the units list, NOT a `<table>` with `<tr>` elements. This ensures HTMX swaps work correctly when deleting units inline.

## 10. Testing Strategy

### Handler Tests
- Test each handler method using `httptest.NewRequest` + `httptest.NewRecorder`.
- Mock the store using pgxmock (matching the pattern in `internal/store/unit_test.go`).
- Test cases:
  - **AdminListUnits**: returns page with units listed
  - **AdminCreateUnit**: valid input creates unit + bindings; missing name returns validation error; duplicate slug returns validation error
  - **AdminUpdateUnit**: valid input updates unit; missing name returns validation error; unit not found returns 404
  - **AdminDeleteUnit**: deletes unit and returns success; unit not found returns 404
  - **RequireAdmin**: non-admin gets 403; admin passes through

### Store Method Tests
- `CreateUnitWithBindings`: verify transaction calls CreateUnit + InsertUnitGroupBinding for each binding
- `UpdateUnitWithBindings`: verify transaction calls UpdateUnit + DeleteUnitGroupBindings + InsertUnitGroupBinding for each binding
- Use pgxmock to set expectations on BEGIN/COMMIT/ROLLBACK.

### E2E Tests (Playwright -- separate effort)
- Navigate to `/admin/units/`, create a unit, verify it appears in the list
- Edit the unit, verify changes are saved
- Delete the unit with confirmation, verify it disappears
- Attempt to access `/admin/units/` without admin privileges, verify 403

## 11. Open Questions

1. **Logo upload**: The `units` table has a `logo_path` column, but the task description doesn't mention logo upload. Should the form include a logo upload field, or defer to a later task? **Recommendation**: Omit logo upload from this task. Add a text input for `logo_path` if needed, or leave it out entirely and add it later.

2. **Slug immutability**: The existing `UpdateUnit` sqlc query does NOT include slug in the SET clause, confirming slug is immutable after creation. The edit form should display the slug as read-only. Confirmed by query analysis.

3. **Admin group field semantics**: The `admin_group` field on the unit is the IdP group whose members become unit admins (distinct from association admin). The form should have a tooltip or help text explaining this.

4. **Flash messages / success toasts**: After a successful create/update, how should the success message be shown on the redirected list page? Options: (a) Use a query parameter like `?msg=created` and render a toast on the list page. (b) Use HTMX `hx-redirect` which does a full page load -- no in-flight toast possible. **Recommendation**: Use `HX-Redirect` header for HTMX requests (this triggers a full-page navigation via HTMX, loading the list page fresh). Success confirmation is implicit (the unit appears in the list). Alternatively, use `HX-Trigger` response header with a custom event that the list page listens for. Simplest approach: redirect and let the list speak for itself.

5. **Nav link for admin**: Should the nav component (`templates/components/nav.html`) include an "Admin" link visible only to admins? **Recommendation**: Yes, add a conditional admin link in the nav. This is a minor change to nav.html using `{{ if .IsAdmin }}`.

## Implementation Order

1. `internal/auth/require_admin.go` -- RequireAdmin middleware
2. `internal/store/unit.go` -- Add `CreateUnitWithBindings` and `UpdateUnitWithBindings`
3. `internal/viewmodel/admin_units.go` -- View model structs
4. `internal/handler/admin_units.go` -- All 6 handler methods
5. `templates/pages/admin_units.html` -- List page
6. `templates/pages/admin_unit_form.html` -- Create/edit form page
7. `templates/components/admin_unit_row.html` -- Unit row component (optional, can be inline in page)
8. `cmd/server/main.go` -- Register routes
9. `templates/components/nav.html` -- Add admin link (optional, minor)
10. Tests for handlers and store methods
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Implementation Notes (2026-03-22)

All acceptance criteria implemented:
- RequireAdmin middleware in `internal/auth/require_admin.go`
- CreateUnitWithBindings / UpdateUnitWithBindings in `internal/store/unit.go`
- View models in `internal/viewmodel/admin_units.go`
- 6 handler methods in `internal/handler/admin_units.go`
- List page template `templates/pages/admin_units.html` (div-based layout per CLAUDE.md)
- Form page template `templates/pages/admin_unit_form.html` (shared create/edit)
- Routes registered in `cmd/server/main.go` under `/admin/units/`
- Tests: 4 auth tests, 6 store tests, 22 handler utility tests
- `go build ./...` and `go vet ./...` pass clean
- All 90+ tests pass
<!-- SECTION:NOTES:END -->
