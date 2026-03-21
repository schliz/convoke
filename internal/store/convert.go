// Package store provides database access methods that wrap sqlc-generated queries.
//
// Error conventions:
//   - Pass-through methods (thin sqlc wrappers) propagate errors as-is.
//   - Composition methods (multi-step operations) wrap with fmt.Errorf("operation: %w", err).
package store

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// --- pgtype conversion helpers ---
//
// These convert between Go types and pgx/pgtype types used by the
// sqlc-generated code. They are intentionally simple and have no
// error paths.

func textToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func ptrToText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func tsToTime(ts pgtype.Timestamptz) time.Time {
	return ts.Time
}

func tsToPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	return &ts.Time
}

func ptrToTs(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func int8ToPtr(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}
	return &i.Int64
}

func ptrToInt8(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *i, Valid: true}
}

func timeToTs(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
