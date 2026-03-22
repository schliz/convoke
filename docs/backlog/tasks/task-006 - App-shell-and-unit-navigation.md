---
id: TASK-006
title: App shell and unit navigation
status: In Progress
assignee: []
created_date: '2026-03-16 14:31'
updated_date: '2026-03-22 13:58'
labels:
  - fullstack
milestone: m-1
dependencies:
  - TASK-004
priority: medium
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the top-level app shell: the navigation bar, unit listing/switcher, and overall page layout that all other pages will live inside.

The existing codebase has templates/layouts/base.html with a basic layout and templates/components/nav.html with a starter nav bar. These need to be extended to include:

- Navigation showing units the current user belongs to (resolved via their IdP groups)
- A way to switch between units (sidebar, dropdown, or nav links)
- Link to the personal dashboard (future, but reserve the nav slot)
- Admin link (visible only to association admins)
- The existing theme toggle should be preserved

The root route (/) currently redirects — it should redirect to the personal dashboard (or a unit listing if dashboard isn't built yet).

The nav component's view model (internal/viewmodel/layout.go LayoutData) needs to be extended with the user's units list and admin status.

Follow the component-oriented template architecture: typed view model structs, precomputed booleans for conditional rendering.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Navigation displays units the current user belongs to
- [ ] #2 User can navigate between different units
- [ ] #3 Admin link visible only to association admins
- [ ] #4 Theme toggle preserved from existing nav
- [ ] #5 Root route redirects to a sensible default page
- [ ] #6 LayoutData view model extended with units and admin status
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
## 1. Context Summary

### What exists today

**Layout:** `templates/layouts/base.html` defines the full page skeleton: HTML `<head>` (fonts, Material Symbols, Tailwind/DaisyUI CSS, HTMX, theme toggle script), `<body>` wrapping `{{ template "nav" .Layout }}` and `<main>{{ template "content" . }}</main>`, plus a toast zone and CSRF header injection. Theme persistence uses `localStorage` with `convoke-light`/`convoke-dark` data-theme toggling.

**Nav:** `templates/components/nav.html` is a minimal DaisyUI `navbar bg-base-200` with the Convoke brand link, a theme toggle button, and a user email display. It receives `LayoutData` (Title, CSRFToken, UserEmail, IsAdmin).

**View model:** `internal/viewmodel/layout.go` defines `LayoutData{Title, CSRFToken, UserEmail, IsAdmin}`. The `IsAdmin` field exists but is not yet used by the nav template.

**Auth:** `internal/auth/auth.go` extracts `RequestUser{ID, Email, IsAdmin, Groups}` from proxy headers and stores it in context. `UserFromContext(ctx)` retrieves it. Groups are synced to `user_idp_groups` on each request.

**Store:** `internal/store/unit.go` provides `ListUnitsByUserGroups(ctx, dbtx, groups)` returning `[]db.Unit` (ID, Name, Slug, Description, LogoPath, ContactEmail, AdminGroup, timestamps). Also provides `IsUnitMember`, `IsUnitAdmin`, `GetUnitBySlug`.

**Handler:** `internal/handler/handler.go` defines `Handler{Store, Renderer, Config}` with `Wrap(AppHandler)` converting error-returning handlers to `http.HandlerFunc`. Only `HealthCheck` exists as a handler.

**Router:** `cmd/server/main.go` uses Go stdlib `http.NewServeMux()`. Routes: `GET /healthz`, `GET /static/`, CSS cache-bust route, and `GET /` redirecting to `/healthz`. Middleware chains: `base` (logging, recovery, auth), `withCSRF` (base + CSRF). The root redirect is explicitly marked as temporary.

**CSS:** Tailwind v4 + DaisyUI v5 via `static/css/input.css` with custom `convoke-light`/`convoke-dark` themes. Built by Node.js toolchain.

**Pages:** Only `templates/pages/health.html` exists (defines a "content" block).

**Renderer:** `internal/render/render.go` parses all layouts + components as base files, then parses each page file on top. `Page()` renders either `layout` (full page) or `content` (HTMX partial). `Component()` renders a named block.

### Design references

The design mockups show two distinct navigation patterns:
- **Personal dashboard** (`docs/design/design/personal_dashboard/`): Mobile-first with a sticky header (brand + notifications + avatar) and a bottom tab bar (Home, Schedule, Invites, Profile).
- **Unit dashboard** (`docs/design/design/unit_dashboard_bar_committee/`): Shows the unit name/icon in the header with a bottom tab bar (Dashboard, Calendar, Tasks, Unit Settings).

Both mockups use a mobile-centric layout with a bottom navigation bar. The design uses the orange primary color (#ec5b13), Public Sans font, and Material Symbols icons.

### What this task must build

The app shell: a top navbar + responsive navigation that shows the user's units, lets them switch between units, reserves a dashboard link, shows an admin link for admins, and preserves the theme toggle. The root `/` route must redirect sensibly.

---

## 2. File Changes Overview

### Files to modify
- `internal/viewmodel/layout.go` -- extend LayoutData with units and admin status
- `templates/components/nav.html` -- rebuild with unit navigation, admin link, responsive behavior
- `templates/layouts/base.html` -- wrap content in a drawer layout for mobile nav
- `cmd/server/main.go` -- add root redirect handler, new page routes
- `internal/handler/handler.go` -- add `NewLayoutData` helper method

### Files to create
- `internal/handler/home.go` -- handler for the root redirect (or a simple home/unit-list page)
- `templates/pages/home.html` -- simple landing page showing unit list (serves as default until personal dashboard is built)

---

## 3. Layout Architecture

### Chosen pattern: Top navbar + mobile drawer sidebar

The app uses a **DaisyUI navbar** at the top with a **drawer** for mobile navigation. This matches the existing nav pattern and DaisyUI's responsive drawer component.

**Desktop (lg+):**
- Horizontal navbar with: brand logo/name (left), unit dropdown menu (center-left), spacer, dashboard link, admin link (if admin), theme toggle, user email/avatar (right).
- No sidebar -- units are accessible via a dropdown in the navbar.

**Mobile (<lg):**
- Compact navbar with: hamburger button (left), brand (center), theme toggle + avatar (right).
- Hamburger opens a drawer sidebar overlay containing: unit list (links), dashboard link, admin link (if admin), user email.

### Why not a bottom tab bar (as in mockups)?

The design mockups show mobile bottom tabs, but those are page-specific (personal dashboard tabs vs unit dashboard tabs). The app shell's nav is the *global* navigation (switching between units and top-level sections), which is distinct from page-specific tabs. Bottom tabs will be added per-page when those features are built (TASK for personal dashboard, TASK for unit dashboard). The global navigation is best served by the top navbar + drawer pattern, which is standard for multi-scope apps.

---

## 4. Navigation Component Design

### Desktop layout (lg+)

```
[hamburger(hidden)] [Convoke logo] [Units dropdown ▾] ---- [Dashboard] [Admin*] [🌙] [user@email]
```

### Mobile layout (<lg)

```
[☰] ------------- [Convoke] ------------- [🌙] [avatar]
```

Drawer sidebar (on hamburger click):
```
┌──────────────────────┐
│ Convoke              │
│                      │
│ My Units             │
│ ├─ Bar Committee     │
│ ├─ Board             │
│ └─ Kitchen Crew      │
│                      │
│ ─────────────────    │
│ 📊 Dashboard         │
│ ⚙️ Admin*            │
│                      │
│ user@example.com     │
└──────────────────────┘
```

### Unit list in nav

Units are displayed as links to `/units/{slug}`. On desktop, they appear in a dropdown menu triggered by a "Units" button. On mobile, they appear as a vertical list in the drawer sidebar.

If the user has no units, the unit area shows a short "No units assigned" message.

### Admin link

Rendered only when `{{if .IsAdmin}}`. Links to `/admin` (placeholder for now -- returns 404 is fine, the link slot is reserved).

### Dashboard link

Links to `/dashboard` (placeholder). Always visible.

### Theme toggle

Preserved as-is: a button calling `toggleTheme()`. Uses Material Symbols `dark_mode` icon. Works the same on desktop and mobile.

---

## 5. View Model Changes

### Extended LayoutData

```go
package viewmodel

import "github.com/schliz/convoke/internal/db"

// NavUnit is a lightweight projection of db.Unit for nav rendering.
type NavUnit struct {
    Name string
    Slug string
}

type LayoutData struct {
    Title     string
    CSRFToken string
    UserEmail string
    IsAdmin   bool
    Units     []NavUnit  // units the user belongs to (via IdP groups)
    HasUnits  bool       // precomputed: len(Units) > 0
}
```

**Key decisions:**
- `NavUnit` is a minimal struct (Name, Slug only) rather than passing full `db.Unit`. The nav does not need description, logo, admin_group, timestamps, etc.
- `HasUnits` is a precomputed boolean following the "typed view model with precomputed booleans" pattern from the architecture skill. Templates check `{{if .HasUnits}}` instead of evaluating `{{if .Units}}`.
- `IsAdmin` already exists but is not currently populated. It will be populated alongside `Units`.

---

## 6. Handler Changes

### LayoutData builder helper

Add a `NewLayoutData` method on `Handler` (or a package-level helper in `handler.go`) that constructs `LayoutData` from the request context. Every page handler will call this instead of manually assembling LayoutData.

```go
// In internal/handler/handler.go

func (h *Handler) NewLayoutData(r *http.Request, title string) viewmodel.LayoutData {
    user := auth.UserFromContext(r.Context())
    csrfToken := middleware.TokenFromContext(r.Context())

    ld := viewmodel.LayoutData{
        Title:     title,
        CSRFToken: csrfToken,
        UserEmail: "",
    }

    if user == nil {
        return ld
    }

    ld.UserEmail = user.Email
    ld.IsAdmin = user.IsAdmin

    // Load units for nav
    units, err := store.ListUnitsByUserGroups(r.Context(), h.Store.DB(), user.Groups)
    if err != nil {
        slog.Error("layout: failed to list units", "error", err, "user_id", user.ID)
        // Non-fatal: render nav without units
        return ld
    }

    ld.Units = make([]viewmodel.NavUnit, len(units))
    for i, u := range units {
        ld.Units[i] = viewmodel.NavUnit{Name: u.Name, Slug: u.Slug}
    }
    ld.HasUnits = len(ld.Units) > 0

    return ld
}
```

**Performance consideration:** `ListUnitsByUserGroups` runs a JOIN query on every page load. For a volunteer app with <100 users and <20 units, this is fine. If it becomes a concern, the result can be cached in a session or request-scoped cache later. Do NOT prematurely optimize.

### Home handler

A new `home.go` handler that checks user state and redirects or renders:

```go
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) error {
    user := auth.UserFromContext(r.Context())
    if user == nil {
        http.Redirect(w, r, "/healthz", http.StatusTemporaryRedirect)
        return nil
    }

    units, err := store.ListUnitsByUserGroups(r.Context(), h.Store.DB(), user.Groups)
    if err != nil {
        return fmt.Errorf("home: list units: %w", err)
    }

    // If user has exactly one unit, redirect to it
    if len(units) == 1 {
        http.Redirect(w, r, "/units/"+units[0].Slug, http.StatusTemporaryRedirect)
        return nil
    }

    // Otherwise, render a simple unit listing page
    data := struct {
        Layout viewmodel.LayoutData
        // no additional fields needed for the basic home page
    }{
        Layout: h.NewLayoutData(r, "Home"),
    }
    h.Renderer.Page(w, r, "home", data)
    return nil
}
```

---

## 7. Route Changes

### In cmd/server/main.go

Replace the temporary root redirect:
```go
// Before:
mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { ... })

// After:
mux.Handle("GET /", base(h.Wrap(h.Home)))
```

The root route now goes through auth middleware (since it needs to know the user's units), and renders or redirects based on user state.

The `GET /` handler in the stdlib mux acts as a catch-all for unmatched paths (since "/" matches everything). The handler must check `r.URL.Path != "/"` and return 404 for non-root paths, matching the current behavior.

### Future route reservations (NOT implemented now, just mentioned for context)
- `GET /dashboard` -- personal dashboard (future task)
- `GET /units/{slug}` -- unit dashboard (future task)
- `GET /admin` -- admin area (future task)

These routes do NOT need to be registered in this task. The nav links will point to them, and they will 404 until implemented. The exception: if `GET /units/{slug}` is needed for unit navigation to actually work, a minimal placeholder handler should be added that renders a "coming soon" page with the correct layout.

**Decision: Add a minimal unit placeholder.** The nav links will be broken links without it, which would be a poor experience even in development. Add a minimal handler:

```go
func (h *Handler) UnitDashboard(w http.ResponseWriter, r *http.Request) error {
    slug := r.PathValue("slug")
    // Validate the unit exists
    unit, err := store.GetUnitBySlug(r.Context(), h.Store.DB(), slug)
    if err != nil {
        return &NotFoundError{Message: "Unit not found"}
    }
    data := struct {
        Layout   viewmodel.LayoutData
        UnitName string
        UnitSlug string
    }{
        Layout:   h.NewLayoutData(r, unit.Name),
        UnitName: unit.Name,
        UnitSlug: unit.Slug,
    }
    h.Renderer.Page(w, r, "unit_placeholder", data)
    return nil
}
```

Register: `mux.Handle("GET /units/{slug}", base(h.Wrap(h.UnitDashboard)))`

Also add a placeholder template `templates/pages/unit_placeholder.html`.

---

## 8. Responsive Design

### Breakpoint strategy

Follow DaisyUI/Tailwind's default breakpoints:
- `<lg` (< 1024px): Mobile/tablet -- hamburger menu, drawer sidebar
- `lg+` (>= 1024px): Desktop -- full horizontal navbar with dropdown

### Mobile drawer implementation

Use DaisyUI's `drawer` component with a checkbox toggle:

```html
<div class="drawer">
    <input id="nav-drawer" type="checkbox" class="drawer-toggle" />
    <div class="drawer-content flex flex-col">
        <!-- Navbar (always visible) -->
        <div class="navbar bg-base-200 shadow-sm">
            <div class="flex-none lg:hidden">
                <label for="nav-drawer" class="btn btn-ghost btn-square">
                    <span class="material-symbols-outlined">menu</span>
                </label>
            </div>
            <!-- ... rest of navbar -->
        </div>
        <!-- Page content -->
        <main>{{ template "content" . }}</main>
    </div>
    <div class="drawer-side z-50">
        <label for="nav-drawer" class="drawer-overlay"></label>
        <div class="menu bg-base-200 min-h-full w-72 p-4">
            <!-- Sidebar content: units, links -->
        </div>
    </div>
</div>
```

This means `base.html` needs to be restructured to wrap the content area inside the drawer structure. The nav template handles both the top bar and the sidebar content.

### Template restructuring

`base.html` will contain the drawer wrapper. `nav.html` will be split into:
- The top navbar section (inside `drawer-content`)
- The sidebar section (inside `drawer-side`)

Both are part of the same `{{ define "nav" }}` template to keep them co-located, since they share the same data (LayoutData). The drawer checkbox links them.

---

## 9. HTMX Considerations

### Nav is static per page load -- no partial updates needed

The navigation reflects the user's unit memberships, which change only when their IdP groups change (i.e., on login/session refresh). There is no scenario where the nav needs to update via HTMX during a page interaction.

The renderer's `Page()` method already handles the HTMX partial case: when `HX-Request` is set, it renders only the `content` block, skipping the layout (and therefore the nav). This means HTMX page navigations will keep the existing nav intact and only swap the content area. This is the correct behavior.

### One consideration: active unit highlighting

When navigating between units via HTMX (hx-get, hx-push-url), the nav's active unit indicator would not update because the nav is not re-rendered. Two options:
1. **Do nothing for now.** Full page navigation (no hx-boost) for unit switching. Simple, correct.
2. **Add hx-boost="true" later** and use `hx-swap-oob` to update the active indicator. This is a future optimization.

**Decision:** Use standard `<a href>` links for unit navigation (full page loads). No HTMX for unit switching in this task.

---

## 10. Edge Cases

### User with no units
- Nav shows "No units" or simply omits the units section.
- Home page renders a message: "You are not a member of any units yet. Contact your organization's admin."
- The root route does NOT redirect (no unit to redirect to).

### User with exactly one unit
- Root route redirects to `/units/{slug}` directly (skip the unit listing page).
- Nav still shows the unit in the dropdown/sidebar for consistency (user should know which unit they are viewing).

### New user before groups sync
- On first login, auth middleware upserts the user and syncs groups. `ListUnitsByUserGroups` runs after auth, so groups are current.
- If a user has groups but no matching unit_group_bindings exist, they will see no units. This is correct -- the admin needs to configure unit group bindings.

### Admin-only users (admin but no unit membership)
- Admin link is visible. Units section shows "No units."
- Root route renders the home page (unit listing, which is empty). Admin can navigate to `/admin`.

### Dev mode
- Fake admin user with `Groups: []string{adminGroup}` is injected. `ListUnitsByUserGroups` will return units bound to the admin group (if any exist).
- If no units exist in the database, the nav will show no units. This is expected for a fresh dev setup.

### Very long unit names
- Use Tailwind `truncate` on unit names in the nav. Max width for the dropdown items.

---

## 11. Testing Strategy

### E2E tests (Playwright) -- primary testing approach

Per the project's architecture skill, Playwright e2e tests are the primary testing strategy. Unit tests are secondary.

**Test scenarios:**
1. **Nav renders units:** Log in as a user with group bindings to 2+ units. Verify the nav dropdown/sidebar lists all units.
2. **Unit navigation:** Click a unit link. Verify redirect to `/units/{slug}`. Verify page renders with correct unit name.
3. **Admin link visibility:** Log in as admin. Verify admin link is visible. Log in as non-admin. Verify admin link is absent.
4. **Theme toggle:** Click theme toggle. Verify `data-theme` attribute changes. Reload. Verify theme persists.
5. **Root redirect (single unit):** User with exactly one unit. Navigate to `/`. Verify redirect to `/units/{slug}`.
6. **Root landing (multiple units):** User with 2+ units. Navigate to `/`. Verify home page shows unit list.
7. **No units:** User with no group bindings. Navigate to `/`. Verify home page with "no units" message.
8. **Mobile drawer:** Set viewport to mobile size. Verify hamburger is visible. Click hamburger. Verify drawer opens with unit links.

### Unit tests

- `viewmodel.NavUnit` construction from `db.Unit` -- trivial, likely not worth a dedicated test.
- `NewLayoutData` -- test that it correctly populates Units and HasUnits from store data. Use pgxmock.
- `Home` handler -- test redirect logic (0 units -> render page, 1 unit -> redirect, 2+ units -> render page).

---

## 12. Open Questions

1. **Active unit indicator:** Should the nav visually indicate which unit is currently being viewed? If so, `LayoutData` needs an `ActiveUnitSlug` field, and the nav template needs to compare it against each unit's slug. **Recommendation:** Yes, add `ActiveUnitSlug string` to `LayoutData`. Handlers set it when rendering unit-scoped pages. The nav highlights the matching unit. Cost is minimal (one extra string field).

2. **Unit logo in nav:** The `db.Unit` model has `LogoPath`. Should unit logos appear next to unit names in the nav? **Recommendation:** Not in this task. Logos add visual complexity and require image handling (fallback for missing logos, sizing). Keep the nav text-only for now.

3. **Notification bell:** The design mockups show a notification bell with a badge. This is out of scope for this task but the nav should leave space for it (the rightmost section of the navbar). **Recommendation:** Reserve the spot but do not implement.

4. **User display name vs email:** The auth middleware stores `Email` but the user record also has `DisplayName`. Should the nav show the display name instead? **Recommendation:** Show email for now (it is reliably present). Display name can be swapped in later by extending `RequestUser` or querying the user record.

5. **Should the root route require authentication?** Currently `/` redirects without auth. With the new home handler going through the `base` middleware chain (which includes auth), unauthenticated users will get a 401. This is correct -- Convoke has no public pages. The reverse proxy (oauth2-proxy) handles the login redirect before requests reach Convoke.
<!-- SECTION:PLAN:END -->
