package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/schliz/convoke/internal/db"
)

// unitColumns defines the column names returned by unit queries, matching the
// order in the db.Unit struct and the SQL column list.
var unitColumns = []string{
	"id", "name", "slug", "description", "logo_path",
	"contact_email", "admin_group", "created_at", "updated_at",
}

// --- ListUnits ---

func TestListUnits_ReturnsAllUnits(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(1), "Alpha Unit", "alpha-unit", pgtype.Text{String: "Alpha description", Valid: true}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}).
		AddRow(int64(2), "Beta Unit", "beta-unit", pgtype.Text{String: "Beta description", Valid: true}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}).
		AddRow(int64(3), "Gamma Unit", "gamma-unit", pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("SELECT .+ FROM units ORDER BY name").
		WillReturnRows(rows)

	units, err := ListUnits(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 3 {
		t.Fatalf("expected 3 units, got %d", len(units))
	}
	if units[0].Name != "Alpha Unit" {
		t.Errorf("expected first unit name 'Alpha Unit', got %q", units[0].Name)
	}
	if units[1].Slug != "beta-unit" {
		t.Errorf("expected second unit slug 'beta-unit', got %q", units[1].Slug)
	}
	if units[2].ID != 3 {
		t.Errorf("expected third unit ID 3, got %d", units[2].ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListUnits_ReturnsEmptySlice(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(unitColumns)
	mock.ExpectQuery("SELECT .+ FROM units ORDER BY name").
		WillReturnRows(rows)

	units, err := ListUnits(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 0 {
		t.Fatalf("expected 0 units, got %d", len(units))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListUnits_PropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM units ORDER BY name").
		WillReturnError(fmt.Errorf("connection lost"))

	units, err := ListUnits(context.Background(), mock)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if units != nil {
		t.Errorf("expected nil units, got %v", units)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- GetUnitByID ---

func TestGetUnitByID_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(42), "Fire Brigade", "fire-brigade", pgtype.Text{String: "The fire brigade", Valid: true}, pgtype.Text{}, pgtype.Text{String: "fire@example.com", Valid: true}, pgtype.Text{String: "fire-admins", Valid: true}, pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("SELECT .+ FROM units WHERE id = \\$1").
		WithArgs(int64(42)).
		WillReturnRows(rows)

	unit, err := GetUnitByID(context.Background(), mock, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.ID != 42 {
		t.Errorf("expected ID 42, got %d", unit.ID)
	}
	if unit.Name != "Fire Brigade" {
		t.Errorf("expected name 'Fire Brigade', got %q", unit.Name)
	}
	if unit.AdminGroup.String != "fire-admins" {
		t.Errorf("expected admin_group 'fire-admins', got %q", unit.AdminGroup.String)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetUnitByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM units WHERE id = \\$1").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = GetUnitByID(context.Background(), mock, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != pgx.ErrNoRows {
		t.Errorf("expected pgx.ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetUnitByID_PropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM units WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnError(fmt.Errorf("db error"))

	_, err = GetUnitByID(context.Background(), mock, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- GetUnitBySlug ---

func TestGetUnitBySlug_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(7), "Bar Committee", "bar-committee", pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("SELECT .+ FROM units WHERE slug = \\$1").
		WithArgs("bar-committee").
		WillReturnRows(rows)

	unit, err := GetUnitBySlug(context.Background(), mock, "bar-committee")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.Slug != "bar-committee" {
		t.Errorf("expected slug 'bar-committee', got %q", unit.Slug)
	}
	if unit.Name != "Bar Committee" {
		t.Errorf("expected name 'Bar Committee', got %q", unit.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetUnitBySlug_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM units WHERE slug = \\$1").
		WithArgs("nonexistent").
		WillReturnError(pgx.ErrNoRows)

	_, err = GetUnitBySlug(context.Background(), mock, "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != pgx.ErrNoRows {
		t.Errorf("expected pgx.ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- ListUnitsByUserGroups ---

func TestListUnitsByUserGroups_ReturnsMatchingUnits(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(1), "Alpha Unit", "alpha-unit", pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}).
		AddRow(int64(3), "Gamma Unit", "gamma-unit", pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("SELECT DISTINCT .+ FROM units .+ JOIN unit_group_bindings .+ WHERE .+ ANY").
		WithArgs([]string{"group-a", "group-c"}).
		WillReturnRows(rows)

	units, err := ListUnitsByUserGroups(context.Background(), mock, []string{"group-a", "group-c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}
	if units[0].Name != "Alpha Unit" {
		t.Errorf("expected first unit 'Alpha Unit', got %q", units[0].Name)
	}
	if units[1].Name != "Gamma Unit" {
		t.Errorf("expected second unit 'Gamma Unit', got %q", units[1].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListUnitsByUserGroups_ReturnsEmptyForNonMatchingGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(unitColumns)
	mock.ExpectQuery("SELECT DISTINCT .+ FROM units .+ JOIN unit_group_bindings .+ WHERE .+ ANY").
		WithArgs([]string{"unknown-group"}).
		WillReturnRows(rows)

	units, err := ListUnitsByUserGroups(context.Background(), mock, []string{"unknown-group"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 0 {
		t.Fatalf("expected 0 units, got %d", len(units))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListUnitsByUserGroups_ShortCircuitsNilGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// No expectations set — the function should not touch the database.

	units, err := ListUnitsByUserGroups(context.Background(), mock, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if units != nil {
		t.Errorf("expected nil, got %v", units)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListUnitsByUserGroups_ShortCircuitsEmptyGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// No expectations set — the function should not touch the database.

	units, err := ListUnitsByUserGroups(context.Background(), mock, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if units != nil {
		t.Errorf("expected nil, got %v", units)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- IsUnitMember ---

func TestIsUnitMember_TrueWhenGroupMatches(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows([]string{"is_member"}).AddRow(true)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(10), []string{"fire-fighters"}).
		WillReturnRows(rows)

	result, err := IsUnitMember(context.Background(), mock, 10, []string{"fire-fighters"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true, got false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestIsUnitMember_FalseWhenNoMatch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows([]string{"is_member"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(10), []string{"unrelated-group"}).
		WillReturnRows(rows)

	result, err := IsUnitMember(context.Background(), mock, 10, []string{"unrelated-group"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false, got true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestIsUnitMember_ShortCircuitsNilGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	result, err := IsUnitMember(context.Background(), mock, 10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false for nil groups, got true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestIsUnitMember_ShortCircuitsEmptyGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	result, err := IsUnitMember(context.Background(), mock, 10, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false for empty groups, got true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- IsUnitAdmin ---

func TestIsUnitAdmin_TrueWhenAssocAdmin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// No expectations — should short-circuit without DB call.

	result, err := IsUnitAdmin(context.Background(), mock, 10, []string{"some-group"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true for assoc admin, got false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestIsUnitAdmin_TrueWhenGroupMatchesAdminGroup(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows([]string{"is_admin"}).AddRow(true)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(10), []string{"fire-admins"}).
		WillReturnRows(rows)

	result, err := IsUnitAdmin(context.Background(), mock, 10, []string{"fire-admins"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true, got false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestIsUnitAdmin_FalseWhenNoMatch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows([]string{"is_admin"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(10), []string{"some-group"}).
		WillReturnRows(rows)

	result, err := IsUnitAdmin(context.Background(), mock, 10, []string{"some-group"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false, got true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestIsUnitAdmin_FalseWhenAdminGroupIsNull(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// The SQL has `admin_group IS NOT NULL`, so a unit with NULL admin_group
	// returns false from EXISTS.
	rows := mock.NewRows([]string{"is_admin"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(10), []string{"some-group"}).
		WillReturnRows(rows)

	result, err := IsUnitAdmin(context.Background(), mock, 10, []string{"some-group"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false for unit with NULL admin_group, got true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestIsUnitAdmin_ShortCircuitsEmptyGroupsNotAssocAdmin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// No expectations — should short-circuit.

	result, err := IsUnitAdmin(context.Background(), mock, 10, []string{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false for empty groups and non-assoc-admin, got true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestIsUnitAdmin_ShortCircuitsNilGroupsNotAssocAdmin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	result, err := IsUnitAdmin(context.Background(), mock, 10, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false for nil groups and non-assoc-admin, got true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestIsUnitAdmin_AssocAdminWithEmptyGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Assoc admin should return true even with empty groups, no DB call.

	result, err := IsUnitAdmin(context.Background(), mock, 10, []string{}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true for assoc admin with empty groups, got false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- CreateUnitWithBindings ---

func TestCreateUnitWithBindings_CreatesUnitAndBindings(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.CreateUnitParams{
		Name:         "Fire Brigade",
		Slug:         "fire-brigade",
		Description:  pgtype.Text{String: "The fire brigade", Valid: true},
		LogoPath:     pgtype.Text{},
		ContactEmail: pgtype.Text{String: "fire@example.com", Valid: true},
		AdminGroup:   pgtype.Text{String: "fire-admins", Valid: true},
	}
	bindings := []string{"fire-fighters", "fire-volunteers"}

	mock.ExpectBegin()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(1), "Fire Brigade", "fire-brigade",
			pgtype.Text{String: "The fire brigade", Valid: true},
			pgtype.Text{},
			pgtype.Text{String: "fire@example.com", Valid: true},
			pgtype.Text{String: "fire-admins", Valid: true},
			pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("INSERT INTO units").
		WithArgs(params.Name, params.Slug, params.Description, params.LogoPath, params.ContactEmail, params.AdminGroup).
		WillReturnRows(rows)

	mock.ExpectExec("INSERT INTO unit_group_bindings").
		WithArgs(int64(1), "fire-fighters").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO unit_group_bindings").
		WithArgs(int64(1), "fire-volunteers").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	s := NewWithPool(mock)
	unit, err := s.CreateUnitWithBindings(context.Background(), params, bindings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.ID != 1 {
		t.Errorf("expected ID 1, got %d", unit.ID)
	}
	if unit.Name != "Fire Brigade" {
		t.Errorf("expected name 'Fire Brigade', got %q", unit.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateUnitWithBindings_NoBindings(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.CreateUnitParams{
		Name: "Solo Unit",
		Slug: "solo-unit",
	}

	mock.ExpectBegin()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(2), "Solo Unit", "solo-unit",
			pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{},
			pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("INSERT INTO units").
		WithArgs(params.Name, params.Slug, params.Description, params.LogoPath, params.ContactEmail, params.AdminGroup).
		WillReturnRows(rows)

	mock.ExpectCommit()

	s := NewWithPool(mock)
	unit, err := s.CreateUnitWithBindings(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.Name != "Solo Unit" {
		t.Errorf("expected name 'Solo Unit', got %q", unit.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateUnitWithBindings_RollsBackOnBindingError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.CreateUnitParams{
		Name: "Fail Unit",
		Slug: "fail-unit",
	}

	mock.ExpectBegin()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(3), "Fail Unit", "fail-unit",
			pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{},
			pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("INSERT INTO units").
		WithArgs(params.Name, params.Slug, params.Description, params.LogoPath, params.ContactEmail, params.AdminGroup).
		WillReturnRows(rows)

	mock.ExpectExec("INSERT INTO unit_group_bindings").
		WithArgs(int64(3), "bad-group").
		WillReturnError(fmt.Errorf("constraint violation"))

	mock.ExpectRollback()

	s := NewWithPool(mock)
	_, err = s.CreateUnitWithBindings(context.Background(), params, []string{"bad-group"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- UpdateUnitWithBindings ---

func TestUpdateUnitWithBindings_UpdatesUnitAndReplacesBindings(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.UpdateUnitParams{
		ID:           1,
		Name:         "Updated Brigade",
		Description:  pgtype.Text{String: "Updated desc", Valid: true},
		LogoPath:     pgtype.Text{},
		ContactEmail: pgtype.Text{String: "new@example.com", Valid: true},
		AdminGroup:   pgtype.Text{String: "fire-admins", Valid: true},
	}
	bindings := []string{"new-group-a", "new-group-b"}

	mock.ExpectBegin()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(1), "Updated Brigade", "fire-brigade",
			pgtype.Text{String: "Updated desc", Valid: true},
			pgtype.Text{},
			pgtype.Text{String: "new@example.com", Valid: true},
			pgtype.Text{String: "fire-admins", Valid: true},
			pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("UPDATE units SET").
		WithArgs(params.ID, params.Name, params.Description, params.LogoPath, params.ContactEmail, params.AdminGroup).
		WillReturnRows(rows)

	mock.ExpectExec("DELETE FROM unit_group_bindings WHERE unit_id = \\$1").
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	mock.ExpectExec("INSERT INTO unit_group_bindings").
		WithArgs(int64(1), "new-group-a").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO unit_group_bindings").
		WithArgs(int64(1), "new-group-b").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	s := NewWithPool(mock)
	unit, err := s.UpdateUnitWithBindings(context.Background(), params, bindings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.Name != "Updated Brigade" {
		t.Errorf("expected name 'Updated Brigade', got %q", unit.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestUpdateUnitWithBindings_ClearsBindingsWhenEmpty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.UpdateUnitParams{
		ID:   1,
		Name: "No Bindings Unit",
	}

	mock.ExpectBegin()

	rows := mock.NewRows(unitColumns).
		AddRow(int64(1), "No Bindings Unit", "fire-brigade",
			pgtype.Text{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{},
			pgtype.Timestamptz{}, pgtype.Timestamptz{})

	mock.ExpectQuery("UPDATE units SET").
		WithArgs(params.ID, params.Name, params.Description, params.LogoPath, params.ContactEmail, params.AdminGroup).
		WillReturnRows(rows)

	mock.ExpectExec("DELETE FROM unit_group_bindings WHERE unit_id = \\$1").
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	mock.ExpectCommit()

	s := NewWithPool(mock)
	unit, err := s.UpdateUnitWithBindings(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.Name != "No Bindings Unit" {
		t.Errorf("expected name 'No Bindings Unit', got %q", unit.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestUpdateUnitWithBindings_RollsBackOnUpdateError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.UpdateUnitParams{
		ID:   999,
		Name: "Nonexistent",
	}

	mock.ExpectBegin()

	mock.ExpectQuery("UPDATE units SET").
		WithArgs(params.ID, params.Name, params.Description, params.LogoPath, params.ContactEmail, params.AdminGroup).
		WillReturnError(pgx.ErrNoRows)

	mock.ExpectRollback()

	s := NewWithPool(mock)
	_, err = s.UpdateUnitWithBindings(context.Background(), params, []string{"group-a"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
