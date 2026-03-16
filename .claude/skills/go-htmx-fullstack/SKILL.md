---
name: go-htmx-fullstack
description: >
  Build lightweight, server-rendered Go web applications with HTMX for interactivity,
  DaisyUI/Tailwind for styling, PostgreSQL for persistence, and Playwright for e2e testing.
  Use this skill when scaffolding or building Go web apps that need partial page updates,
  server-side rendering, HTMX-driven UI, or a Node-less frontend design system.
  Also use when the user asks about HTMX patterns, Go handler architectures,
  OOB swaps, CSRF with HTMX, or e2e-first testing strategies.
---

# Go + HTMX Full-Stack Architecture

This skill captures the architectural decisions, patterns, and pitfalls for building lightweight, server-rendered Go web applications. The stack replaces SPA complexity with HTMX-driven partial updates, uses DaisyUI/Tailwind without a Node.js runtime, and prioritizes e2e tests over unit tests.

## Stack Overview

| Layer | Technology | Role |
|-------|-----------|------|
| Server | Go stdlib `net/http` | Routing, HTTP handling |
| Database | PostgreSQL + pgx | Persistence, connection pooling |
| Migrations | Goose (embedded) | Schema management, auto-applied at startup |
| Templates | Go `html/template` | Server-side rendering |
| Interactivity | HTMX | Partial page updates, no SPA framework |
| Styling | Tailwind CSS v4 + DaisyUI v5 | Utility-first CSS with component library |
| Testing | Playwright | E2E tests against real infrastructure |
| Auth | oauth2-proxy | External auth via forwarded headers |
| Containerization | Docker multi-stage | Minimal production images |

## Project Layout

```
cmd/server/         # Entrypoint, routing, server setup
internal/
  handler/          # HTTP handlers (return errors, not responses)
  store/            # Database queries (accept DBTX interface)
  render/           # Template rendering engine
  auth/             # Authentication middleware
  middleware/       # Middleware chain (logging, recovery, CSRF)
  model/            # Domain types
  viewmodel/        # Typed view model structs (template "props")
  config/           # Environment-based configuration
migrations/         # Goose SQL migrations (embedded via go:embed)
templates/
  layouts/          # Base HTML wrappers (full page skeleton)
  pages/            # Top-level content (one per route, defines "content" block)
  components/       # Reusable UI components (one define block each, typed struct input)
  fragments/        # HTMX response wrappers (render a component without layout)
static/css/         # Tailwind input + generated output
e2e/                # Playwright tests (isolated package.json)
test/               # Test infrastructure (seed data, Dockerfiles, auth config)
```

---

## 1. Error-Returning Handlers

Handlers return `error` instead of writing HTTP responses directly. A central `Wrap()` adapter maps errors to appropriate HTTP responses. This separates business logic from response formatting.

### The Pattern

```go
// AppHandler is the handler signature — returns error instead of writing responses
type AppHandler func(w http.ResponseWriter, r *http.Request) error

// Handler holds shared dependencies
type Handler struct {
    Store    *store.Store
    Renderer *render.Renderer
    Config   *config.Config
}

// Wrap converts AppHandler to http.HandlerFunc
func (h *Handler) Wrap(fn AppHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := fn(w, r); err != nil {
            h.handleError(w, r, err)
        }
    }
}
```

### Custom Error Types

Define semantic error types that map to HTTP status codes:

```go
type NotFoundError struct{ Message string }
type ForbiddenError struct{ Message string }
type ValidationError struct{ Message string }

func (e *NotFoundError) Error() string   { return e.Message }
func (e *ForbiddenError) Error() string  { return e.Message }
func (e *ValidationError) Error() string { return e.Message }
```

### Central Error Handler

The error handler distinguishes HTMX requests from full-page requests. HTMX errors render as toast notifications; full-page errors render an error page.

```go
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
    var notFound *NotFoundError
    var forbidden *ForbiddenError
    var validation *ValidationError

    code := http.StatusInternalServerError
    msg := "Internal Server Error"

    switch {
    case errors.As(err, &notFound):
        code = http.StatusNotFound
        msg = notFound.Message
    case errors.As(err, &forbidden):
        code = http.StatusForbidden
        msg = forbidden.Message
    case errors.As(err, &validation):
        code = http.StatusBadRequest
        msg = validation.Message
    default:
        slog.Error("unhandled error", "error", err)
    }

    if r.Header.Get("HX-Request") == "true" {
        w.WriteHeader(code)
        h.Renderer.Component(w, "toast", viewmodel.Toast{
            Type: "error", Message: msg,
        })
        return
    }
    h.RenderErrorPage(w, r, code, msg)
}
```

### Handler Example

Handlers focus purely on business logic. Errors bubble up naturally:

```go
func (h *Handler) CalendarDay(w http.ResponseWriter, r *http.Request) error {
    user := auth.UserFromContext(r.Context())
    calID := r.PathValue("id")
    dateStr := r.PathValue("date")

    day, err := time.Parse("2006-01-02", dateStr)
    if err != nil {
        return &ValidationError{Message: "Invalid date"}
    }

    entries, err := h.Store.ListEntriesForDay(r.Context(), calID, day)
    if err != nil {
        return fmt.Errorf("calendar day: list entries: %w", err)
    }

    data := h.buildCalendarDayPage(r.Context(), user, day, entries)
    h.Renderer.Page(w, r, "calendar_day", data)
    return nil
}
```

### Route Registration

Routes use `Wrap()` and middleware chains:

```go
mux.Handle("GET /calendar/{id}/day/{date}", withCSRF(h.Wrap(h.CalendarDay)))
mux.Handle("POST /entries/{id}/join", withCSRF(h.Wrap(h.JoinEntry)))
mux.Handle("GET /fragments/shift-card/{id}", withCSRF(h.Wrap(h.ShiftCardFragment)))
mux.Handle("POST /admin/units/{id}/edit", withCSRF(auth.RequireAdmin(h.Wrap(h.EditUnit))))
```

---

## 2. DBTX Interface for Store Methods

Store methods accept a `DBTX` interface satisfied by both the connection pool and transactions. This means the same query function works in both contexts without knowing which one it's in.

```go
// DBTX is the common interface for pool and transaction
type DBTX interface {
    Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct {
    pool *pgxpool.Pool
}

// DB returns the pool for non-transactional queries
func (s *Store) DB() DBTX { return s.pool }

// WithTx wraps a function in a transaction with automatic rollback on error/panic
func (s *Store) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback(ctx)
            panic(p)
        }
    }()
    if err := fn(tx); err != nil {
        _ = tx.Rollback(ctx)
        return err
    }
    return tx.Commit(ctx)
}
```

### Store Method Example

Store methods are package-level functions, not receiver methods. They accept `DBTX` as a parameter:

```go
func GetEntryByID(ctx context.Context, db DBTX, id int64) (*model.Entry, error) {
    var e model.Entry
    err := db.QueryRow(ctx,
        `SELECT id, calendar_id, name, entry_type, start_at, end_at,
                location, required_participants, max_participants
         FROM entries WHERE id = $1`, id,
    ).Scan(&e.ID, &e.CalendarID, &e.Name, &e.EntryType, &e.StartAt, &e.EndAt,
        &e.Location, &e.RequiredParticipants, &e.MaxParticipants)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrNotFound
    }
    return &e, err
}
```

### Transaction Usage in Handlers

```go
func (h *Handler) JoinEntry(w http.ResponseWriter, r *http.Request) error {
    entryID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
    user := auth.UserFromContext(r.Context())

    err := h.Store.WithTx(r.Context(), func(tx pgx.Tx) error {
        entry, err := store.GetEntryForUpdate(r.Context(), tx, entryID)
        if err != nil { return err }

        count, err := store.CountAttendees(r.Context(), tx, entryID)
        if err != nil { return err }

        if entry.MaxParticipants > 0 && count >= entry.MaxParticipants {
            return &ValidationError{Message: "Shift is full"}
        }

        return store.CreateAttendance(r.Context(), tx, entryID, user.ID, model.StatusAccepted)
    })
    if err != nil { return err }

    // Re-render just the shift card component
    data := h.buildShiftCardData(r.Context(), entryID, user)
    h.Renderer.Component(w, "shift-card", data)
    return nil
}
```

---

## 3. Middleware Chain

Middleware composes right-to-left: the first argument is the outermost (executes first).

```go
type Middleware func(http.Handler) http.Handler

func Chain(mws ...Middleware) Middleware {
    return func(next http.Handler) http.Handler {
        for i := len(mws) - 1; i >= 0; i-- {
            next = mws[i](next)
        }
        return next
    }
}
```

### Two Standard Chains

```go
// Base: logging → recovery → auth
base := middleware.Chain(
    middleware.Logging(),
    middleware.Recovery(),
    auth.Middleware(store, adminGroup),
)

// WithCSRF: base + CSRF validation on mutations
withCSRF := middleware.Chain(base, middleware.CSRF(csrfSecret))
```

GET-only routes can use `base` (no CSRF needed). All mutation routes use `withCSRF`. Role-restricted routes add `RequireAdmin` or similar on top.

### CSRF Middleware

Uses HMAC-signed cookies. The token is stored in context for templates to access:

```go
func CSRF(secret []byte) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := getOrCreateSignedToken(w, r, secret)
            ctx := context.WithValue(r.Context(), csrfContextKey{}, token)

            if isMutatingMethod(r.Method) {
                if r.Header.Get("X-CSRF-Token") != token {
                    http.Error(w, "Forbidden: CSRF mismatch", http.StatusForbidden)
                    return
                }
            }

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

## 4. Component-Oriented Template Architecture

The template system implements component-oriented design without a JavaScript framework. A "component" is a Go template that accepts a single typed struct (its props), renders one piece of UI, and can be re-rendered independently via HTMX. This gives the same architectural benefits as React components — isolation, composability, typed interfaces — with zero client-side complexity.

### Three Rules

**Rule 1: Every template receives a typed struct, never a raw map.** The struct is the component's interface contract, equivalent to React props. All conditional rendering logic is precomputed as boolean fields in the struct — templates never evaluate complex permission or state logic.

**Rule 2: Components are templates that define one block, accept one struct, and live in `templates/components/`.** A component can compose other components. Pages compose components. Pages are never composed by other pages or components.

**Rule 3: HTMX fragment endpoints re-render individual components in place.** When user interaction changes state, the server re-renders just the affected component and HTMX swaps it into the DOM. This is the equivalent of React's state-triggered re-render.

### View Model Structs

View model structs live in `internal/viewmodel/` and define the data contract for every template. There are two levels: page-level structs that compose component-level structs.

```go
// internal/viewmodel/entry.go

// ShiftCard is the "props" for the shift-card component.
// All display logic is precomputed — the template only checks booleans.
type ShiftCard struct {
    ID                int64
    Name              string
    Start             time.Time
    End               time.Time
    Location          string
    CalendarName      string
    CalendarColor     string
    FilledSlots       int
    RequiredSlots     int
    MaxSlots          int
    UserIsParticipant bool
    CanJoin           bool      // precomputed: !isPast && !isFull && !isParticipant && hasPermission
    CanLeave          bool      // precomputed: isParticipant && !isPast
    CanEdit           bool      // precomputed: isAdmin || (isUnitMember && calendar.entryCreation == unit_members)
    ShowSubstitute    bool      // precomputed: isParticipant && isPast-deadline && !isPast
    StaffingStatus    string    // precomputed: "understaffed" | "minimum-met" | "full"
}

// MeetingCard is the "props" for the meeting-card component.
type MeetingCard struct {
    ID              int64
    Name            string
    Start           time.Time
    End             time.Time
    Location        string
    CalendarName    string
    AcceptedCount   int
    DeclinedCount   int
    PendingCount    int
    TotalAudience   int
    UserStatus      string    // "accepted" | "declined" | "pending" | "" (not in audience)
    CanRespond      bool
    CanEdit         bool
    ResponseDeadline *time.Time
    DeadlinePassed  bool
}

// CalendarDayPage is the page-level struct composing multiple components.
type CalendarDayPage struct {
    Layout    LayoutData
    Day       time.Time
    Shifts    []ShiftCard
    Meetings  []MeetingCard
    CanCreate bool
    CalendarID int64
}
```

### Builder Methods

Handlers use builder methods to construct view model structs from domain models. This is where permission checks, time comparisons, and business logic execute — keeping templates pure.

```go
// internal/handler/viewmodel_builders.go

func (h *Handler) buildShiftCardData(ctx context.Context, entryID int64, user *auth.RequestUser) viewmodel.ShiftCard {
    entry, _ := h.Store.GetEntryByID(ctx, entryID)
    count, _ := h.Store.CountAttendees(ctx, entryID)
    isParticipant, _ := h.Store.IsAttendee(ctx, entryID, user.ID)
    calendar, _ := h.Store.GetCalendarByID(ctx, entry.CalendarID)
    isMember := h.isUnitMember(ctx, user, calendar.UnitID)
    isAdmin := h.isUnitAdmin(ctx, user, calendar.UnitID)
    isPast := entry.EndAt.Before(time.Now())
    isFull := entry.MaxParticipants > 0 && count >= entry.MaxParticipants

    return viewmodel.ShiftCard{
        ID:                entry.ID,
        Name:              entry.Name,
        Start:             entry.StartAt,
        End:               entry.EndAt,
        Location:          entry.Location,
        FilledSlots:       count,
        RequiredSlots:     entry.RequiredParticipants,
        MaxSlots:          entry.MaxParticipants,
        UserIsParticipant: isParticipant,
        CanJoin:           !isPast && !isFull && !isParticipant && (isMember || isAdmin),
        CanLeave:          isParticipant && !isPast,
        CanEdit:           isAdmin,
        ShowSubstitute:    isParticipant && !isPast && entry.ResponseDeadline != nil && time.Now().After(*entry.ResponseDeadline),
        StaffingStatus:    computeStaffingStatus(count, entry.RequiredParticipants, entry.MaxParticipants),
    }
}
```

### Template File Conventions

**Components** — one file, one `{{ define }}` block, one struct input:

```html
<!-- templates/components/shift_card.html -->
{{ define "shift-card" }}
<div id="shift-{{ .ID }}" class="card bg-base-200 shadow-md">
  <div class="card-body">
    <h3 class="card-title">{{ .Name }}</h3>
    <p class="text-sm opacity-70">{{ formatTime .Start }} – {{ formatTime .End }}</p>
    {{ if .Location }}<p class="text-sm">{{ .Location }}</p>{{ end }}

    {{ template "staffing-bar" . }}

    <div class="card-actions justify-end">
      {{ if .CanJoin }}
        <button class="btn btn-primary btn-sm"
                hx-post="/entries/{{ .ID }}/join"
                hx-target="#shift-{{ .ID }}"
                hx-swap="outerHTML">
          Join
        </button>
      {{ end }}
      {{ if .CanLeave }}
        <button class="btn btn-ghost btn-sm"
                hx-post="/entries/{{ .ID }}/leave"
                hx-target="#shift-{{ .ID }}"
                hx-swap="outerHTML">
          Leave
        </button>
      {{ end }}
      {{ if .ShowSubstitute }}
        <button class="btn btn-warning btn-sm"
                hx-post="/entries/{{ .ID }}/substitute"
                hx-target="#shift-{{ .ID }}"
                hx-swap="outerHTML">
          Find substitute
        </button>
      {{ end }}
    </div>
  </div>
</div>
{{ end }}
```

```html
<!-- templates/components/staffing_bar.html -->
{{ define "staffing-bar" }}
<div class="flex items-center gap-2">
  <progress class="progress {{ if eq .StaffingStatus "understaffed" }}progress-error
    {{ else if eq .StaffingStatus "minimum-met" }}progress-warning
    {{ else }}progress-success{{ end }}"
    value="{{ .FilledSlots }}"
    max="{{ if .MaxSlots }}{{ .MaxSlots }}{{ else }}{{ .RequiredSlots }}{{ end }}">
  </progress>
  <span class="text-sm font-mono">{{ .FilledSlots }}/{{ .RequiredSlots }}</span>
</div>
{{ end }}
```

**Pages** — extend layout, compose components:

```html
<!-- templates/pages/calendar_day.html -->
{{ define "content" }}
<div id="main-content" class="container mx-auto p-4">
  <h1 class="text-2xl font-bold mb-4">{{ formatDate .Day }}</h1>

  {{ if .Shifts }}
    <h2 class="text-lg font-semibold mb-2">Shifts</h2>
    <div class="grid gap-4 md:grid-cols-2">
      {{ range .Shifts }}
        {{ template "shift-card" . }}
      {{ end }}
    </div>
  {{ end }}

  {{ if .Meetings }}
    <h2 class="text-lg font-semibold mt-6 mb-2">Meetings</h2>
    <div class="grid gap-4 md:grid-cols-2">
      {{ range .Meetings }}
        {{ template "meeting-card" . }}
      {{ end }}
    </div>
  {{ end }}
</div>
{{ end }}
```

**Layout** — wraps page content:

```html
<!-- templates/layouts/base.html -->
{{ define "layout" }}
<!DOCTYPE html>
<html lang="en" data-theme="light-theme">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ .Layout.CSRFToken }}">
    <title>{{ .Layout.Title }} — Convoke</title>
    <link rel="stylesheet" href="{{ cssFile }}">
    <script src="/static/js/htmx.min.js"></script>
</head>
<body class="min-h-screen bg-base-100">
    {{ template "nav" .Layout }}
    <main>{{ template "content" . }}</main>
    <div id="modal" style="display:none"></div>
    <div id="toast-zone" class="toast toast-end toast-top z-50"></div>
    <script>
    document.body.addEventListener('htmx:configRequest', function(event) {
        var token = document.querySelector('meta[name="csrf-token"]').getAttribute('content');
        event.detail.headers['X-CSRF-Token'] = token;
    });
    </script>
</body>
</html>
{{ end }}
```

### Renderer: Dual Render Paths

The renderer exposes two methods. `Page()` renders a full page (layout + content + components) for normal requests or just the content block for HTMX navigations. `Component()` renders a single named component template for HTMX partial updates.

```go
type Renderer struct {
    templates map[string]*template.Template
    devMode   bool
    templateDir string
}

// Page renders a full page. HTMX requests get only the content block;
// normal requests get the full layout wrapping the content.
func (r *Renderer) Page(w http.ResponseWriter, req *http.Request, name string, data any) {
    t := r.getTemplate(name)
    if req.Header.Get("HX-Request") == "true" {
        t.ExecuteTemplate(w, "content", data)
    } else {
        t.ExecuteTemplate(w, "layout", data)
    }
}

// Component renders a single named template block. Used for HTMX fragment
// responses that swap one component in place (e.g., re-rendering a shift card
// after the user joins).
func (r *Renderer) Component(w http.ResponseWriter, name string, data any) {
    t := r.getAnyTemplate()
    t.ExecuteTemplate(w, name, data)
}

// ComponentOOB renders a component with the hx-swap-oob attribute for
// out-of-band updates (updating parts of the page outside the hx-target).
func (r *Renderer) ComponentOOB(w http.ResponseWriter, name string, data any) {
    // Wraps the component output with OOB swap attribute
    t := r.getAnyTemplate()
    t.ExecuteTemplate(w, name+"-oob", data)
}
```

### Fragment Endpoints

Every interactive component has a corresponding fragment endpoint that re-renders it. The handler builds the view model and renders just the component.

```go
// Full page route — renders layout + page + all components
mux.Handle("GET /calendar/{id}/day/{date}", withCSRF(h.Wrap(h.CalendarDay)))

// Fragment route — renders only the shift card component for HTMX swap
mux.Handle("GET /fragments/shift-card/{id}", withCSRF(h.Wrap(h.ShiftCardFragment)))

// Mutation routes — perform action, then re-render the affected component
mux.Handle("POST /entries/{id}/join", withCSRF(h.Wrap(h.JoinEntry)))
mux.Handle("POST /entries/{id}/leave", withCSRF(h.Wrap(h.LeaveEntry)))
mux.Handle("POST /entries/{id}/rsvp", withCSRF(h.Wrap(h.RSVPEntry)))
```

```go
// Fragment handler: fetch fresh data, render just the component
func (h *Handler) ShiftCardFragment(w http.ResponseWriter, r *http.Request) error {
    entryID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
    user := auth.UserFromContext(r.Context())
    data := h.buildShiftCardData(r.Context(), entryID, user)
    h.Renderer.Component(w, "shift-card", data)
    return nil
}
```

### Dev Mode Hot-Reload

When dev mode is enabled, templates are re-parsed on every request. No build step or restart needed during development.

### Template Helper Functions

Register helpers in a `FuncMap` for formatting, arithmetic, and data passing:

```go
funcMap := template.FuncMap{
    "formatTime":     formatTime,      // time.Time → "15:04"
    "formatDate":     formatDate,      // time.Time → "Mon, 02 Jan 2006"
    "formatDateTime": formatDateTime,  // time.Time → "02.01.2006 15:04"
    "timeUntil":      timeUntil,       // time.Time → "in 3 hours"
    "cssFile":        func() string { return cssPath },
    "add":            addInt,
    "sub":            subInt,
}
```

**IMPORTANT: Never use the `map` helper to pass ad-hoc data to components.** If a component needs data, define it in the component's struct. Ad-hoc maps circumvent the typed interface contract and make refactoring impossible. The only exception is passing a single extra value to a sub-component that is also used standalone — prefer restructuring the view model over using `map`.

---

## 5. HTMX Patterns and Pitfalls

### Global CSRF Injection

Inject the CSRF token into every HTMX request automatically via `htmx:configRequest`:

```html
<meta name="csrf-token" content="{{ .Layout.CSRFToken }}">
<script>
    document.body.addEventListener('htmx:configRequest', function(event) {
        var token = document.querySelector('meta[name="csrf-token"]').getAttribute('content');
        event.detail.headers['X-CSRF-Token'] = token;
    });
</script>
```

This eliminates the need to add `hx-headers` to every form element.

### Component Re-Render Pattern

The standard interaction cycle for a component:

1. User clicks a button inside a component (e.g., "Join" on a shift card)
2. HTMX sends a POST to the mutation endpoint (e.g., `/entries/5/join`)
3. The handler performs the action and re-renders the component with fresh data
4. HTMX swaps the response into the DOM, replacing the old component

```html
<!-- Inside shift-card component: button targets itself -->
<button hx-post="/entries/{{ .ID }}/join"
        hx-target="#shift-{{ .ID }}"
        hx-swap="outerHTML">
  Join
</button>
```

The mutation handler re-renders the component:

```go
func (h *Handler) JoinEntry(w http.ResponseWriter, r *http.Request) error {
    // ... perform join ...
    data := h.buildShiftCardData(r.Context(), entryID, user)
    h.Renderer.Component(w, "shift-card", data)
    return nil
}
```

The user sees the card update in place — slot count increments, "Join" button disappears, "Leave" button appears. Same UX as a React state update, zero client-side JavaScript.

### OOB (Out-of-Band) Swaps

OOB swaps update parts of the page outside the main `hx-target`. Use when a mutation affects multiple components (e.g., joining a shift should also update a header counter).

```go
// Primary response: updated shift card
h.Renderer.Component(w, "shift-card", shiftData)
// OOB: also update the staffing summary in the sidebar
h.Renderer.ComponentOOB(w, "staffing-summary", summaryData)
```

The OOB template includes the swap attribute conditionally:

```html
{{ define "staffing-summary-oob" }}
<div id="staffing-summary" hx-swap-oob="true">
  {{ template "staffing-summary" . }}
</div>
{{ end }}
```

### PITFALL: OOB Swaps on Table Rows

**Browsers strip `<tr>` tags when parsed inside a `<div>`, which HTMX uses internally for OOB processing.** Never use `hx-swap-oob` directly on `<tr>` elements.

**Workaround:** Use `hx-target` + `hx-swap="outerHTML"` on the triggering element instead:

```html
<!-- WRONG: OOB on <tr> — will silently fail in browsers -->
<tr id="row-5" hx-swap-oob="outerHTML">...</tr>

<!-- CORRECT: Target the row from the trigger -->
<button hx-post="/entries/5/toggle"
        hx-target="#row-5"
        hx-swap="outerHTML">
    Toggle
</button>
```

### Modal Pattern

1. Create an empty modal container in the layout:
   ```html
   <div id="modal" style="display:none"></div>
   ```

2. Load modal content via HTMX:
   ```html
   <button hx-get="/entries/5/edit" hx-target="#modal">Edit</button>
   ```

3. The modal template shows itself with an IIFE:
   ```html
   <div class="modal modal-open"
        onclick="if(event.target===this){this.parentElement.style.display='none';this.remove();}">
       <div class="modal-box">
           <!-- form content -->
           <button hx-post="/entries/5"
                   hx-target="#shift-5"
                   hx-swap="outerHTML">
               Save
           </button>
       </div>
   </div>
   <script>
   (function() {
       document.getElementById('modal').style.display = '';
   })();
   </script>
   ```

4. Close on success using `hx-on::after-request`:
   ```html
   hx-on::after-request="if(event.detail.successful){
       var m=document.getElementById('modal');
       m.innerHTML='';
       m.style.display='none';
   }"
   ```

5. Handle errors in modals by intercepting `htmx:beforeSwap`:
   ```javascript
   btn.addEventListener('htmx:beforeSwap', function(evt) {
       if (evt.detail.xhr.status >= 400) {
           evt.detail.shouldSwap = false;
       }
   });
   ```

### Toast Notifications

Toasts use OOB `beforeend` swap to append to a container:

```html
{{ define "toast" }}
<div hx-swap-oob="beforeend:#toast-zone">
    <div class="alert alert-{{ .Type }} shadow-lg toast-enter">
        <span>{{ .Message }}</span>
    </div>
    <script>
    (function() {
        var el = document.currentScript.previousElementSibling;
        setTimeout(function() {
            el.classList.add('toast-exit');
            el.addEventListener('animationend', function() { el.remove(); });
        }, 3000);
        document.currentScript.remove();
    })();
    </script>
</div>
{{ end }}
```

### URL State with Navigation

Use `hx-push-url="true"` on navigation links so the URL updates and back/forward work:

```html
<a hx-get="/calendar/3/day/2026-03-17"
   hx-target="#main-content"
   hx-push-url="true"
   class="btn btn-ghost">
  Next day →
</a>
```

### Confirmation Before Destructive Actions

Load a confirmation modal instead of performing the action directly:

```html
<button hx-get="/entries/{{ .ID }}/confirm-delete"
        hx-target="#modal">
  Delete
</button>
```

### JavaScript Philosophy

No frontend framework. No Alpine.js. Only vanilla JavaScript for:
- Theme switching (localStorage + `data-theme` attribute)
- Modal show/hide (IIFE scripts in template fragments)
- Toast auto-dismiss (setTimeout + animation)
- CSRF token injection (single global event listener)

All JS is inline in templates, scoped with IIFEs to avoid global pollution.

---

## 6. Node-less Design System

### Philosophy

Node.js is a build-time dependency only. It downloads the Tailwind CLI and DaisyUI package for CSS generation. No Node.js runtime, no bundler, no npm scripts for the application.

### Root package.json

Contains only CSS build tooling:

```json
{
  "devDependencies": {
    "@tailwindcss/cli": "^4.0.0",
    "daisyui": "^5.0.0",
    "tailwindcss": "^4.0.0"
  }
}
```

No `scripts`, no `description`, no `license`, no application dependencies. Build commands go in the Makefile:

```makefile
css:
    npx @tailwindcss/cli -i static/css/input.css -o static/css/styles.css --minify
```

### Tailwind CSS v4 Configuration

Tailwind v4 uses CSS-based configuration instead of `tailwind.config.js`:

```css
/* static/css/input.css */
@import "tailwindcss";

@plugin "daisyui/theme" {
  name: "light-theme";
  default: true;
  color-scheme: "light";
  /* OKLch color definitions */
}

@plugin "daisyui/theme" {
  name: "dark-theme";
  prefersdark: true;
  color-scheme: "dark";
}

@plugin "daisyui" {
  themes: light-theme --default, dark-theme --prefersdark;
}

@source "../../templates/**/*.html";
```

The `@source` directive tells Tailwind where to scan for class names (PurgeCSS).

### Theme Switching

Light/dark theme via `data-theme` attribute on `<html>`, persisted in `localStorage`:

```javascript
(function() {
    var theme = localStorage.getItem('theme') ||
        (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark-theme' : 'light-theme');
    document.documentElement.setAttribute('data-theme', theme);
})();

function toggleTheme() {
    var current = document.documentElement.getAttribute('data-theme');
    var next = current === 'dark-theme' ? 'light-theme' : 'dark-theme';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);
}
```

### Docker Multi-Stage Build for CSS

CSS is built in a dedicated Docker stage so the final image has no Node.js:

```dockerfile
# Stage 1: Build CSS
FROM node:lts-alpine AS css
WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci
COPY static/css/input.css static/css/
COPY templates/ templates/
RUN npx @tailwindcss/cli -i static/css/input.css -o static/css/styles.css --minify

# Stage 2: Build Go binary
FROM golang:alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=css /build/static/css/styles.css static/css/
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o app ./cmd/server

# Stage 3: Minimal runtime
FROM alpine:latest
COPY --from=builder /build/app /app
COPY --from=builder /build/static /static
COPY --from=builder /build/templates /templates
COPY --from=builder /build/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/app"]
```

### CSS Versioning for Cache Busting

Hash the CSS file at startup and use it in the filename. Serve with immutable cache headers:

```go
// At startup
hash := sha256sum(cssFile)[:8]
cssPath := fmt.Sprintf("/static/css/styles.%s.css", hash)

// Template helper
funcMap["cssFile"] = func() string { return cssPath }

// In layout template
<link rel="stylesheet" href="{{ cssFile }}">
```

---

## 7. Authentication via Forwarded Headers

Authentication is handled externally by oauth2-proxy. The Go app never sees passwords or tokens — it trusts forwarded headers.

### Header Extraction

```go
func AuthMiddleware(store *store.Store, adminGroup string) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            email := r.Header.Get("X-Forwarded-Email")
            if email == "" {
                if r.Header.Get("HX-Request") == "true" {
                    w.Header().Set("HX-Redirect", "/")
                    w.WriteHeader(http.StatusUnauthorized)
                    return
                }
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            groups := parseGroups(r.Header.Get("X-Forwarded-Groups"))
            isAdmin := contains(groups, adminGroup)

            // Auto-create user on first login, sync groups on subsequent logins
            user, err := store.GetOrCreateUser(ctx, email, groups, isAdmin)
            if err != nil { ... }

            ctx := contextWithUser(r.Context(), &RequestUser{
                ID: user.ID, Email: email, IsAdmin: isAdmin, Groups: groups,
            })
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Context-Based User Access

```go
type RequestUser struct {
    ID      int64
    Email   string
    IsAdmin bool
    Groups  []string   // IdP groups, used to resolve unit membership
}

func UserFromContext(ctx context.Context) *RequestUser {
    u, _ := ctx.Value(contextKey{}).(*RequestUser)
    return u
}

// In any handler:
user := auth.UserFromContext(r.Context())
```

### Unit Membership Resolution

Unit membership is resolved from the user's IdP groups, not from a membership table:

```go
// Check if user is a member of a unit by comparing their IdP groups
// against the unit's configured group bindings
func (h *Handler) isUnitMember(ctx context.Context, user *auth.RequestUser, unitID int64) bool {
    unit, _ := h.Store.GetUnitByID(ctx, unitID)
    for _, binding := range unit.GroupBindings {
        for _, userGroup := range user.Groups {
            if binding == userGroup {
                return true
            }
        }
    }
    return false
}

func (h *Handler) isUnitAdmin(ctx context.Context, user *auth.RequestUser, unitID int64) bool {
    if user.IsAdmin { return true }
    unit, _ := h.Store.GetUnitByID(ctx, unitID)
    if unit.AdminGroup == "" { return false }
    return slices.Contains(user.Groups, unit.AdminGroup)
}
```

### HTMX-Aware Unauthorized Responses

When an HTMX request hits an auth boundary, use `HX-Redirect` header instead of a normal redirect — HTMX intercepts this and performs a full-page navigation.

---

## 8. Embedded Migrations

Migrations are SQL files embedded into the binary and auto-applied at startup. No separate migration tool or deployment step needed.

```go
// migrations/embed.go
//go:embed *.sql
var FS embed.FS

// In main.go
func runMigrations(databaseURL string) error {
    db, err := sql.Open("pgx", databaseURL)
    if err != nil { return err }
    defer db.Close()

    goose.SetBaseFS(migrations.FS)
    goose.SetDialect("postgres")
    return goose.Up(db, ".")
}
```

Goose migration format:

```sql
-- +goose Up
CREATE TABLE units (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    slug       TEXT UNIQUE NOT NULL,
    admin_group TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE unit_group_bindings (
    unit_id    BIGINT REFERENCES units(id) ON DELETE CASCADE,
    group_name TEXT NOT NULL,
    PRIMARY KEY (unit_id, group_name)
);

CREATE TABLE calendars (
    id                     BIGSERIAL PRIMARY KEY,
    unit_id                BIGINT REFERENCES units(id) ON DELETE CASCADE,
    name                   TEXT NOT NULL,
    entry_creation         TEXT NOT NULL DEFAULT 'admins_only',
    visibility             TEXT NOT NULL DEFAULT 'unit',
    participation          TEXT NOT NULL DEFAULT 'viewers',
    participant_visibility TEXT NOT NULL DEFAULT 'everyone',
    color                  TEXT NOT NULL DEFAULT '#3b82f6',
    sort_order             INT NOT NULL DEFAULT 0,
    is_event               BOOLEAN NOT NULL DEFAULT FALSE,
    event_start            DATE,
    event_end              DATE,
    created_at             TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE entries (
    id                     BIGSERIAL PRIMARY KEY,
    calendar_id            BIGINT REFERENCES calendars(id) ON DELETE CASCADE,
    name                   TEXT NOT NULL,
    entry_type             TEXT NOT NULL CHECK (entry_type IN ('shift', 'meeting')),
    start_at               TIMESTAMPTZ NOT NULL,
    end_at                 TIMESTAMPTZ NOT NULL,
    location               TEXT,
    description            TEXT,
    required_participants  INT NOT NULL DEFAULT 0,
    max_participants       INT NOT NULL DEFAULT 0,
    response_deadline      TIMESTAMPTZ,
    recurrence_rule_id     BIGINT,
    created_at             TIMESTAMPTZ DEFAULT NOW(),
    modified_at            TIMESTAMPTZ DEFAULT NOW(),
    CHECK (start_at < end_at),
    CHECK (max_participants = 0 OR max_participants >= required_participants)
);

CREATE TABLE attendance (
    id           BIGSERIAL PRIMARY KEY,
    entry_id     BIGINT REFERENCES entries(id) ON DELETE CASCADE,
    user_id      BIGINT REFERENCES users(id) ON DELETE CASCADE,
    status       TEXT NOT NULL CHECK (status IN ('accepted', 'declined', 'pending', 'needs_substitute', 'replaced')),
    responded_at TIMESTAMPTZ,
    note         TEXT,
    UNIQUE (entry_id, user_id)
);

CREATE INDEX idx_entries_calendar_start ON entries (calendar_id, start_at);
CREATE INDEX idx_attendance_entry ON attendance (entry_id);
CREATE INDEX idx_attendance_user_status ON attendance (user_id, status);

-- +goose Down
DROP TABLE IF EXISTS attendance;
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS calendars;
DROP TABLE IF EXISTS unit_group_bindings;
DROP TABLE IF EXISTS units;
```

---

## 9. E2E-First Testing Strategy

### Philosophy

Prioritize end-to-end tests over unit tests. E2E tests exercise the full stack — real database, real auth flow, real browser interactions. They catch integration issues that unit tests miss and serve as living documentation of user-facing behavior.

### Test Infrastructure

The e2e stack runs in Docker Compose with:
- **App**: Go binary built with `-cover` flag for coverage instrumentation
- **PostgreSQL**: Test database with seed data
- **Identity Provider** (e.g., Keycloak): Real OAuth2/OIDC provider
- **Auth Proxy** (e.g., oauth2-proxy): Real header forwarding

### Playwright Configuration

```typescript
// e2e/playwright.config.ts
export default defineConfig({
    fullyParallel: false,     // Sequential: tests share DB state
    workers: 1,               // One at a time
    retries: 0,               // Fail fast
    projects: [
        {
            name: 'setup',
            testMatch: /.*\.setup\.ts/,
        },
        {
            name: 'member-tests',
            testMatch: /^(?!.*admin-).*\.spec\.ts$/,
            dependencies: ['setup'],
            use: { storageState: '.auth/member.json' },
        },
        {
            name: 'admin-tests',
            testMatch: /admin-.*\.spec\.ts$/,
            dependencies: ['setup'],
            use: { storageState: '.auth/admin.json' },
        },
    ],
});
```

### Auth Setup

The setup project performs real login flows against the identity provider. Browser storage state is saved to files and reused by test projects:

```typescript
// e2e/auth.setup.ts
test('authenticate as member', async ({ page }) => {
    await page.goto('/');
    await page.fill('#username', 'testmember@example.com');
    await page.fill('#password', 'testpass');
    await page.click('#login-submit');
    await page.waitForURL('**/calendar');
    await page.context().storageState({ path: '.auth/member.json' });
});
```

### Go Coverage from E2E

Build the app binary with `-cover`, set `GOCOVERDIR` in the container, and collect coverage after tests:

```dockerfile
# test/Dockerfile.cover
RUN CGO_ENABLED=0 go build -cover -o app ./cmd/server
```

```yaml
# docker-compose.test.yml
app:
    build:
        dockerfile: test/Dockerfile.cover
    environment:
        GOCOVERDIR: /coverage
    volumes:
        - ./coverage:/coverage
```

```makefile
e2e-coverage:
    go tool covdata textfmt -i=./coverage -o=coverage.out
    go tool cover -func=coverage.out
```

### Makefile Orchestration

```makefile
e2e: e2e-up
    cd e2e && npx playwright test; TEST_EXIT=$$?; \
    $(MAKE) e2e-coverage; \
    $(MAKE) e2e-down; \
    exit $$TEST_EXIT

e2e-up:
    mkdir -p coverage
    docker compose -f docker-compose.test.yml up -d --build --wait

e2e-down:
    docker compose -f docker-compose.test.yml down -v
    rm -rf coverage
```

---

## 10. Server Setup Pattern

### Initialization Sequence

```go
func main() {
    cfg := config.Load()                          // 1. Load env config
    dbpool := pgxpool.New(ctx, cfg.DatabaseURL)   // 2. Connect to DB
    runMigrations(cfg.DatabaseURL)                // 3. Auto-migrate
    s := store.New(dbpool)                        // 4. Create store
    rndr := render.New(cfg.TemplateDir, cfg.Dev)  // 5. Parse templates
    h := &handler.Handler{Store: s, Renderer: rndr, Config: cfg}

    // 6. Build middleware chains
    base := middleware.Chain(middleware.Logging(), middleware.Recovery(), auth.Middleware(s, cfg.AdminGroup))
    withCSRF := middleware.Chain(base, middleware.CSRF(csrfSecret))

    // 7. Register routes
    mux := http.NewServeMux()
    mux.Handle("GET /healthz", http.HandlerFunc(healthCheck))
    mux.Handle("GET /calendar/{id}/day/{date}", withCSRF(h.Wrap(h.CalendarDay)))
    mux.Handle("POST /entries/{id}/join", withCSRF(h.Wrap(h.JoinEntry)))
    mux.Handle("GET /fragments/shift-card/{id}", withCSRF(h.Wrap(h.ShiftCardFragment)))
    // ...

    // 8. Graceful shutdown
    srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux}
    go srv.ListenAndServe()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
```

### Environment Configuration

All config via environment variables with sensible defaults:

```go
type Config struct {
    DatabaseURL  string // required, no default
    ListenAddr   string // default ":8080"
    DevMode      bool   // default false
    AdminGroup   string // default "admin"
    TemplateDir  string // default "./templates"
    StaticDir    string // default "./static"
}
```

---

## Quick Reference: Common HTMX Attributes

| Attribute | Use Case | Example |
|-----------|----------|---------|
| `hx-get` | Load content | `hx-get="/entries/5/edit"` |
| `hx-post` | Submit data | `hx-post="/entries/5/join"` |
| `hx-target` | Where to put response | `hx-target="#shift-5"` |
| `hx-swap` | How to insert | `outerHTML`, `innerHTML`, `beforeend` |
| `hx-swap-oob` | Update elsewhere | `hx-swap-oob="true"` (on response element) |
| `hx-trigger` | What starts request | `click`, `change`, `submit` |
| `hx-vals` | Extra values | `hx-vals='js:{"qty": el.value}'` |
| `hx-push-url` | Update URL bar | `hx-push-url="true"` |
| `hx-on::after-request` | Post-request action | Close modal on success |

## Quick Reference: Component Checklist

When creating a new component:

1. **Define the view model struct** in `internal/viewmodel/` with all display data and precomputed boolean flags
2. **Write the builder method** in the handler package that constructs the struct from domain models
3. **Create the template** in `templates/components/` with `{{ define "component-name" }}`, using only fields from the struct
4. **Add a stable HTML ID** to the root element (`id="component-{{ .ID }}"`) so HTMX can target it
5. **Add a fragment endpoint** in the router if the component needs independent re-rendering
6. **Wire HTMX attributes** on interactive elements: `hx-post`, `hx-target="#component-{{ .ID }}"`, `hx-swap="outerHTML"`
