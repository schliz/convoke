package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/schliz/convoke/internal/db"
)

// GetCalendarByID returns a single calendar by its primary key.
// Returns pgx.ErrNoRows if no calendar exists with the given ID.
func GetCalendarByID(ctx context.Context, dbtx db.DBTX, id int64) (db.Calendar, error) {
	return db.New(dbtx).GetCalendarByID(ctx, id)
}

// GetCalendarBySlug returns a single calendar by its URL slug.
// Returns pgx.ErrNoRows if no calendar exists with the given slug.
func GetCalendarBySlug(ctx context.Context, dbtx db.DBTX, slug string) (db.Calendar, error) {
	return db.New(dbtx).GetCalendarBySlug(ctx, slug)
}

// GetCalendarWithUnit returns a calendar with its owning unit's name and slug.
// Returns pgx.ErrNoRows if no calendar exists with the given ID.
func GetCalendarWithUnit(ctx context.Context, dbtx db.DBTX, id int64) (db.GetCalendarWithUnitRow, error) {
	return db.New(dbtx).GetCalendarWithUnit(ctx, id)
}

// ListCalendarsByUnit returns all calendars for a unit, ordered by sort_order
// then name.
func ListCalendarsByUnit(ctx context.Context, dbtx db.DBTX, unitID int64) ([]db.Calendar, error) {
	return db.New(dbtx).ListCalendarsByUnit(ctx, unitID)
}

// DeleteCalendar deletes a calendar by ID. Entries, custom viewers, and other
// dependent rows cascade via FK constraints in PostgreSQL.
func DeleteCalendar(ctx context.Context, dbtx db.DBTX, id int64) error {
	return db.New(dbtx).DeleteCalendar(ctx, id)
}

// ListVisibleCalendars returns calendars visible to a user. If isAdmin is true,
// returns all calendars (admin bypass). Otherwise evaluates visibility rules
// against the user's group memberships via the normalized junction tables.
func ListVisibleCalendars(ctx context.Context, dbtx db.DBTX, userID int64, isAdmin bool) ([]db.Calendar, error) {
	q := db.New(dbtx)
	if isAdmin {
		return q.ListAllCalendars(ctx)
	}
	return q.ListVisibleCalendarsForUser(ctx, userID)
}

// GetCustomViewerUnits returns the units that are custom viewers of a calendar.
func GetCustomViewerUnits(ctx context.Context, dbtx db.DBTX, calendarID int64) ([]db.GetCustomViewerUnitsRow, error) {
	return db.New(dbtx).GetCustomViewerUnits(ctx, calendarID)
}

// CreateCalendarWithViewers inserts a new calendar and, if visibility is
// 'custom', sets the custom viewer units. Runs in a transaction via Store.WithTx.
func (s *Store) CreateCalendarWithViewers(
	ctx context.Context,
	params db.CreateCalendarParams,
	customViewerUnitIDs []int64,
) (db.Calendar, error) {
	var cal db.Calendar

	err := s.WithTx(ctx, func(tx pgx.Tx, q *db.Queries) error {
		var txErr error
		cal, txErr = q.CreateCalendar(ctx, params)
		if txErr != nil {
			return fmt.Errorf("create calendar: %w", txErr)
		}

		if params.Visibility == "custom" {
			for _, unitID := range customViewerUnitIDs {
				if txErr = q.InsertCalendarCustomViewer(ctx, db.InsertCalendarCustomViewerParams{
					CalendarID: cal.ID,
					UnitID:     unitID,
				}); txErr != nil {
					return fmt.Errorf("insert custom viewer: %w", txErr)
				}
			}
		}

		return nil
	})

	return cal, err
}

// UpdateCalendarWithViewers updates calendar properties and replaces the custom
// viewer units. Always clears existing custom viewers, then re-inserts if
// visibility is 'custom'. Runs in a transaction via Store.WithTx.
func (s *Store) UpdateCalendarWithViewers(
	ctx context.Context,
	params db.UpdateCalendarParams,
	customViewerUnitIDs []int64,
) (db.Calendar, error) {
	var cal db.Calendar

	err := s.WithTx(ctx, func(tx pgx.Tx, q *db.Queries) error {
		var txErr error
		cal, txErr = q.UpdateCalendar(ctx, params)
		if txErr != nil {
			return fmt.Errorf("update calendar: %w", txErr)
		}

		// Always clear and re-set custom viewers (idempotent).
		if txErr = q.DeleteCalendarCustomViewers(ctx, cal.ID); txErr != nil {
			return fmt.Errorf("delete custom viewers: %w", txErr)
		}

		if params.Visibility == "custom" {
			for _, unitID := range customViewerUnitIDs {
				if txErr = q.InsertCalendarCustomViewer(ctx, db.InsertCalendarCustomViewerParams{
					CalendarID: cal.ID,
					UnitID:     unitID,
				}); txErr != nil {
					return fmt.Errorf("insert custom viewer: %w", txErr)
				}
			}
		}

		return nil
	})

	return cal, err
}
