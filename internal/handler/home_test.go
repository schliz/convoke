package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v5"

	"github.com/schliz/convoke/internal/auth"
	"github.com/schliz/convoke/internal/config"
	"github.com/schliz/convoke/internal/render"
	"github.com/schliz/convoke/internal/store"
)

func TestHome_SingleUnit_RedirectsToUnit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(1), "Bar Committee", "bar-committee", pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("SELECT DISTINCT .+ FROM units").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(rows)

	s := store.NewWithPool(mock)
	h := &Handler{
		Store:    s,
		Renderer: render.New("../../templates", false),
		Config:   &config.Config{},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(withTestUser(req.Context(), &auth.RequestUser{
		ID:      1,
		Email:   "user@example.com",
		IsAdmin: false,
		Groups:  []string{"bar-committee"},
	}))
	rec := httptest.NewRecorder()

	if err := h.Home(rec, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := rec.Result()
	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "/units/bar-committee" {
		t.Errorf("expected redirect to /units/bar-committee, got %q", loc)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestHome_MultipleUnits_RendersHomePage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Single query: Home handler fetches units and passes them to
	// NewLayoutDataWithUnits, avoiding the duplicate query.
	rows := mock.NewRows(unitColumns).
		AddRow(int64(1), "Bar Committee", "bar-committee", pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}).
		AddRow(int64(2), "Kitchen Crew", "kitchen-crew", pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("SELECT DISTINCT .+ FROM units").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(rows)

	s := store.NewWithPool(mock)
	h := &Handler{
		Store:    s,
		Renderer: render.New("../../templates", false),
		Config:   &config.Config{},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(withTestUser(req.Context(), &auth.RequestUser{
		ID:      1,
		Email:   "user@example.com",
		IsAdmin: false,
		Groups:  []string{"bar-committee", "kitchen-crew"},
	}))
	rec := httptest.NewRecorder()

	if err := h.Home(rec, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := rec.Result()
	// Should render the page, not redirect
	if resp.StatusCode == http.StatusTemporaryRedirect {
		t.Error("expected page render, got redirect")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestHome_NoUnits_RendersHomePage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// No units query expected because groups are empty — ListUnitsByUserGroups
	// returns nil,nil for empty groups without a DB call.

	s := store.NewWithPool(mock)
	h := &Handler{
		Store:    s,
		Renderer: render.New("../../templates", false),
		Config:   &config.Config{},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(withTestUser(req.Context(), &auth.RequestUser{
		ID:      1,
		Email:   "user@example.com",
		IsAdmin: false,
		Groups:  nil,
	}))
	rec := httptest.NewRecorder()

	if err := h.Home(rec, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := rec.Result()
	if resp.StatusCode == http.StatusTemporaryRedirect {
		t.Error("expected page render, got redirect")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestHome_NonRootPath_Returns404(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	s := store.NewWithPool(mock)
	h := &Handler{
		Store:    s,
		Renderer: render.New("../../templates", false),
		Config:   &config.Config{},
	}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req = req.WithContext(withTestUser(req.Context(), &auth.RequestUser{
		ID:      1,
		Email:   "user@example.com",
		IsAdmin: false,
		Groups:  nil,
	}))
	rec := httptest.NewRecorder()

	err = h.Home(rec, req)
	if err == nil {
		t.Fatal("expected error for non-root path")
	}

	if !isNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

// unitColumns matches the column order used by unit queries.
var unitColumns = []string{
	"id", "name", "slug", "description", "logo_path",
	"contact_email", "admin_group", "created_at", "updated_at",
}

// withTestUser injects a RequestUser into the context using the auth package's
// exported context functions.
func withTestUser(ctx context.Context, user *auth.RequestUser) context.Context {
	return auth.ContextWithUser(ctx, user)
}

// isNotFoundError checks if err is a *NotFoundError.
func isNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}
