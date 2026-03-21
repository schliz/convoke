package model

import "time"

// EntryType represents the type of calendar entry.
type EntryType string

const (
	EntryTypeShift   EntryType = "shift"
	EntryTypeMeeting EntryType = "meeting"
)

// Entry is the domain-level entry type returned by store methods.
// It converts pgtype wrappers to ergonomic Go types and includes
// joined calendar context when available.
type Entry struct {
	ID               int64
	Slug             string
	CalendarID       int64
	Name             string
	Type             EntryType
	StartsAt         time.Time
	EndsAt           time.Time
	Location         *string
	Description      *string
	ResponseDeadline *time.Time
	RecurrenceRuleID *int64
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Joined calendar context (populated by methods that join calendar data).
	CalendarName  string
	CalendarSlug  string
	CalendarColor *string
	UnitID        int64

	// Shift-specific (populated when Type == "shift" and details are fetched).
	RequiredParticipants *int32
	MaxParticipants      *int32
}
