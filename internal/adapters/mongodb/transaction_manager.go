// Package mongodb provides MongoDB implementations of the repository interfaces.
package mongodb

import (
	"context"
	"fmt"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// contextKey is a private type for context keys used to store and retrieve values from a context.
type contextKey int

const (
	// sessionKey is the key for storing and retrieving the MongoDB session value in the context.
	sessionKey contextKey = iota
)

// TransactionManager implements the ports.TransactionManager interface for MongoDB.
// It provides methods for managing database transactions including beginning,
// committing, and rolling back transactions.
type TransactionManager struct {
	client *mongo.Client // MongoDB client for database operations
	logger *zap.Logger   // Logger for recording transaction events
	tracer trace.Tracer  // Tracer for OpenTelemetry tracing
}

// NewTransactionManager creates a new MongoDB transaction manager.
// It initializes a transaction manager with the provided MongoDB client and logger.
//
// Parameters:
//   - client: A MongoDB client instance for database operations
//   - logger: A zap logger for logging transaction events
//
// Returns:
//   - A pointer to a new TransactionManager instance
func NewTransactionManager(client *mongo.Client, logger *zap.Logger) *TransactionManager {
	return &TransactionManager{
		client: client,
		logger: logger,
		tracer: otel.Tracer("mongodb.transaction_manager"),
	}
}

// BeginTx begins a new MongoDB transaction and stores the session in the context.
// If a session already exists in the context, it returns the context unchanged.
// The method handles context cancellation and session creation errors.
//
// Parameters:
//   - ctx: The context for the transaction, which may be cancelled
//
// Returns:
//   - A new context containing the MongoDB session
//   - An error if the transaction could not be started, or nil on success
func (tm *TransactionManager) BeginTx(ctx context.Context) (context.Context, error) {
	ctx, span := tm.tracer.Start(ctx, "TransactionManager.BeginTx")
	defer span.End()

	// Check for context cancellation
	if ctx.Err() != nil {
		tm.logger.Warn("Context already cancelled before beginning transaction", zap.Error(ctx.Err()))
		return ctx, domain.NewTransactionError("begin", ctx.Err())
	}

	// Check if there's already a session in the context
	if session := GetSession(ctx); session != nil {
		tm.logger.Debug("Session already exists in context")
		return ctx, nil
	}

	// Start a new session
	session, err := tm.client.StartSession()
	if err != nil {
		tm.logger.Error("Failed to start session", zap.Error(err))
		return ctx, domain.NewTransactionError("begin", err)
	}

	// Store session in context
	ctx = context.WithValue(ctx, sessionKey, session)

	// Start a transaction
	err = session.StartTransaction()
	if err != nil {
		session.EndSession(ctx)
		tm.logger.Error("Failed to start transaction", zap.Error(err))
		return ctx, domain.NewTransactionError("begin", err)
	}

	tm.logger.Debug("Transaction started")
	return ctx, nil
}

// CommitTx commits the MongoDB transaction stored in the context.
// It retrieves the session from the context, commits the transaction, and ends the session.
// The method handles context cancellation and missing session errors.
//
// Parameters:
//   - ctx: The context containing the MongoDB session
//
// Returns:
//   - An error if the transaction could not be committed, or nil on success
func (tm *TransactionManager) CommitTx(ctx context.Context) error {
	ctx, span := tm.tracer.Start(ctx, "TransactionManager.CommitTx")
	defer span.End()

	// Check for context cancellation
	if ctx.Err() != nil {
		tm.logger.Warn("Context already cancelled before committing transaction", zap.Error(ctx.Err()))
		return domain.NewTransactionError("commit", ctx.Err())
	}

	session := GetSession(ctx)
	if session == nil {
		tm.logger.Debug("No session found in context")
		return domain.NewTransactionError("commit", fmt.Errorf("no session found in context"))
	}

	// Commit the transaction
	err := session.CommitTransaction(ctx)
	if err != nil {
		tm.logger.Error("Failed to commit transaction", zap.Error(err))
		return domain.NewTransactionError("commit", err)
	}

	// End the session
	session.EndSession(ctx)
	tm.logger.Debug("Transaction committed and session ended")
	return nil
}

// RollbackTx rolls back the MongoDB transaction stored in the context.
// It retrieves the session from the context, aborts the transaction, and ends the session.
// Unlike CommitTx, this method will attempt to rollback even if the context is cancelled,
// using a background context if necessary.
//
// Parameters:
//   - ctx: The context containing the MongoDB session
//
// Returns:
//   - An error if the transaction could not be rolled back, or nil on success
func (tm *TransactionManager) RollbackTx(ctx context.Context) error {
	ctx, span := tm.tracer.Start(ctx, "TransactionManager.RollbackTx")
	defer span.End()

	// For rollback, we don't check context cancellation because we want to
	// attempt to rollback even if the context is cancelled

	session := GetSession(ctx)
	if session == nil {
		tm.logger.Debug("No session found in context")
		return domain.NewTransactionError("rollback", fmt.Errorf("no session found in context"))
	}

	// Create a background context for rollback if the original context is cancelled
	rollbackCtx := ctx
	if ctx.Err() != nil {
		tm.logger.Warn("Using background context for rollback because original context is cancelled",
			zap.Error(ctx.Err()))
		rollbackCtx = context.Background()
	}

	// Abort the transaction
	err := session.AbortTransaction(rollbackCtx)
	if err != nil {
		tm.logger.Error("Failed to abort transaction", zap.Error(err))
		return domain.NewTransactionError("rollback", err)
	}

	// End the session
	session.EndSession(rollbackCtx)
	tm.logger.Debug("Transaction aborted and session ended")
	return nil
}

// GetSession retrieves the MongoDB session from the context.
// It safely type-asserts the session value from the context.
//
// Parameters:
//   - ctx: The context that may contain a MongoDB session
//
// Returns:
//   - The MongoDB session if found, or nil if not found or if type assertion fails
func GetSession(ctx context.Context) mongo.Session {
	if session, ok := ctx.Value(sessionKey).(mongo.Session); ok {
		return session
	}
	return nil
}

// MustGetSession retrieves the MongoDB session from the context and returns an error if not found.
// This is a convenience function for cases where a session is required.
//
// Parameters:
//   - ctx: The context that should contain a MongoDB session
//
// Returns:
//   - The MongoDB session if found
//   - An error if no session is found in the context
func MustGetSession(ctx context.Context) (mongo.Session, error) {
	session := GetSession(ctx)
	if session == nil {
		return nil, fmt.Errorf("no session found in context")
	}
	return session, nil
}

// WithTx is a helper function to execute a function within a MongoDB transaction.
// It handles the entire transaction lifecycle:
// 1. Starting a session and transaction
// 2. Executing the provided function within the transaction
// 3. Committing the transaction if the function succeeds
// 4. Rolling back the transaction if the function fails
// 5. Properly handling context cancellation at any point
// 6. Ensuring the session is always ended, even if the context is cancelled
//
// Parameters:
//   - ctx: The context for the transaction, which may be cancelled
//   - fn: The function to execute within the transaction, which receives the session context
//
// Returns:
//   - The error from the function if it fails
//   - A transaction error if the transaction operations fail
//   - nil if the function and transaction operations succeed
func (tm *TransactionManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, span := tm.tracer.Start(ctx, "TransactionManager.WithTx")
	defer span.End()

	// Check for context cancellation
	if ctx.Err() != nil {
		tm.logger.Warn("Context already cancelled before beginning transaction", zap.Error(ctx.Err()))
		return domain.NewTransactionError("begin", ctx.Err())
	}

	// Start a session
	session, err := tm.client.StartSession()
	if err != nil {
		tm.logger.Error("Failed to start session", zap.Error(err))
		return domain.NewTransactionError("begin", err)
	}

	// Use a separate context for ending the session to ensure it happens even if the original context is cancelled
	defer func() {
		endCtx := ctx
		if ctx.Err() != nil {
			endCtx = context.Background()
		}
		session.EndSession(endCtx)
	}()

	// Execute the function within a transaction
	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// Start a transaction
		if err := session.StartTransaction(); err != nil {
			tm.logger.Error("Failed to start transaction", zap.Error(err))
			return domain.NewTransactionError("begin", err)
		}

		// Execute the function
		if err := fn(sc); err != nil {
			// Check if context was cancelled
			if sc.Err() != nil && err != sc.Err() {
				tm.logger.Warn("Context cancelled during transaction execution",
					zap.Error(sc.Err()),
					zap.Error(err))
			}

			// Abort the transaction
			// If context is cancelled, we still try to abort with the session context
			// MongoDB driver will handle this appropriately
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				tm.logger.Error("Failed to abort transaction after error",
					zap.Error(abortErr),
					zap.Error(err))
			}

			// Return the original error
			return err
		}

		// Check for context cancellation before commit
		if sc.Err() != nil {
			tm.logger.Warn("Context cancelled before committing transaction", zap.Error(sc.Err()))

			// Abort the transaction
			// Even with a cancelled context, we try to abort
			// MongoDB driver will handle this appropriately
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				tm.logger.Error("Failed to abort transaction after context cancellation", zap.Error(abortErr))
			}

			return domain.NewTransactionError("commit", sc.Err())
		}

		// Commit the transaction
		if err := session.CommitTransaction(sc); err != nil {
			tm.logger.Error("Failed to commit transaction", zap.Error(err))
			return domain.NewTransactionError("commit", err)
		}

		return nil
	})
}
