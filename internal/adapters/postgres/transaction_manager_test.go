package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/adapters/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestTransactionManager tests the PostgreSQL transaction manager
func TestTransactionManager(t *testing.T) {
	// Skip if short flag is set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get PostgreSQL DSN from environment variable or use default
	pgDSN := os.Getenv("TEST_POSTGRES_DSN")
	if pgDSN == "" {
		// Try using the non-root user first
		username := os.Getenv("POSTGRESQL_USERNAME")
		password := os.Getenv("POSTGRESQL_PASSWORD")

		// Fall back to postgres user if non-root user is not set
		if username == "" {
			username = "postgres"
			password = os.Getenv("POSTGRESQL_POSTGRES_PASSWORD")

			// If password is still empty, use a default password for tests
			if password == "" {
				password = "NVsHFXcxqUsMoEgiUnE7jvzXxhp3gn9nsgkXCsetAHLhcpyLRmWhKixUpfr3J7tE"
			}
		}

		pgDSN = "postgres://" + username + ":" + password + "@localhost:5432/family_service_test?sslmode=disable"
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to PostgreSQL
	config, err := pgxpool.ParseConfig(pgDSN)
	require.NoError(t, err, "Failed to parse PostgreSQL config")

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	defer pool.Close()

	// Create logger
	logger := zaptest.NewLogger(t)

	// Create transaction manager
	txManager := postgres.NewTransactionManager(pool, logger)
	require.NotNil(t, txManager, "Transaction manager should not be nil")

	// Test BeginTx
	t.Run("BeginTx", func(t *testing.T) {
		// Begin transaction
		txCtx, err := txManager.BeginTx(ctx)
		require.NoError(t, err, "Failed to begin transaction")
		require.NotNil(t, txCtx, "Transaction context should not be nil")

		// Verify transaction exists in context
		tx := postgres.GetTx(txCtx)
		require.NotNil(t, tx, "Transaction should exist in context")

		// Rollback transaction
		err = txManager.RollbackTx(txCtx)
		require.NoError(t, err, "Failed to rollback transaction")
	})

	// Test BeginTx with existing transaction
	t.Run("BeginTxWithExistingTransaction", func(t *testing.T) {
		// Begin first transaction
		txCtx, err := txManager.BeginTx(ctx)
		require.NoError(t, err, "Failed to begin first transaction")

		// Begin second transaction with same context
		txCtx2, err := txManager.BeginTx(txCtx)
		require.NoError(t, err, "Failed to begin second transaction")
		assert.Equal(t, txCtx, txCtx2, "Transaction contexts should be the same")

		// Rollback transaction
		err = txManager.RollbackTx(txCtx)
		require.NoError(t, err, "Failed to rollback transaction")
	})

	// Test CommitTx
	t.Run("CommitTx", func(t *testing.T) {
		// Begin transaction
		txCtx, err := txManager.BeginTx(ctx)
		require.NoError(t, err, "Failed to begin transaction")

		// Create a test table
		_, err = postgres.GetTx(txCtx).Exec(txCtx, `
			CREATE TABLE IF NOT EXISTS test_commit (
				id SERIAL PRIMARY KEY,
				value TEXT
			)
		`)
		require.NoError(t, err, "Failed to create test table")

		// Insert a row
		_, err = postgres.GetTx(txCtx).Exec(txCtx, `
			INSERT INTO test_commit (value) VALUES ('test')
		`)
		require.NoError(t, err, "Failed to insert row")

		// Commit transaction
		err = txManager.CommitTx(txCtx)
		require.NoError(t, err, "Failed to commit transaction")

		// Verify row exists
		var count int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM test_commit WHERE value = 'test'`).Scan(&count)
		require.NoError(t, err, "Failed to count rows")
		assert.Equal(t, 1, count, "Row should exist after commit")

		// Clean up
		_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS test_commit`)
		require.NoError(t, err, "Failed to drop test table")
	})

	// Test RollbackTx
	t.Run("RollbackTx", func(t *testing.T) {
		// Begin transaction
		txCtx, err := txManager.BeginTx(ctx)
		require.NoError(t, err, "Failed to begin transaction")

		// Create a test table
		_, err = postgres.GetTx(txCtx).Exec(txCtx, `
			CREATE TABLE IF NOT EXISTS test_rollback (
				id SERIAL PRIMARY KEY,
				value TEXT
			)
		`)
		require.NoError(t, err, "Failed to create test table")

		// Insert a row
		_, err = postgres.GetTx(txCtx).Exec(txCtx, `
			INSERT INTO test_rollback (value) VALUES ('test')
		`)
		require.NoError(t, err, "Failed to insert row")

		// Rollback transaction
		err = txManager.RollbackTx(txCtx)
		require.NoError(t, err, "Failed to rollback transaction")

		// Verify table doesn't exist (should be rolled back)
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'test_rollback'
			)
		`).Scan(&exists)
		require.NoError(t, err, "Failed to check if table exists")
		assert.False(t, exists, "Table should not exist after rollback")
	})

	// Test CommitTx with no transaction
	t.Run("CommitTxWithNoTransaction", func(t *testing.T) {
		// Commit with no transaction
		err := txManager.CommitTx(ctx)
		require.Error(t, err, "Expected error when committing with no transaction")
		assert.Contains(t, err.Error(), "no transaction found in context")
	})

	// Test RollbackTx with no transaction
	t.Run("RollbackTxWithNoTransaction", func(t *testing.T) {
		// Rollback with no transaction
		err := txManager.RollbackTx(ctx)
		require.Error(t, err, "Expected error when rolling back with no transaction")
		assert.Contains(t, err.Error(), "no transaction found in context")
	})

	// Test WithTx success
	t.Run("WithTxSuccess", func(t *testing.T) {
		// Create a test table
		_, err = pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS test_with_tx_success (
				id SERIAL PRIMARY KEY,
				value TEXT
			)
		`)
		require.NoError(t, err, "Failed to create test table")

		// Execute function within transaction
		err = txManager.WithTx(ctx, func(txCtx context.Context) error {
			// Insert a row
			_, err := postgres.GetTx(txCtx).Exec(txCtx, `
				INSERT INTO test_with_tx_success (value) VALUES ('test')
			`)
			return err
		})
		require.NoError(t, err, "Failed to execute function within transaction")

		// Verify row exists
		var count int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM test_with_tx_success WHERE value = 'test'`).Scan(&count)
		require.NoError(t, err, "Failed to count rows")
		assert.Equal(t, 1, count, "Row should exist after successful transaction")

		// Clean up
		_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS test_with_tx_success`)
		require.NoError(t, err, "Failed to drop test table")
	})

	// Test WithTx error
	t.Run("WithTxError", func(t *testing.T) {
		// Create a test table
		_, err = pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS test_with_tx_error (
				id SERIAL PRIMARY KEY,
				value TEXT
			)
		`)
		require.NoError(t, err, "Failed to create test table")

		// Execute function within transaction that returns an error
		err = txManager.WithTx(ctx, func(txCtx context.Context) error {
			// Insert a row
			_, err := postgres.GetTx(txCtx).Exec(txCtx, `
				INSERT INTO test_with_tx_error (value) VALUES ('test')
			`)
			require.NoError(t, err, "Failed to insert row")

			// Return an error to trigger rollback
			return errors.New("test error")
		})
		require.Error(t, err, "Expected error from function within transaction")
		assert.Contains(t, err.Error(), "test error")

		// Verify row doesn't exist (should be rolled back)
		var count int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM test_with_tx_error WHERE value = 'test'`).Scan(&count)
		require.NoError(t, err, "Failed to count rows")
		assert.Equal(t, 0, count, "Row should not exist after failed transaction")

		// Clean up
		_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS test_with_tx_error`)
		require.NoError(t, err, "Failed to drop test table")
	})

	// Test WithTx panic
	t.Run("WithTxPanic", func(t *testing.T) {
		// Create a test table
		_, err = pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS test_with_tx_panic (
				id SERIAL PRIMARY KEY,
				value TEXT
			)
		`)
		require.NoError(t, err, "Failed to create test table")

		// Execute function within transaction that panics
		assert.Panics(t, func() {
			_ = txManager.WithTx(ctx, func(txCtx context.Context) error {
				// Insert a row
				_, err := postgres.GetTx(txCtx).Exec(txCtx, `
					INSERT INTO test_with_tx_panic (value) VALUES ('test')
				`)
				require.NoError(t, err, "Failed to insert row")

				// Panic to trigger rollback
				panic("test panic")
			})
		}, "Expected panic from function within transaction")

		// Verify row doesn't exist (should be rolled back)
		var count int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM test_with_tx_panic WHERE value = 'test'`).Scan(&count)
		require.NoError(t, err, "Failed to count rows")
		assert.Equal(t, 0, count, "Row should not exist after panicked transaction")

		// Clean up
		_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS test_with_tx_panic`)
		require.NoError(t, err, "Failed to drop test table")
	})
}
