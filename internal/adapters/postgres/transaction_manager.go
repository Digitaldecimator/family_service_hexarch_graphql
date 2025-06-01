package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// contextKey is a private type for context keys
type contextKey int

const (
	// txKey is the key for transaction value in the context
	txKey contextKey = iota
	// originalCtxKey is the key for the original context
	originalCtxKey
)

// TransactionManager implements the ports.TransactionManager interface for PostgreSQL
type TransactionManager struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
	tracer trace.Tracer
}

// NewTransactionManager creates a new PostgreSQL transaction manager
func NewTransactionManager(pool *pgxpool.Pool, logger *zap.Logger) *TransactionManager {
	return &TransactionManager{
		pool:   pool,
		logger: logger,
		tracer: otel.Tracer("postgres.transaction_manager"),
	}
}

// BeginTx begins a new transaction and stores it in the context
func (tm *TransactionManager) BeginTx(ctx context.Context) (context.Context, error) {
	// Store the original context
	originalCtx := ctx

	// Start a new span
	ctx, span := tm.tracer.Start(ctx, "TransactionManager.BeginTx")
	defer span.End()

	// Check if there's already a transaction in the context
	if tx := getTx(ctx); tx != nil {
		tm.logger.Debug("Transaction already exists in context")
		// Return the original context
		return originalCtx, nil
	}

	// Begin a new transaction
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		tm.logger.Error("Failed to begin transaction", zap.Error(err))
		return ctx, fmt.Errorf("failed to begin transaction: %w", err)
	}

	tm.logger.Debug("Transaction started")
	return context.WithValue(ctx, txKey, tx), nil
}

// CommitTx commits the transaction stored in the context
func (tm *TransactionManager) CommitTx(ctx context.Context) error {
	ctx, span := tm.tracer.Start(ctx, "TransactionManager.CommitTx")
	defer span.End()

	tx := getTx(ctx)
	if tx == nil {
		tm.logger.Debug("No transaction found in context")
		return fmt.Errorf("no transaction found in context")
	}

	if err := tx.Commit(ctx); err != nil {
		tm.logger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tm.logger.Debug("Transaction committed")
	return nil
}

// RollbackTx rolls back the transaction stored in the context
func (tm *TransactionManager) RollbackTx(ctx context.Context) error {
	ctx, span := tm.tracer.Start(ctx, "TransactionManager.RollbackTx")
	defer span.End()

	tx := getTx(ctx)
	if tx == nil {
		tm.logger.Debug("No transaction found in context")
		return fmt.Errorf("no transaction found in context")
	}

	if err := tx.Rollback(ctx); err != nil {
		tm.logger.Error("Failed to rollback transaction", zap.Error(err))
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	tm.logger.Debug("Transaction rolled back")
	return nil
}

// getTx retrieves the transaction from the context
func getTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey).(pgx.Tx); ok {
		return tx
	}
	return nil
}

// GetTx is a helper function to get the transaction from the context
// This is used by the repositories to get the transaction
func GetTx(ctx context.Context) pgx.Tx {
	return getTx(ctx)
}

// WithTx is a helper function to execute a function within a transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (tm *TransactionManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, span := tm.tracer.Start(ctx, "TransactionManager.WithTx")
	defer span.End()

	// Begin transaction
	ctx, err := tm.BeginTx(ctx)
	if err != nil {
		return err
	}

	// Ensure transaction is rolled back on panic
	defer func() {
		if r := recover(); r != nil {
			tm.logger.Error("Panic in transaction", zap.Any("recover", r))
			_ = tm.RollbackTx(ctx)
			panic(r) // Re-throw panic after rollback
		}
	}()

	// Execute the function
	if err := fn(ctx); err != nil {
		// Rollback transaction on error
		rollbackErr := tm.RollbackTx(ctx)
		if rollbackErr != nil {
			tm.logger.Error("Failed to rollback transaction after error",
				zap.Error(rollbackErr),
				zap.Error(err))
		}
		return err
	}

	// Commit transaction
	if err := tm.CommitTx(ctx); err != nil {
		return err
	}

	return nil
}
