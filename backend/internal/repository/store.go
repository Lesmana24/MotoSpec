package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/motospec/backend/internal/repository/db"
)

// Store provides all functions to execute db queries and transactions.
type Store interface {
	db.Querier
	ExecTx(ctx context.Context, fn func(*db.Queries) error) error
	Pool() *pgxpool.Pool
}

// SQLStore implements the Store interface using pgxpool.Pool and sqlc generated Queries.
type SQLStore struct {
	*db.Queries
	pool *pgxpool.Pool
}

var _ Store = (*SQLStore)(nil)

// NewStore creates a new SQLStore instance.
func NewStore(pool *pgxpool.Pool) *SQLStore {
	return &SQLStore{
		Queries: db.New(pool),
		pool:    pool,
	}
}

// ExecTx executes a function within a database transaction.
func (s *SQLStore) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := s.Queries.WithTx(tx)
	if err := fn(q); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Pool returns the underlying pgxpool.Pool (useful for health checks or pinging).
func (s *SQLStore) Pool() *pgxpool.Pool {
	return s.pool
}
