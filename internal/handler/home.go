package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/schliz/convoke/internal/auth"
	"github.com/schliz/convoke/internal/middleware"
	"github.com/schliz/convoke/internal/store"
	"github.com/schliz/convoke/internal/viewmodel"
)

// NewLayoutData constructs a LayoutData from the request context.
// Every page handler should call this instead of manually assembling LayoutData.
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

	// Load units for nav.
	units, err := store.ListUnitsByUserGroups(r.Context(), h.Store.DB(), user.Groups)
	if err != nil {
		slog.Error("layout: failed to list units", "error", err, "user_id", user.ID)
		// Non-fatal: render nav without units.
		return ld
	}

	ld.Units = make([]viewmodel.NavUnit, len(units))
	for i, u := range units {
		ld.Units[i] = viewmodel.NavUnit{Name: u.Name, Slug: u.Slug}
	}
	ld.HasUnits = len(ld.Units) > 0

	return ld
}

// Home handles the root route. It redirects users with exactly one unit to
// that unit's page, and renders a unit listing for users with zero or multiple
// units.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) error {
	// The stdlib mux uses "/" as a catch-all; reject non-root paths.
	if r.URL.Path != "/" {
		return &NotFoundError{Message: "Page not found"}
	}

	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/healthz", http.StatusTemporaryRedirect)
		return nil
	}

	units, err := store.ListUnitsByUserGroups(r.Context(), h.Store.DB(), user.Groups)
	if err != nil {
		return fmt.Errorf("home: list units: %w", err)
	}

	// If user has exactly one unit, redirect to it.
	if len(units) == 1 {
		http.Redirect(w, r, "/units/"+units[0].Slug, http.StatusTemporaryRedirect)
		return nil
	}

	// Otherwise, render a unit listing page.
	data := struct {
		Layout viewmodel.LayoutData
	}{
		Layout: h.NewLayoutData(r, "Home"),
	}
	h.Renderer.Page(w, r, "home", data)
	return nil
}

// UnitDashboard renders a placeholder page for a unit. The actual unit
// dashboard will be implemented in a future task.
func (h *Handler) UnitDashboard(w http.ResponseWriter, r *http.Request) error {
	slug := r.PathValue("slug")

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
