package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v5"

	"github.com/schliz/convoke/internal/model"
)

// entryWithCalendarColumns defines the column names returned by entry queries
// that join calendar data, matching the order in the sqlc-generated row types.
var entryWithCalendarColumns = []string{
	"id", "slug", "calendar_id", "name", "type",
	"starts_at", "ends_at", "location", "description",
	"response_deadline", "recurrence_rule_id", "created_at", "updated_at",
	"calendar_name", "calendar_slug", "calendar_color", "unit_id",
}

// entryColumns defines the column names returned by plain entry queries.
var entryColumns = []string{
	"id", "slug", "calendar_id", "name", "type",
	"starts_at", "ends_at", "location", "description",
	"response_deadline", "recurrence_rule_id", "created_at", "updated_at",
}

// shiftDetailColumns defines the column names for entry_shift_details.
var shiftDetailColumns = []string{
	"entry_id", "required_participants", "max_participants",
}

// testTime is a fixed reference time for tests.
var testTime = time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)

// makeEntryWithCalendarRow creates a row of values for an entry with calendar context.
func makeEntryWithCalendarRow(id int64, slug, name, entryType string, startsAt, endsAt time.Time, calName, calSlug string, unitID int64) []any {
	return []any{
		id, slug, int64(1), name, entryType,
		pgtype.Timestamptz{Time: startsAt, Valid: true},
		pgtype.Timestamptz{Time: endsAt, Valid: true},
		pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Int8{},
		pgtype.Timestamptz{Time: testTime, Valid: true},
		pgtype.Timestamptz{Time: testTime, Valid: true},
		calName, calSlug, pgtype.Text{String: "#3b82f6", Valid: true}, unitID,
	}
}

// --- GetEntryByID ---

func TestGetEntryByID_Shift(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 15, 16, 0, 0, 0, time.UTC)

	entryRow := mock.NewRows(entryWithCalendarColumns).
		AddRow(makeEntryWithCalendarRow(1, "shift-1", "Morning Shift", "shift", start, end, "Fire Cal", "fire-cal", int64(10))...)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(entryRow)

	shiftRow := mock.NewRows(shiftDetailColumns).
		AddRow(int64(1), int32(3), int32(5))

	mock.ExpectQuery("SELECT .+ FROM entry_shift_details WHERE entry_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(shiftRow)

	entry, err := GetEntryByID(context.Background(), mock, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != 1 {
		t.Errorf("expected ID 1, got %d", entry.ID)
	}
	if entry.Name != "Morning Shift" {
		t.Errorf("expected name 'Morning Shift', got %q", entry.Name)
	}
	if entry.Type != model.EntryTypeShift {
		t.Errorf("expected type 'shift', got %q", entry.Type)
	}
	if entry.CalendarName != "Fire Cal" {
		t.Errorf("expected calendar name 'Fire Cal', got %q", entry.CalendarName)
	}
	if entry.UnitID != 10 {
		t.Errorf("expected unit ID 10, got %d", entry.UnitID)
	}
	if entry.RequiredParticipants == nil || *entry.RequiredParticipants != 3 {
		t.Errorf("expected required_participants 3, got %v", entry.RequiredParticipants)
	}
	if entry.MaxParticipants == nil || *entry.MaxParticipants != 5 {
		t.Errorf("expected max_participants 5, got %v", entry.MaxParticipants)
	}
	if !entry.StartsAt.Equal(start) {
		t.Errorf("expected starts_at %v, got %v", start, entry.StartsAt)
	}
	if !entry.EndsAt.Equal(end) {
		t.Errorf("expected ends_at %v, got %v", end, entry.EndsAt)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetEntryByID_ShiftWithoutDetailsRow(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 15, 16, 0, 0, 0, time.UTC)

	entryRow := mock.NewRows(entryWithCalendarColumns).
		AddRow(makeEntryWithCalendarRow(1, "shift-1", "Morning Shift", "shift", start, end, "Fire Cal", "fire-cal", int64(10))...)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(entryRow)

	// Shift details row does not exist — should be handled gracefully.
	mock.ExpectQuery("SELECT .+ FROM entry_shift_details WHERE entry_id = \\$1").
		WithArgs(int64(1)).
		WillReturnError(pgx.ErrNoRows)

	entry, err := GetEntryByID(context.Background(), mock, 1)
	if err != nil {
		t.Fatalf("expected no error for missing shift details, got: %v", err)
	}
	if entry.ID != 1 {
		t.Errorf("expected ID 1, got %d", entry.ID)
	}
	if entry.Type != model.EntryTypeShift {
		t.Errorf("expected type 'shift', got %q", entry.Type)
	}
	// Shift fields should be nil when details row is missing.
	if entry.RequiredParticipants != nil {
		t.Errorf("expected nil required_participants, got %v", entry.RequiredParticipants)
	}
	if entry.MaxParticipants != nil {
		t.Errorf("expected nil max_participants, got %v", entry.MaxParticipants)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetEntryByID_Meeting(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 20, 15, 0, 0, 0, time.UTC)
	loc := "Conference Room A"

	entryRow := mock.NewRows(entryWithCalendarColumns).
		AddRow(
			int64(2), "meeting-1", int64(1), "Team Standup", "meeting",
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
			pgtype.Text{String: loc, Valid: true}, pgtype.Text{},
			pgtype.Timestamptz{}, pgtype.Int8{},
			pgtype.Timestamptz{Time: testTime, Valid: true},
			pgtype.Timestamptz{Time: testTime, Valid: true},
			"Meetings Cal", "meetings-cal", pgtype.Text{}, int64(5),
		)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.id = \\$1").
		WithArgs(int64(2)).
		WillReturnRows(entryRow)

	// Meeting type should NOT query shift details.

	entry, err := GetEntryByID(context.Background(), mock, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Type != model.EntryTypeMeeting {
		t.Errorf("expected type 'meeting', got %q", entry.Type)
	}
	if entry.Location == nil || *entry.Location != loc {
		t.Errorf("expected location %q, got %v", loc, entry.Location)
	}
	if entry.RequiredParticipants != nil {
		t.Errorf("expected nil required_participants for meeting, got %v", entry.RequiredParticipants)
	}
	if entry.CalendarColor != nil {
		t.Errorf("expected nil calendar_color, got %v", entry.CalendarColor)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetEntryByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.id = \\$1").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = GetEntryByID(context.Background(), mock, 999)
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

// --- GetEntryBySlug ---

func TestGetEntryBySlug_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC)

	entryRow := mock.NewRows(entryWithCalendarColumns).
		AddRow(makeEntryWithCalendarRow(5, "day-shift-apr", "Day Shift", "shift", start, end, "Ops Cal", "ops-cal", int64(3))...)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.slug = \\$1").
		WithArgs("day-shift-apr").
		WillReturnRows(entryRow)

	shiftRow := mock.NewRows(shiftDetailColumns).
		AddRow(int64(5), int32(2), int32(4))

	mock.ExpectQuery("SELECT .+ FROM entry_shift_details WHERE entry_id = \\$1").
		WithArgs(int64(5)).
		WillReturnRows(shiftRow)

	entry, err := GetEntryBySlug(context.Background(), mock, "day-shift-apr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Slug != "day-shift-apr" {
		t.Errorf("expected slug 'day-shift-apr', got %q", entry.Slug)
	}
	if entry.CalendarSlug != "ops-cal" {
		t.Errorf("expected calendar slug 'ops-cal', got %q", entry.CalendarSlug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetEntryBySlug_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.slug = \\$1").
		WithArgs("nonexistent").
		WillReturnError(pgx.ErrNoRows)

	_, err = GetEntryBySlug(context.Background(), mock, "nonexistent")
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

// --- GetEntryForUpdate ---

func TestGetEntryForUpdate_LocksEntryRow(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 15, 16, 0, 0, 0, time.UTC)

	entryRow := mock.NewRows(entryWithCalendarColumns).
		AddRow(makeEntryWithCalendarRow(7, "locked-shift", "Locked Shift", "shift", start, end, "Lock Cal", "lock-cal", int64(2))...)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.id = \\$1.+FOR UPDATE OF e").
		WithArgs(int64(7)).
		WillReturnRows(entryRow)

	entry, err := GetEntryForUpdate(context.Background(), mock, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != 7 {
		t.Errorf("expected ID 7, got %d", entry.ID)
	}
	if entry.Name != "Locked Shift" {
		t.Errorf("expected name 'Locked Shift', got %q", entry.Name)
	}
	// GetEntryForUpdate should NOT fetch shift details (not needed for locking).
	if entry.RequiredParticipants != nil {
		t.Errorf("expected nil required_participants, got %v", entry.RequiredParticipants)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetEntryForUpdate_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.id = \\$1.+FOR UPDATE OF e").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = GetEntryForUpdate(context.Background(), mock, 999)
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

// --- CreateEntry ---

func TestCreateEntry_Shift(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 20, 6, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	req := int32(2)
	max := int32(4)

	// Expect CreateEntry INSERT
	createRow := mock.NewRows(entryColumns).
		AddRow(
			int64(10), "new-shift", int64(1), "New Shift", "shift",
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
			pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Int8{},
			pgtype.Timestamptz{Time: testTime, Valid: true},
			pgtype.Timestamptz{Time: testTime, Valid: true},
		)
	mock.ExpectQuery("INSERT INTO entries").
		WithArgs("new-shift", int64(1), "New Shift", "shift",
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
			pgtype.Text{}, pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Int8{},
		).
		WillReturnRows(createRow)

	// Expect UpsertEntryShiftDetails
	shiftRow := mock.NewRows(shiftDetailColumns).
		AddRow(int64(10), int32(2), int32(4))
	mock.ExpectQuery("INSERT INTO entry_shift_details").
		WithArgs(int64(10), int32(2), int32(4)).
		WillReturnRows(shiftRow)

	// Expect GetEntryWithCalendar for the returned model.Entry
	finalRow := mock.NewRows(entryWithCalendarColumns).
		AddRow(makeEntryWithCalendarRow(10, "new-shift", "New Shift", "shift", start, end, "Fire Cal", "fire-cal", int64(1))...)
	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.id = \\$1").
		WithArgs(int64(10)).
		WillReturnRows(finalRow)

	// Expect GetEntryShiftDetails for the returned model
	shiftRow2 := mock.NewRows(shiftDetailColumns).
		AddRow(int64(10), int32(2), int32(4))
	mock.ExpectQuery("SELECT .+ FROM entry_shift_details WHERE entry_id = \\$1").
		WithArgs(int64(10)).
		WillReturnRows(shiftRow2)

	entry, err := CreateEntry(context.Background(), mock, CreateEntryParams{
		Slug:                 "new-shift",
		CalendarID:           1,
		Name:                 "New Shift",
		Type:                 model.EntryTypeShift,
		StartsAt:             start,
		EndsAt:               end,
		RequiredParticipants: &req,
		MaxParticipants:      &max,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != 10 {
		t.Errorf("expected ID 10, got %d", entry.ID)
	}
	if entry.Type != model.EntryTypeShift {
		t.Errorf("expected type shift, got %q", entry.Type)
	}
	if entry.RequiredParticipants == nil || *entry.RequiredParticipants != 2 {
		t.Errorf("expected required_participants 2, got %v", entry.RequiredParticipants)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateEntry_Meeting(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 21, 11, 0, 0, 0, time.UTC)
	loc := "Room B"
	desc := "Weekly sync"

	createRow := mock.NewRows(entryColumns).
		AddRow(
			int64(11), "weekly-sync", int64(2), "Weekly Sync", "meeting",
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
			pgtype.Text{String: loc, Valid: true},
			pgtype.Text{String: desc, Valid: true},
			pgtype.Timestamptz{}, pgtype.Int8{},
			pgtype.Timestamptz{Time: testTime, Valid: true},
			pgtype.Timestamptz{Time: testTime, Valid: true},
		)
	mock.ExpectQuery("INSERT INTO entries").
		WithArgs("weekly-sync", int64(2), "Weekly Sync", "meeting",
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
			pgtype.Text{String: loc, Valid: true},
			pgtype.Text{String: desc, Valid: true},
			pgtype.Timestamptz{}, pgtype.Int8{},
		).
		WillReturnRows(createRow)

	// No shift details for meetings.

	// Expect GetEntryWithCalendar
	finalRow := mock.NewRows(entryWithCalendarColumns).
		AddRow(
			int64(11), "weekly-sync", int64(2), "Weekly Sync", "meeting",
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
			pgtype.Text{String: loc, Valid: true},
			pgtype.Text{String: desc, Valid: true},
			pgtype.Timestamptz{}, pgtype.Int8{},
			pgtype.Timestamptz{Time: testTime, Valid: true},
			pgtype.Timestamptz{Time: testTime, Valid: true},
			"Meetings Cal", "meetings-cal", pgtype.Text{}, int64(5),
		)
	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.id = \\$1").
		WithArgs(int64(11)).
		WillReturnRows(finalRow)

	// No shift detail fetch for meetings.

	entry, err := CreateEntry(context.Background(), mock, CreateEntryParams{
		Slug:        "weekly-sync",
		CalendarID:  2,
		Name:        "Weekly Sync",
		Type:        model.EntryTypeMeeting,
		StartsAt:    start,
		EndsAt:      end,
		Location:    &loc,
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Type != model.EntryTypeMeeting {
		t.Errorf("expected type meeting, got %q", entry.Type)
	}
	if entry.Location == nil || *entry.Location != loc {
		t.Errorf("expected location %q, got %v", loc, entry.Location)
	}
	if entry.Description == nil || *entry.Description != desc {
		t.Errorf("expected description %q, got %v", desc, entry.Description)
	}
	if entry.RequiredParticipants != nil {
		t.Errorf("expected nil required_participants for meeting, got %v", entry.RequiredParticipants)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateEntry_PropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 20, 6, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)

	mock.ExpectQuery("INSERT INTO entries").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("unique violation"))

	_, err = CreateEntry(context.Background(), mock, CreateEntryParams{
		Slug:       "dup-entry",
		CalendarID: 1,
		Name:       "Dup Entry",
		Type:       model.EntryTypeShift,
		StartsAt:   start,
		EndsAt:     end,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- UpdateEntry ---

func TestUpdateEntry_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 22, 17, 0, 0, 0, time.UTC)
	newLoc := "Building C"
	updatedAt := time.Date(2026, 3, 22, 8, 0, 0, 0, time.UTC)

	updateRow := mock.NewRows(entryColumns).
		AddRow(
			int64(1), "shift-1", int64(1), "Updated Shift", "shift",
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
			pgtype.Text{String: newLoc, Valid: true}, pgtype.Text{},
			pgtype.Timestamptz{}, pgtype.Int8{},
			pgtype.Timestamptz{Time: testTime, Valid: true},
			pgtype.Timestamptz{Time: updatedAt, Valid: true},
		)
	mock.ExpectQuery("UPDATE entries SET").
		WithArgs(int64(1), "Updated Shift",
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
			pgtype.Text{String: newLoc, Valid: true},
			pgtype.Text{},
			pgtype.Timestamptz{},
		).
		WillReturnRows(updateRow)

	// Expect GetEntryWithCalendar for the returned model
	finalRow := mock.NewRows(entryWithCalendarColumns).
		AddRow(
			int64(1), "shift-1", int64(1), "Updated Shift", "shift",
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
			pgtype.Text{String: newLoc, Valid: true}, pgtype.Text{},
			pgtype.Timestamptz{}, pgtype.Int8{},
			pgtype.Timestamptz{Time: testTime, Valid: true},
			pgtype.Timestamptz{Time: updatedAt, Valid: true},
			"Fire Cal", "fire-cal", pgtype.Text{String: "#3b82f6", Valid: true}, int64(10),
		)
	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(finalRow)

	// Expect shift details since it's a shift
	shiftRow := mock.NewRows(shiftDetailColumns).
		AddRow(int64(1), int32(3), int32(5))
	mock.ExpectQuery("SELECT .+ FROM entry_shift_details WHERE entry_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(shiftRow)

	entry, err := UpdateEntry(context.Background(), mock, UpdateEntryParams{
		ID:       1,
		Name:     "Updated Shift",
		StartsAt: start,
		EndsAt:   end,
		Location: &newLoc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Name != "Updated Shift" {
		t.Errorf("expected name 'Updated Shift', got %q", entry.Name)
	}
	if entry.Location == nil || *entry.Location != newLoc {
		t.Errorf("expected location %q, got %v", newLoc, entry.Location)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestUpdateEntry_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 22, 17, 0, 0, 0, time.UTC)

	mock.ExpectQuery("UPDATE entries SET").
		WithArgs(int64(999), "Ghost", pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err = UpdateEntry(context.Background(), mock, UpdateEntryParams{
		ID:       999,
		Name:     "Ghost",
		StartsAt: start,
		EndsAt:   end,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- DeleteEntry ---

func TestDeleteEntry_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec("DELETE FROM entries WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = DeleteEntry(context.Background(), mock, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDeleteEntry_PropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec("DELETE FROM entries WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnError(fmt.Errorf("db error"))

	err = DeleteEntry(context.Background(), mock, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- ListEntriesByCalendar ---

func TestListEntriesByCalendar_ReturnsOrderedEntries(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	s1 := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	e1 := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	s2 := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)
	e2 := time.Date(2026, 3, 15, 18, 0, 0, 0, time.UTC)

	rows := mock.NewRows(entryWithCalendarColumns).
		AddRow(makeEntryWithCalendarRow(1, "shift-am", "AM Shift", "shift", s1, e1, "Cal", "cal", int64(1))...).
		AddRow(makeEntryWithCalendarRow(2, "shift-pm", "PM Shift", "shift", s2, e2, "Cal", "cal", int64(1))...)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.calendar_id = \\$1.+AND e\\.starts_at >= \\$2.+AND e\\.starts_at < \\$3.+ORDER BY e\\.starts_at").
		WithArgs(int64(1),
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
		).
		WillReturnRows(rows)

	entries, err := ListEntriesByCalendar(context.Background(), mock, 1, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "AM Shift" {
		t.Errorf("expected first entry 'AM Shift', got %q", entries[0].Name)
	}
	if entries[1].Name != "PM Shift" {
		t.Errorf("expected second entry 'PM Shift', got %q", entries[1].Name)
	}
	// Verify calendar context is populated.
	if entries[0].CalendarName != "Cal" {
		t.Errorf("expected calendar name 'Cal', got %q", entries[0].CalendarName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListEntriesByCalendar_EmptyRange(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// start == end returns zero rows.
	sameTime := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	rows := mock.NewRows(entryWithCalendarColumns)
	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.calendar_id = \\$1").
		WithArgs(int64(1),
			pgtype.Timestamptz{Time: sameTime, Valid: true},
			pgtype.Timestamptz{Time: sameTime, Valid: true},
		).
		WillReturnRows(rows)

	entries, err := ListEntriesByCalendar(context.Background(), mock, 1, sameTime, sameTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListEntriesByCalendar_PropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE e\\.calendar_id = \\$1").
		WithArgs(int64(1),
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
		).
		WillReturnError(fmt.Errorf("connection lost"))

	entries, err := ListEntriesByCalendar(context.Background(), mock, 1, start, end)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- ListEntriesByUnit ---

func TestListEntriesByUnit_ReturnsEntriesAcrossCalendars(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	s1 := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	e1 := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	s2 := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	e2 := time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC)

	rows := mock.NewRows(entryWithCalendarColumns).
		AddRow(makeEntryWithCalendarRow(1, "shift-1", "Morning Shift", "shift", s1, e1, "Ops Cal", "ops-cal", int64(10))...).
		AddRow(makeEntryWithCalendarRow(2, "meeting-1", "Standup", "meeting", s2, e2, "Meetings Cal", "meetings-cal", int64(10))...)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE c\\.unit_id = \\$1.+AND e\\.starts_at >= \\$2.+AND e\\.starts_at < \\$3.+ORDER BY e\\.starts_at").
		WithArgs(int64(10),
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
		).
		WillReturnRows(rows)

	entries, err := ListEntriesByUnit(context.Background(), mock, 10, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Both entries should have UnitID == 10.
	for i, e := range entries {
		if e.UnitID != 10 {
			t.Errorf("entry[%d]: expected unit_id 10, got %d", i, e.UnitID)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListEntriesByUnit_ReturnsEmptySlice(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)

	rows := mock.NewRows(entryWithCalendarColumns)
	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+WHERE c\\.unit_id = \\$1").
		WithArgs(int64(99),
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
		).
		WillReturnRows(rows)

	entries, err := ListEntriesByUnit(context.Background(), mock, 99, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- ListEntriesByUser ---

func TestListEntriesByUser_ReturnsAcceptedAndPending(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)
	s1 := time.Date(2026, 3, 16, 8, 0, 0, 0, time.UTC)
	e1 := time.Date(2026, 3, 16, 16, 0, 0, 0, time.UTC)
	s2 := time.Date(2026, 3, 18, 14, 0, 0, 0, time.UTC)
	e2 := time.Date(2026, 3, 18, 15, 0, 0, 0, time.UTC)

	rows := mock.NewRows(entryWithCalendarColumns).
		AddRow(makeEntryWithCalendarRow(3, "shift-a", "Day Shift", "shift", s1, e1, "Fire Cal", "fire-cal", int64(1))...).
		AddRow(makeEntryWithCalendarRow(5, "meeting-x", "Review", "meeting", s2, e2, "Meetings", "meetings", int64(1))...)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+JOIN attendances a.+WHERE a\\.user_id = \\$1.+AND a\\.status IN.+AND e\\.starts_at >= \\$2.+AND e\\.starts_at < \\$3.+ORDER BY e\\.starts_at").
		WithArgs(int64(42),
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
		).
		WillReturnRows(rows)

	entries, err := ListEntriesByUser(context.Background(), mock, 42, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Slug != "shift-a" {
		t.Errorf("expected first entry slug 'shift-a', got %q", entries[0].Slug)
	}
	if entries[1].Slug != "meeting-x" {
		t.Errorf("expected second entry slug 'meeting-x', got %q", entries[1].Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListEntriesByUser_NoAttendances(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)

	rows := mock.NewRows(entryWithCalendarColumns)
	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+JOIN attendances a.+WHERE a\\.user_id = \\$1").
		WithArgs(int64(42),
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
		).
		WillReturnRows(rows)

	entries, err := ListEntriesByUser(context.Background(), mock, 42, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListEntriesByUser_PropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery("SELECT e\\..+FROM entries e.+JOIN calendars c.+JOIN attendances a.+WHERE a\\.user_id = \\$1").
		WithArgs(int64(42),
			pgtype.Timestamptz{Time: start, Valid: true},
			pgtype.Timestamptz{Time: end, Valid: true},
		).
		WillReturnError(fmt.Errorf("connection refused"))

	entries, err := ListEntriesByUser(context.Background(), mock, 42, start, end)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- Conversion helpers ---

func TestTextToPtr_Valid(t *testing.T) {
	result := textToPtr(pgtype.Text{String: "hello", Valid: true})
	if result == nil || *result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestTextToPtr_Invalid(t *testing.T) {
	result := textToPtr(pgtype.Text{})
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestTsToPtr_Valid(t *testing.T) {
	now := time.Now()
	result := tsToPtr(pgtype.Timestamptz{Time: now, Valid: true})
	if result == nil || !result.Equal(now) {
		t.Errorf("expected %v, got %v", now, result)
	}
}

func TestTsToPtr_Invalid(t *testing.T) {
	result := tsToPtr(pgtype.Timestamptz{})
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestInt8ToPtr_Valid(t *testing.T) {
	result := int8ToPtr(pgtype.Int8{Int64: 42, Valid: true})
	if result == nil || *result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestInt8ToPtr_Invalid(t *testing.T) {
	result := int8ToPtr(pgtype.Int8{})
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestPtrToText_Nil(t *testing.T) {
	result := ptrToText(nil)
	if result.Valid {
		t.Errorf("expected invalid pgtype.Text, got valid")
	}
}

func TestPtrToText_NonNil(t *testing.T) {
	s := "hello"
	result := ptrToText(&s)
	if !result.Valid || result.String != "hello" {
		t.Errorf("expected valid pgtype.Text with 'hello', got %v", result)
	}
}

func TestPtrToTs_Nil(t *testing.T) {
	result := ptrToTs(nil)
	if result.Valid {
		t.Errorf("expected invalid pgtype.Timestamptz, got valid")
	}
}

func TestPtrToTs_NonNil(t *testing.T) {
	now := time.Now()
	result := ptrToTs(&now)
	if !result.Valid || !result.Time.Equal(now) {
		t.Errorf("expected valid pgtype.Timestamptz with %v, got %v", now, result)
	}
}

func TestPtrToInt8_Nil(t *testing.T) {
	result := ptrToInt8(nil)
	if result.Valid {
		t.Errorf("expected invalid pgtype.Int8, got valid")
	}
}

func TestPtrToInt8_NonNil(t *testing.T) {
	v := int64(42)
	result := ptrToInt8(&v)
	if !result.Valid || result.Int64 != 42 {
		t.Errorf("expected valid pgtype.Int8 with 42, got %v", result)
	}
}

func TestTimeToTs(t *testing.T) {
	now := time.Now()
	result := timeToTs(now)
	if !result.Valid || !result.Time.Equal(now) {
		t.Errorf("expected valid pgtype.Timestamptz with %v, got %v", now, result)
	}
}

func TestTsToTime(t *testing.T) {
	now := time.Now()
	result := tsToTime(pgtype.Timestamptz{Time: now, Valid: true})
	if !result.Equal(now) {
		t.Errorf("expected %v, got %v", now, result)
	}
}
