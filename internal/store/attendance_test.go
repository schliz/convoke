package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// --- BulkCreatePendingAttendance ---

func TestBulkCreatePendingAttendance_InsertsMultipleRecords(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	entryID := int64(100)
	userIDs := []int64{1, 2, 3}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO attendances").
		WithArgs(entryID, int64(1), "pending").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	batch.ExpectExec("INSERT INTO attendances").
		WithArgs(entryID, int64(2), "pending").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	batch.ExpectExec("INSERT INTO attendances").
		WithArgs(entryID, int64(3), "pending").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	inserted, err := BulkCreatePendingAttendance(context.Background(), mock, entryID, userIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted != 3 {
		t.Errorf("expected 3 inserted, got %d", inserted)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestBulkCreatePendingAttendance_ReturnsZeroForEmptyUserIDs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// No expectations — should short-circuit without touching the database.

	inserted, err := BulkCreatePendingAttendance(context.Background(), mock, 100, []int64{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted != 0 {
		t.Errorf("expected 0 inserted, got %d", inserted)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestBulkCreatePendingAttendance_ReturnsZeroForNilUserIDs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// No expectations — should short-circuit without touching the database.

	inserted, err := BulkCreatePendingAttendance(context.Background(), mock, 100, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted != 0 {
		t.Errorf("expected 0 inserted, got %d", inserted)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestBulkCreatePendingAttendance_CountsSkippedDuplicates(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	entryID := int64(100)
	userIDs := []int64{1, 2, 3}

	// User 1 is new, user 2 already exists (0 rows affected), user 3 is new
	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO attendances").
		WithArgs(entryID, int64(1), "pending").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	batch.ExpectExec("INSERT INTO attendances").
		WithArgs(entryID, int64(2), "pending").
		WillReturnResult(pgxmock.NewResult("INSERT", 0))
	batch.ExpectExec("INSERT INTO attendances").
		WithArgs(entryID, int64(3), "pending").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	inserted, err := BulkCreatePendingAttendance(context.Background(), mock, entryID, userIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted != 2 {
		t.Errorf("expected 2 inserted (1 skipped duplicate), got %d", inserted)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestBulkCreatePendingAttendance_PropagatesError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	entryID := int64(100)
	userIDs := []int64{1, 2}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO attendances").
		WithArgs(entryID, int64(1), "pending").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	batch.ExpectExec("INSERT INTO attendances").
		WithArgs(entryID, int64(2), "pending").
		WillReturnError(fmt.Errorf("fk violation"))

	_, err = BulkCreatePendingAttendance(context.Background(), mock, entryID, userIDs)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestBulkCreatePendingAttendance_SingleUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	entryID := int64(50)
	userIDs := []int64{42}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO attendances").
		WithArgs(entryID, int64(42), "pending").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	inserted, err := BulkCreatePendingAttendance(context.Background(), mock, entryID, userIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", inserted)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
