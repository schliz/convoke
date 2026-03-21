package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/schliz/convoke/internal/db"
	"github.com/schliz/convoke/internal/model"
)

// --- Row conversion ---

// entryFromGetWithCalendarRow converts a GetEntryWithCalendar row to model.Entry.
func entryFromGetWithCalendarRow(row db.GetEntryWithCalendarRow) model.Entry {
	return model.Entry{
		ID:               row.ID,
		Slug:             row.Slug,
		CalendarID:       row.CalendarID,
		Name:             row.Name,
		Type:             model.EntryType(row.Type),
		StartsAt:         tsToTime(row.StartsAt),
		EndsAt:           tsToTime(row.EndsAt),
		Location:         textToPtr(row.Location),
		Description:      textToPtr(row.Description),
		ResponseDeadline: tsToPtr(row.ResponseDeadline),
		RecurrenceRuleID: int8ToPtr(row.RecurrenceRuleID),
		CreatedAt:        tsToTime(row.CreatedAt),
		UpdatedAt:        tsToTime(row.UpdatedAt),
		CalendarName:     row.CalendarName,
		CalendarSlug:     row.CalendarSlug,
		CalendarColor:    textToPtr(row.CalendarColor),
		UnitID:           row.UnitID,
	}
}

// entryFromGetWithCalendarBySlugRow converts a GetEntryWithCalendarBySlug row to model.Entry.
func entryFromGetWithCalendarBySlugRow(row db.GetEntryWithCalendarBySlugRow) model.Entry {
	return model.Entry{
		ID:               row.ID,
		Slug:             row.Slug,
		CalendarID:       row.CalendarID,
		Name:             row.Name,
		Type:             model.EntryType(row.Type),
		StartsAt:         tsToTime(row.StartsAt),
		EndsAt:           tsToTime(row.EndsAt),
		Location:         textToPtr(row.Location),
		Description:      textToPtr(row.Description),
		ResponseDeadline: tsToPtr(row.ResponseDeadline),
		RecurrenceRuleID: int8ToPtr(row.RecurrenceRuleID),
		CreatedAt:        tsToTime(row.CreatedAt),
		UpdatedAt:        tsToTime(row.UpdatedAt),
		CalendarName:     row.CalendarName,
		CalendarSlug:     row.CalendarSlug,
		CalendarColor:    textToPtr(row.CalendarColor),
		UnitID:           row.UnitID,
	}
}

// entryFromForUpdateRow converts a GetEntryForUpdate row to model.Entry.
func entryFromForUpdateRow(row db.GetEntryForUpdateRow) model.Entry {
	return model.Entry{
		ID:               row.ID,
		Slug:             row.Slug,
		CalendarID:       row.CalendarID,
		Name:             row.Name,
		Type:             model.EntryType(row.Type),
		StartsAt:         tsToTime(row.StartsAt),
		EndsAt:           tsToTime(row.EndsAt),
		Location:         textToPtr(row.Location),
		Description:      textToPtr(row.Description),
		ResponseDeadline: tsToPtr(row.ResponseDeadline),
		RecurrenceRuleID: int8ToPtr(row.RecurrenceRuleID),
		CreatedAt:        tsToTime(row.CreatedAt),
		UpdatedAt:        tsToTime(row.UpdatedAt),
		CalendarName:     row.CalendarName,
		CalendarSlug:     row.CalendarSlug,
		CalendarColor:    textToPtr(row.CalendarColor),
		UnitID:           row.UnitID,
	}
}

// entryFromCalendarWithCalendarRow converts a ListEntriesByCalendarWithCalendar row to model.Entry.
func entryFromCalendarWithCalendarRow(row db.ListEntriesByCalendarWithCalendarRow) model.Entry {
	return model.Entry{
		ID:               row.ID,
		Slug:             row.Slug,
		CalendarID:       row.CalendarID,
		Name:             row.Name,
		Type:             model.EntryType(row.Type),
		StartsAt:         tsToTime(row.StartsAt),
		EndsAt:           tsToTime(row.EndsAt),
		Location:         textToPtr(row.Location),
		Description:      textToPtr(row.Description),
		ResponseDeadline: tsToPtr(row.ResponseDeadline),
		RecurrenceRuleID: int8ToPtr(row.RecurrenceRuleID),
		CreatedAt:        tsToTime(row.CreatedAt),
		UpdatedAt:        tsToTime(row.UpdatedAt),
		CalendarName:     row.CalendarName,
		CalendarSlug:     row.CalendarSlug,
		CalendarColor:    textToPtr(row.CalendarColor),
		UnitID:           row.UnitID,
	}
}

// entryFromUnitRow converts a ListEntriesByUnit row to model.Entry.
func entryFromUnitRow(row db.ListEntriesByUnitRow) model.Entry {
	return model.Entry{
		ID:               row.ID,
		Slug:             row.Slug,
		CalendarID:       row.CalendarID,
		Name:             row.Name,
		Type:             model.EntryType(row.Type),
		StartsAt:         tsToTime(row.StartsAt),
		EndsAt:           tsToTime(row.EndsAt),
		Location:         textToPtr(row.Location),
		Description:      textToPtr(row.Description),
		ResponseDeadline: tsToPtr(row.ResponseDeadline),
		RecurrenceRuleID: int8ToPtr(row.RecurrenceRuleID),
		CreatedAt:        tsToTime(row.CreatedAt),
		UpdatedAt:        tsToTime(row.UpdatedAt),
		CalendarName:     row.CalendarName,
		CalendarSlug:     row.CalendarSlug,
		CalendarColor:    textToPtr(row.CalendarColor),
		UnitID:           row.UnitID,
	}
}

// entryFromUserRow converts a ListEntriesByUser row to model.Entry.
func entryFromUserRow(row db.ListEntriesByUserRow) model.Entry {
	return model.Entry{
		ID:               row.ID,
		Slug:             row.Slug,
		CalendarID:       row.CalendarID,
		Name:             row.Name,
		Type:             model.EntryType(row.Type),
		StartsAt:         tsToTime(row.StartsAt),
		EndsAt:           tsToTime(row.EndsAt),
		Location:         textToPtr(row.Location),
		Description:      textToPtr(row.Description),
		ResponseDeadline: tsToPtr(row.ResponseDeadline),
		RecurrenceRuleID: int8ToPtr(row.RecurrenceRuleID),
		CreatedAt:        tsToTime(row.CreatedAt),
		UpdatedAt:        tsToTime(row.UpdatedAt),
		CalendarName:     row.CalendarName,
		CalendarSlug:     row.CalendarSlug,
		CalendarColor:    textToPtr(row.CalendarColor),
		UnitID:           row.UnitID,
	}
}

// --- Store method param types ---

// CreateEntryParams holds the input for CreateEntry.
type CreateEntryParams struct {
	Slug                 string
	CalendarID           int64
	Name                 string
	Type                 model.EntryType
	StartsAt             time.Time
	EndsAt               time.Time
	Location             *string
	Description          *string
	ResponseDeadline     *time.Time
	RecurrenceRuleID     *int64
	RequiredParticipants *int32 // required when Type == EntryTypeShift
	MaxParticipants      *int32 // required when Type == EntryTypeShift
}

// UpdateEntryParams holds the input for UpdateEntry.
type UpdateEntryParams struct {
	ID               int64
	Name             string
	StartsAt         time.Time
	EndsAt           time.Time
	Location         *string
	Description      *string
	ResponseDeadline *time.Time
}

// --- Store methods ---

// GetEntryByID returns a single entry with calendar context.
// For shifts, it also fetches shift-specific details (required/max participants).
// Returns pgx.ErrNoRows if the entry does not exist.
func GetEntryByID(ctx context.Context, dbtx db.DBTX, id int64) (*model.Entry, error) {
	q := db.New(dbtx)
	row, err := q.GetEntryWithCalendar(ctx, id)
	if err != nil {
		return nil, err
	}
	entry := entryFromGetWithCalendarRow(row)

	if entry.Type == model.EntryTypeShift {
		if err := populateShiftDetails(ctx, q, &entry); err != nil {
			return nil, err
		}
	}

	return &entry, nil
}

// GetEntryBySlug returns a single entry by its slug with calendar context.
// For shifts, it also fetches shift-specific details.
// Returns pgx.ErrNoRows if the entry does not exist.
func GetEntryBySlug(ctx context.Context, dbtx db.DBTX, slug string) (*model.Entry, error) {
	q := db.New(dbtx)
	row, err := q.GetEntryWithCalendarBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	entry := entryFromGetWithCalendarBySlugRow(row)

	if entry.Type == model.EntryTypeShift {
		if err := populateShiftDetails(ctx, q, &entry); err != nil {
			return nil, err
		}
	}

	return &entry, nil
}

// GetEntryForUpdate returns an entry locked with SELECT FOR UPDATE.
// Must be called within a transaction (dbtx should be a pgx.Tx).
// Does NOT fetch shift details — the caller typically only needs
// the lock and basic entry data for attendance operations.
// Returns pgx.ErrNoRows if the entry does not exist.
func GetEntryForUpdate(ctx context.Context, dbtx db.DBTX, id int64) (*model.Entry, error) {
	q := db.New(dbtx)
	row, err := q.GetEntryForUpdate(ctx, id)
	if err != nil {
		return nil, err
	}
	entry := entryFromForUpdateRow(row)
	return &entry, nil
}

// CreateEntry inserts a new entry and its type-specific details.
// For shifts, it also inserts entry_shift_details and re-fetches the full model.
//
// IMPORTANT: For shift-type entries, the caller must pass a transaction-scoped
// DBTX (e.g., from Store.WithTx) to ensure atomicity of the multi-step operation.
func CreateEntry(ctx context.Context, dbtx db.DBTX, params CreateEntryParams) (*model.Entry, error) {
	q := db.New(dbtx)

	dbEntry, err := q.CreateEntry(ctx, db.CreateEntryParams{
		Slug:             params.Slug,
		CalendarID:       params.CalendarID,
		Name:             params.Name,
		Type:             string(params.Type),
		StartsAt:         timeToTs(params.StartsAt),
		EndsAt:           timeToTs(params.EndsAt),
		Location:         ptrToText(params.Location),
		Description:      ptrToText(params.Description),
		ResponseDeadline: ptrToTs(params.ResponseDeadline),
		RecurrenceRuleID: ptrToInt8(params.RecurrenceRuleID),
	})
	if err != nil {
		return nil, err
	}

	if params.Type == model.EntryTypeShift && params.RequiredParticipants != nil {
		maxP := int32(0)
		if params.MaxParticipants != nil {
			maxP = *params.MaxParticipants
		}
		_, err := q.UpsertEntryShiftDetails(ctx, db.UpsertEntryShiftDetailsParams{
			EntryID:              dbEntry.ID,
			RequiredParticipants: *params.RequiredParticipants,
			MaxParticipants:      maxP,
		})
		if err != nil {
			return nil, err
		}
	}

	return GetEntryByID(ctx, dbtx, dbEntry.ID)
}

// UpdateEntry updates an entry's mutable properties and returns the
// updated entry with calendar context.
// Returns pgx.ErrNoRows if the entry does not exist.
func UpdateEntry(ctx context.Context, dbtx db.DBTX, params UpdateEntryParams) (*model.Entry, error) {
	q := db.New(dbtx)

	dbEntry, err := q.UpdateEntry(ctx, db.UpdateEntryParams{
		ID:               params.ID,
		Name:             params.Name,
		StartsAt:         timeToTs(params.StartsAt),
		EndsAt:           timeToTs(params.EndsAt),
		Location:         ptrToText(params.Location),
		Description:      ptrToText(params.Description),
		ResponseDeadline: ptrToTs(params.ResponseDeadline),
	})
	if err != nil {
		return nil, err
	}

	return GetEntryByID(ctx, dbtx, dbEntry.ID)
}

// DeleteEntry removes an entry by ID. Attendance records cascade via FK.
func DeleteEntry(ctx context.Context, dbtx db.DBTX, id int64) error {
	return db.New(dbtx).DeleteEntry(ctx, id)
}

// ListEntriesByCalendar returns entries in a calendar within a date range,
// ordered by starts_at. Uses the idx_entries_calendar_starts composite index.
func ListEntriesByCalendar(ctx context.Context, dbtx db.DBTX, calendarID int64, start, end time.Time) ([]model.Entry, error) {
	q := db.New(dbtx)
	rows, err := q.ListEntriesByCalendarWithCalendar(ctx, db.ListEntriesByCalendarWithCalendarParams{
		CalendarID: calendarID,
		StartsAt:   timeToTs(start),
		StartsAt_2: timeToTs(end),
	})
	if err != nil {
		return nil, err
	}

	entries := make([]model.Entry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, entryFromCalendarWithCalendarRow(row))
	}
	return entries, nil
}

// ListEntriesByUnit returns entries across all of a unit's calendars
// within a date range, ordered by starts_at.
func ListEntriesByUnit(ctx context.Context, dbtx db.DBTX, unitID int64, start, end time.Time) ([]model.Entry, error) {
	q := db.New(dbtx)
	rows, err := q.ListEntriesByUnit(ctx, db.ListEntriesByUnitParams{
		UnitID:     unitID,
		StartsAt:   timeToTs(start),
		StartsAt_2: timeToTs(end),
	})
	if err != nil {
		return nil, err
	}

	entries := make([]model.Entry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, entryFromUnitRow(row))
	}
	return entries, nil
}

// ListEntriesByUser returns entries the user has accepted or is pending on,
// within a date range, ordered by starts_at. Powers the personal dashboard.
func ListEntriesByUser(ctx context.Context, dbtx db.DBTX, userID int64, start, end time.Time) ([]model.Entry, error) {
	q := db.New(dbtx)
	rows, err := q.ListEntriesByUser(ctx, db.ListEntriesByUserParams{
		UserID:     userID,
		StartsAt:   timeToTs(start),
		StartsAt_2: timeToTs(end),
	})
	if err != nil {
		return nil, err
	}

	entries := make([]model.Entry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, entryFromUserRow(row))
	}
	return entries, nil
}

// --- Internal helpers ---

// populateShiftDetails fetches and populates shift-specific fields on the entry.
// If no shift details row exists (pgx.ErrNoRows), the shift fields are left nil
// and no error is returned.
func populateShiftDetails(ctx context.Context, q *db.Queries, entry *model.Entry) error {
	details, err := q.GetEntryShiftDetails(ctx, entry.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // shift entry without details row, leave fields nil
		}
		return err
	}
	entry.RequiredParticipants = &details.RequiredParticipants
	entry.MaxParticipants = &details.MaxParticipants
	return nil
}
