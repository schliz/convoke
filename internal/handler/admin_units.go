package handler

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/schliz/convoke/internal/db"
	"github.com/schliz/convoke/internal/store"
	"github.com/schliz/convoke/internal/viewmodel"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// isValidSlug checks whether a slug matches the required format:
// lowercase alphanumeric with hyphens, no leading/trailing/double hyphens.
func isValidSlug(slug string) bool {
	if slug == "" {
		return false
	}
	return slugPattern.MatchString(slug)
}

// parseGroupBindings filters, trims, and deduplicates group binding values
// from form input.
func parseGroupBindings(raw []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, g := range raw {
		g = strings.TrimSpace(g)
		if g != "" && !seen[g] {
			seen[g] = true
			result = append(result, g)
		}
	}
	return result
}

// AdminListUnits renders the unit list page.
func (h *Handler) AdminListUnits(w http.ResponseWriter, r *http.Request) error {
	units, err := store.ListUnits(r.Context(), h.Store.DB())
	if err != nil {
		return fmt.Errorf("admin list units: %w", err)
	}

	var items []viewmodel.AdminUnitListItem
	for _, u := range units {
		bindings, err := h.Store.Queries().GetUnitGroupBindings(r.Context(), u.ID)
		if err != nil {
			return fmt.Errorf("admin list units: get bindings for unit %d: %w", u.ID, err)
		}

		items = append(items, viewmodel.AdminUnitListItem{
			ID:            u.ID,
			Name:          u.Name,
			Slug:          u.Slug,
			Description:   u.Description.String,
			AdminGroup:    u.AdminGroup.String,
			ContactEmail:  u.ContactEmail.String,
			GroupBindings: bindings,
		})
	}

	data := viewmodel.AdminUnitsPage{
		Layout: h.NewLayoutData(r, "Unit Management"),
		Units:  items,
	}

	h.Renderer.Page(w, r, "admin_units", data)
	return nil
}

// AdminNewUnit renders the create unit form.
func (h *Handler) AdminNewUnit(w http.ResponseWriter, r *http.Request) error {
	data := viewmodel.AdminUnitFormPage{
		Layout: h.NewLayoutData(r, "Create Unit"),
		IsNew:  true,
		Unit:   viewmodel.AdminUnitFormData{},
		Errors: nil,
	}

	h.Renderer.Page(w, r, "admin_unit_form", data)
	return nil
}

// AdminCreateUnit handles the create unit form submission.
func (h *Handler) AdminCreateUnit(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return &ValidationError{Message: "Invalid form data"}
	}

	formData := viewmodel.AdminUnitFormData{
		Name:          strings.TrimSpace(r.FormValue("name")),
		Slug:          strings.TrimSpace(r.FormValue("slug")),
		Description:   strings.TrimSpace(r.FormValue("description")),
		ContactEmail:  strings.TrimSpace(r.FormValue("contact_email")),
		AdminGroup:    strings.TrimSpace(r.FormValue("admin_group")),
		GroupBindings: parseGroupBindings(r.Form["group_bindings"]),
	}

	// Validate
	errs := make(map[string]string)
	if formData.Name == "" {
		errs["name"] = "Name is required"
	}
	if formData.Slug == "" {
		errs["slug"] = "Slug is required"
	} else if !isValidSlug(formData.Slug) {
		errs["slug"] = "Slug must be lowercase alphanumeric with hyphens (e.g. fire-brigade)"
	}

	// Check slug uniqueness
	if formData.Slug != "" && isValidSlug(formData.Slug) {
		_, slugErr := store.GetUnitBySlug(r.Context(), h.Store.DB(), formData.Slug)
		if slugErr == nil {
			errs["slug"] = "A unit with this slug already exists"
		} else if !errors.Is(slugErr, pgx.ErrNoRows) {
			return fmt.Errorf("admin create unit: check slug: %w", slugErr)
		}
	}

	if len(errs) > 0 {
		data := viewmodel.AdminUnitFormPage{
			Layout: h.NewLayoutData(r, "Create Unit"),
			IsNew:  true,
			Unit:   formData,
			Errors: errs,
		}
		h.Renderer.Page(w, r, "admin_unit_form", data)
		return nil
	}

	// Create unit
	params := db.CreateUnitParams{
		Name:         formData.Name,
		Slug:         formData.Slug,
		Description:  strToText(formData.Description),
		LogoPath:     pgtype.Text{},
		ContactEmail: strToText(formData.ContactEmail),
		AdminGroup:   strToText(formData.AdminGroup),
	}

	_, err := h.Store.CreateUnitWithBindings(r.Context(), params, formData.GroupBindings)
	if err != nil {
		return fmt.Errorf("admin create unit: %w", err)
	}

	// Redirect to list
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/admin/units/")
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	http.Redirect(w, r, "/admin/units/", http.StatusSeeOther)
	return nil
}

// AdminEditUnit renders the edit unit form.
func (h *Handler) AdminEditUnit(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return &NotFoundError{Message: "Unit not found"}
	}

	unit, err := store.GetUnitByID(r.Context(), h.Store.DB(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &NotFoundError{Message: "Unit not found"}
		}
		return fmt.Errorf("admin edit unit: %w", err)
	}

	bindings, err := h.Store.Queries().GetUnitGroupBindings(r.Context(), unit.ID)
	if err != nil {
		return fmt.Errorf("admin edit unit: get bindings: %w", err)
	}

	data := viewmodel.AdminUnitFormPage{
		Layout: h.NewLayoutData(r, "Edit Unit"),
		IsNew:  false,
		Unit: viewmodel.AdminUnitFormData{
			ID:            unit.ID,
			Name:          unit.Name,
			Slug:          unit.Slug,
			Description:   unit.Description.String,
			ContactEmail:  unit.ContactEmail.String,
			AdminGroup:    unit.AdminGroup.String,
			GroupBindings: bindings,
		},
		Errors: nil,
	}

	h.Renderer.Page(w, r, "admin_unit_form", data)
	return nil
}

// AdminUpdateUnit handles the edit unit form submission.
func (h *Handler) AdminUpdateUnit(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return &NotFoundError{Message: "Unit not found"}
	}

	if err := r.ParseForm(); err != nil {
		return &ValidationError{Message: "Invalid form data"}
	}

	formData := viewmodel.AdminUnitFormData{
		ID:            id,
		Name:          strings.TrimSpace(r.FormValue("name")),
		Slug:          "", // slug is immutable, read from existing unit
		Description:   strings.TrimSpace(r.FormValue("description")),
		ContactEmail:  strings.TrimSpace(r.FormValue("contact_email")),
		AdminGroup:    strings.TrimSpace(r.FormValue("admin_group")),
		GroupBindings: parseGroupBindings(r.Form["group_bindings"]),
	}

	// Get existing unit for the slug (immutable)
	existing, err := store.GetUnitByID(r.Context(), h.Store.DB(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &NotFoundError{Message: "Unit not found"}
		}
		return fmt.Errorf("admin update unit: get existing: %w", err)
	}
	formData.Slug = existing.Slug

	// Validate
	errs := make(map[string]string)
	if formData.Name == "" {
		errs["name"] = "Name is required"
	}

	if len(errs) > 0 {
		data := viewmodel.AdminUnitFormPage{
			Layout: h.NewLayoutData(r, "Edit Unit"),
			IsNew:  false,
			Unit:   formData,
			Errors: errs,
		}
		h.Renderer.Page(w, r, "admin_unit_form", data)
		return nil
	}

	// Update unit
	params := db.UpdateUnitParams{
		ID:           id,
		Name:         formData.Name,
		Description:  strToText(formData.Description),
		LogoPath:     existing.LogoPath, // preserve existing logo
		ContactEmail: strToText(formData.ContactEmail),
		AdminGroup:   strToText(formData.AdminGroup),
	}

	_, err = h.Store.UpdateUnitWithBindings(r.Context(), params, formData.GroupBindings)
	if err != nil {
		return fmt.Errorf("admin update unit: %w", err)
	}

	// Redirect to list
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/admin/units/")
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	http.Redirect(w, r, "/admin/units/", http.StatusSeeOther)
	return nil
}

// AdminDeleteUnit handles unit deletion.
func (h *Handler) AdminDeleteUnit(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return &NotFoundError{Message: "Unit not found"}
	}

	if err := h.Store.Queries().DeleteUnit(r.Context(), id); err != nil {
		return fmt.Errorf("admin delete unit: %w", err)
	}

	if r.Header.Get("HX-Request") == "true" {
		// Return empty body; the hx-swap="delete" on the trigger element
		// will remove the unit row from the DOM.
		w.WriteHeader(http.StatusOK)
		return nil
	}
	http.Redirect(w, r, "/admin/units/", http.StatusSeeOther)
	return nil
}

// strToText converts a Go string to pgtype.Text, with Valid=false for empty strings.
func strToText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
