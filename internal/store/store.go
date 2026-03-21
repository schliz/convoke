package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/schliz/convoke/internal/db"
)

// Pool is the minimal interface on the connection pool required by Store.
// Both *pgxpool.Pool and pgxmock.PgxPoolIface satisfy this interface.
type Pool interface {
	db.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Store struct {
	pool    Pool
	queries *db.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{
		pool:    pool,
		queries: db.New(pool),
	}
}

// NewWithPool creates a Store from any Pool implementation.
// This is useful for testing with pgxmock.
func NewWithPool(pool Pool) *Store {
	return &Store{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Queries returns the sqlc-generated query functions for non-transactional use.
func (s *Store) Queries() *db.Queries {
	return s.queries
}

// DB returns the underlying pool as a DBTX for non-transactional queries.
func (s *Store) DB() db.DBTX {
	return s.pool
}

// WithTx wraps a function in a transaction with automatic rollback on error/panic.
// Inside the callback, use db.New(tx) to get a transaction-scoped *db.Queries.
func (s *Store) WithTx(ctx context.Context, fn func(tx pgx.Tx, q *db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()
	if err := fn(tx, db.New(tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}
