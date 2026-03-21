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

// calendarColumns defines the column names returned by calendar queries,
// matching the order in the db.Calendar struct and the SQL column list.
var calendarColumns = []string{
	"id", "slug", "unit_id", "name", "creation_policy", "visibility",
	"participation", "participant_visibility", "color", "sort_order",
	"created_at", "updated_at",
}

// calendarWithUnitColumns extends calendarColumns with the joined unit fields.
var calendarWithUnitColumns = append(
	append([]string{}, calendarColumns...),
	"unit_name", "unit_slug",
)

// customViewerUnitColumns defines the columns returned by GetCustomViewerUnits.
var customViewerUnitColumns = []string{"id", "name", "slug"}

// addCalendarRow adds a calendar row to the given pgxmock.Rows.
func addCalendarRow(rows *pgxmock.Rows, id int64, slug string, unitID int64, name, visibility string, sortOrder int32) *pgxmock.Rows {
	return rows.AddRow(
		id, slug, unitID, name,
		"admins_only",   // creation_policy
		visibility,      // visibility
		"viewers",       // participation
		"everyone",      // participant_visibility
		pgtype.Text{},   // color
		sortOrder,       // sort_order
		pgtype.Timestamptz{}, // created_at
		pgtype.Timestamptz{}, // updated_at
	)
}

// --- GetCalendarByID ---

func TestGetCalendarByID_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 1, "test-cal", 10, "Test Calendar", "association", 0)

	mock.ExpectQuery("SELECT .+ FROM calendars WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	cal, err := GetCalendarByID(context.Background(), mock, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cal.ID != 1 {
		t.Errorf("expected ID 1, got %d", cal.ID)
	}
	if cal.Slug != "test-cal" {
		t.Errorf("expected slug 'test-cal', got %q", cal.Slug)
	}
	if cal.Name != "Test Calendar" {
		t.Errorf("expected name 'Test Calendar', got %q", cal.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetCalendarByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM calendars WHERE id = \\$1").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = GetCalendarByID(context.Background(), mock, 999)
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

// --- GetCalendarBySlug ---

func TestGetCalendarBySlug_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 5, "my-cal", 10, "My Calendar", "unit", 1)

	mock.ExpectQuery("SELECT .+ FROM calendars WHERE slug = \\$1").
		WithArgs("my-cal").
		WillReturnRows(rows)

	cal, err := GetCalendarBySlug(context.Background(), mock, "my-cal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cal.Slug != "my-cal" {
		t.Errorf("expected slug 'my-cal', got %q", cal.Slug)
	}
	if cal.Name != "My Calendar" {
		t.Errorf("expected name 'My Calendar', got %q", cal.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetCalendarBySlug_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM calendars WHERE slug = \\$1").
		WithArgs("nonexistent").
		WillReturnError(pgx.ErrNoRows)

	_, err = GetCalendarBySlug(context.Background(), mock, "nonexistent")
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

// --- GetCalendarWithUnit ---

func TestGetCalendarWithUnit_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(calendarWithUnitColumns).AddRow(
		int64(1), "test-cal", int64(10), "Test Calendar",
		"admins_only", "association", "viewers", "everyone",
		pgtype.Text{}, int32(0),
		pgtype.Timestamptz{}, pgtype.Timestamptz{},
		"Fire Brigade", "fire-brigade",
	)

	mock.ExpectQuery("SELECT .+ FROM calendars c JOIN units u ON .+ WHERE c\\.id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	cal, err := GetCalendarWithUnit(context.Background(), mock, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cal.ID != 1 {
		t.Errorf("expected ID 1, got %d", cal.ID)
	}
	if cal.Name != "Test Calendar" {
		t.Errorf("expected name 'Test Calendar', got %q", cal.Name)
	}
	if cal.UnitName != "Fire Brigade" {
		t.Errorf("expected unit name 'Fire Brigade', got %q", cal.UnitName)
	}
	if cal.UnitSlug != "fire-brigade" {
		t.Errorf("expected unit slug 'fire-brigade', got %q", cal.UnitSlug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetCalendarWithUnit_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM calendars c JOIN units u ON .+ WHERE c\\.id = \\$1").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = GetCalendarWithUnit(context.Background(), mock, 999)
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

// --- ListCalendarsByUnit ---

func TestListCalendarsByUnit_ReturnsOrderedCalendars(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 1, "cal-a", 10, "Alpha Cal", "association", 0)
	addCalendarRow(rows, 2, "cal-b", 10, "Beta Cal", "unit", 1)
	addCalendarRow(rows, 3, "cal-c", 10, "Charlie Cal", "custom", 1)

	mock.ExpectQuery("SELECT .+ FROM calendars WHERE unit_id = \\$1 ORDER BY sort_order, name").
		WithArgs(int64(10)).
		WillReturnRows(rows)

	cals, err := ListCalendarsByUnit(context.Background(), mock, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cals) != 3 {
		t.Fatalf("expected 3 calendars, got %d", len(cals))
	}
	if cals[0].Name != "Alpha Cal" {
		t.Errorf("expected first calendar 'Alpha Cal', got %q", cals[0].Name)
	}
	if cals[1].SortOrder != 1 {
		t.Errorf("expected second calendar sort_order 1, got %d", cals[1].SortOrder)
	}
	if cals[2].Slug != "cal-c" {
		t.Errorf("expected third calendar slug 'cal-c', got %q", cals[2].Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListCalendarsByUnit_ReturnsEmptySlice(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(calendarColumns)
	mock.ExpectQuery("SELECT .+ FROM calendars WHERE unit_id = \\$1 ORDER BY sort_order, name").
		WithArgs(int64(99)).
		WillReturnRows(rows)

	cals, err := ListCalendarsByUnit(context.Background(), mock, 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cals) != 0 {
		t.Fatalf("expected 0 calendars, got %d", len(cals))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListCalendarsByUnit_PropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM calendars WHERE unit_id = \\$1 ORDER BY sort_order, name").
		WithArgs(int64(10)).
		WillReturnError(fmt.Errorf("connection lost"))

	cals, err := ListCalendarsByUnit(context.Background(), mock, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if cals != nil {
		t.Errorf("expected nil calendars, got %v", cals)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- DeleteCalendar ---

func TestDeleteCalendar_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec("DELETE FROM calendars WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = DeleteCalendar(context.Background(), mock, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDeleteCalendar_NonExistentIsNotError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec("DELETE FROM calendars WHERE id = \\$1").
		WithArgs(int64(999)).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = DeleteCalendar(context.Background(), mock, 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDeleteCalendar_PropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec("DELETE FROM calendars WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnError(fmt.Errorf("db error"))

	err = DeleteCalendar(context.Background(), mock, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- ListVisibleCalendars ---

func TestListVisibleCalendars_AdminSeesAll(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 1, "cal-a", 10, "Association Cal", "association", 0)
	addCalendarRow(rows, 2, "cal-b", 10, "Unit Cal", "unit", 1)
	addCalendarRow(rows, 3, "cal-c", 20, "Custom Cal", "custom", 2)

	mock.ExpectQuery("SELECT .+ FROM calendars ORDER BY sort_order, name").
		WillReturnRows(rows)

	cals, err := ListVisibleCalendars(context.Background(), mock, 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cals) != 3 {
		t.Fatalf("expected 3 calendars for admin, got %d", len(cals))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListVisibleCalendars_NonAdminUsesVisibilityQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 1, "cal-a", 10, "Association Cal", "association", 0)

	mock.ExpectQuery("SELECT DISTINCT .+ FROM calendars c WHERE").
		WithArgs(int64(42)).
		WillReturnRows(rows)

	cals, err := ListVisibleCalendars(context.Background(), mock, 42, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cals) != 1 {
		t.Fatalf("expected 1 calendar for non-admin, got %d", len(cals))
	}
	if cals[0].Name != "Association Cal" {
		t.Errorf("expected 'Association Cal', got %q", cals[0].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListVisibleCalendars_AdminPropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM calendars ORDER BY sort_order, name").
		WillReturnError(fmt.Errorf("db error"))

	_, err = ListVisibleCalendars(context.Background(), mock, 1, true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListVisibleCalendars_NonAdminPropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT DISTINCT .+ FROM calendars c WHERE").
		WithArgs(int64(42)).
		WillReturnError(fmt.Errorf("connection lost"))

	_, err = ListVisibleCalendars(context.Background(), mock, 42, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- GetCustomViewerUnits ---

func TestGetCustomViewerUnits_ReturnsUnits(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(customViewerUnitColumns).
		AddRow(int64(10), "Alpha Unit", "alpha-unit").
		AddRow(int64(20), "Beta Unit", "beta-unit")

	mock.ExpectQuery("SELECT .+ FROM calendar_custom_viewers .+ JOIN units .+ WHERE .+calendar_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	units, err := GetCustomViewerUnits(context.Background(), mock, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}
	if units[0].Name != "Alpha Unit" {
		t.Errorf("expected first unit 'Alpha Unit', got %q", units[0].Name)
	}
	if units[1].Slug != "beta-unit" {
		t.Errorf("expected second unit slug 'beta-unit', got %q", units[1].Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetCustomViewerUnits_ReturnsEmptySlice(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows(customViewerUnitColumns)
	mock.ExpectQuery("SELECT .+ FROM calendar_custom_viewers .+ JOIN units .+ WHERE .+calendar_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	units, err := GetCustomViewerUnits(context.Background(), mock, 1)
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

// --- CreateCalendarWithViewers ---

func TestCreateCalendarWithViewers_AssociationVisibility(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.CreateCalendarParams{
		Slug:                  "new-cal",
		UnitID:                10,
		Name:                  "New Calendar",
		CreationPolicy:        "admins_only",
		Visibility:            "association",
		Participation:         "viewers",
		ParticipantVisibility: "everyone",
		Color:                 pgtype.Text{},
		SortOrder:             0,
	}

	// Expect transaction
	mock.ExpectBegin()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 1, "new-cal", 10, "New Calendar", "association", 0)

	mock.ExpectQuery("INSERT INTO calendars").
		WithArgs(
			params.Slug, params.UnitID, params.Name,
			params.CreationPolicy, params.Visibility,
			params.Participation, params.ParticipantVisibility,
			params.Color, params.SortOrder,
		).
		WillReturnRows(rows)

	mock.ExpectCommit()

	s := &Store{pool: nil, queries: nil}
	s.pool = nil // WithTx uses pool, but we mock the whole pool interface

	cal, err := CreateCalendarWithViewers(context.Background(), mock, params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cal.Name != "New Calendar" {
		t.Errorf("expected name 'New Calendar', got %q", cal.Name)
	}
	if cal.Slug != "new-cal" {
		t.Errorf("expected slug 'new-cal', got %q", cal.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateCalendarWithViewers_CustomVisibility(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.CreateCalendarParams{
		Slug:                  "custom-cal",
		UnitID:                10,
		Name:                  "Custom Calendar",
		CreationPolicy:        "admins_only",
		Visibility:            "custom",
		Participation:         "viewers",
		ParticipantVisibility: "everyone",
		Color:                 pgtype.Text{},
		SortOrder:             0,
	}

	mock.ExpectBegin()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 5, "custom-cal", 10, "Custom Calendar", "custom", 0)

	mock.ExpectQuery("INSERT INTO calendars").
		WithArgs(
			params.Slug, params.UnitID, params.Name,
			params.CreationPolicy, params.Visibility,
			params.Participation, params.ParticipantVisibility,
			params.Color, params.SortOrder,
		).
		WillReturnRows(rows)

	// Expect custom viewer inserts
	mock.ExpectExec("INSERT INTO calendar_custom_viewers").
		WithArgs(int64(5), int64(20)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO calendar_custom_viewers").
		WithArgs(int64(5), int64(30)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	cal, err := CreateCalendarWithViewers(context.Background(), mock, params, []int64{20, 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cal.ID != 5 {
		t.Errorf("expected ID 5, got %d", cal.ID)
	}
	if cal.Visibility != "custom" {
		t.Errorf("expected visibility 'custom', got %q", cal.Visibility)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateCalendarWithViewers_RollsBackOnInsertError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.CreateCalendarParams{
		Slug:                  "fail-cal",
		UnitID:                10,
		Name:                  "Fail Calendar",
		CreationPolicy:        "admins_only",
		Visibility:            "custom",
		Participation:         "viewers",
		ParticipantVisibility: "everyone",
		Color:                 pgtype.Text{},
		SortOrder:             0,
	}

	mock.ExpectBegin()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 7, "fail-cal", 10, "Fail Calendar", "custom", 0)

	mock.ExpectQuery("INSERT INTO calendars").
		WithArgs(
			params.Slug, params.UnitID, params.Name,
			params.CreationPolicy, params.Visibility,
			params.Participation, params.ParticipantVisibility,
			params.Color, params.SortOrder,
		).
		WillReturnRows(rows)

	mock.ExpectExec("INSERT INTO calendar_custom_viewers").
		WithArgs(int64(7), int64(20)).
		WillReturnError(fmt.Errorf("FK violation"))

	mock.ExpectRollback()

	_, err = CreateCalendarWithViewers(context.Background(), mock, params, []int64{20})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- UpdateCalendarWithViewers ---

func TestUpdateCalendarWithViewers_ChangesToCustom(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.UpdateCalendarParams{
		ID:                    1,
		Name:                  "Updated Calendar",
		CreationPolicy:        "admins_only",
		Visibility:            "custom",
		Participation:         "viewers",
		ParticipantVisibility: "everyone",
		Color:                 pgtype.Text{},
		SortOrder:             0,
	}

	mock.ExpectBegin()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 1, "updated-cal", 10, "Updated Calendar", "custom", 0)

	mock.ExpectQuery("UPDATE calendars SET").
		WithArgs(
			params.ID, params.Name,
			params.CreationPolicy, params.Visibility,
			params.Participation, params.ParticipantVisibility,
			params.Color, params.SortOrder,
		).
		WillReturnRows(rows)

	// Always clear custom viewers first
	mock.ExpectExec("DELETE FROM calendar_custom_viewers WHERE calendar_id = \\$1").
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	// Insert new viewers
	mock.ExpectExec("INSERT INTO calendar_custom_viewers").
		WithArgs(int64(1), int64(20)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	cal, err := UpdateCalendarWithViewers(context.Background(), mock, params, []int64{20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cal.Name != "Updated Calendar" {
		t.Errorf("expected name 'Updated Calendar', got %q", cal.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestUpdateCalendarWithViewers_ChangesToUnitClearsViewers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.UpdateCalendarParams{
		ID:                    1,
		Name:                  "Unit Calendar",
		CreationPolicy:        "admins_only",
		Visibility:            "unit",
		Participation:         "viewers",
		ParticipantVisibility: "everyone",
		Color:                 pgtype.Text{},
		SortOrder:             0,
	}

	mock.ExpectBegin()

	rows := mock.NewRows(calendarColumns)
	addCalendarRow(rows, 1, "unit-cal", 10, "Unit Calendar", "unit", 0)

	mock.ExpectQuery("UPDATE calendars SET").
		WithArgs(
			params.ID, params.Name,
			params.CreationPolicy, params.Visibility,
			params.Participation, params.ParticipantVisibility,
			params.Color, params.SortOrder,
		).
		WillReturnRows(rows)

	// Clear custom viewers (even though visibility is not 'custom')
	mock.ExpectExec("DELETE FROM calendar_custom_viewers WHERE calendar_id = \\$1").
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	// No InsertCalendarCustomViewer calls expected when visibility != 'custom'

	mock.ExpectCommit()

	cal, err := UpdateCalendarWithViewers(context.Background(), mock, params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cal.Visibility != "unit" {
		t.Errorf("expected visibility 'unit', got %q", cal.Visibility)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestUpdateCalendarWithViewers_RollsBackOnError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	params := db.UpdateCalendarParams{
		ID:                    1,
		Name:                  "Fail Update",
		CreationPolicy:        "admins_only",
		Visibility:            "custom",
		Participation:         "viewers",
		ParticipantVisibility: "everyone",
		Color:                 pgtype.Text{},
		SortOrder:             0,
	}

	mock.ExpectBegin()

	mock.ExpectQuery("UPDATE calendars SET").
		WithArgs(
			params.ID, params.Name,
			params.CreationPolicy, params.Visibility,
			params.Participation, params.ParticipantVisibility,
			params.Color, params.SortOrder,
		).
		WillReturnError(fmt.Errorf("db error"))

	mock.ExpectRollback()

	_, err = UpdateCalendarWithViewers(context.Background(), mock, params, []int64{20})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
