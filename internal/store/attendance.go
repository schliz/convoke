package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// BatchSender is the minimal interface for sending batch operations.
// Satisfied by *pgxpool.Pool, pgx.Tx, pgx.Conn, and pgxmock.
type BatchSender interface {
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// BulkCreatePendingAttendance creates pending attendance records for multiple
// users on a single entry. Uses pgx batch operations for efficiency, sending
// all statements in a single network round-trip.
//
// Silently skips users who already have an attendance record (ON CONFLICT DO NOTHING).
// Returns the number of records actually inserted.
//
// Returns (0, nil) for nil or empty userIDs (no database round-trip).
func BulkCreatePendingAttendance(ctx context.Context, sender BatchSender, entryID int64, userIDs []int64) (int64, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, uid := range userIDs {
		batch.Queue(
			`INSERT INTO attendances (entry_id, user_id, status)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (entry_id, user_id) DO NOTHING`,
			entryID, uid, "pending",
		)
	}

	br := sender.SendBatch(ctx, batch)
	defer br.Close()

	var inserted int64
	for range userIDs {
		ct, err := br.Exec()
		if err != nil {
			return inserted, fmt.Errorf("bulk create attendance: %w", err)
		}
		inserted += ct.RowsAffected()
	}

	return inserted, nil
}
