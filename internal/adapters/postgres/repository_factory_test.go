package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/adapters/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestRepositoryFactory tests the PostgreSQL repository factory
func TestRepositoryFactory(t *testing.T) {
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

	// Create logger
	logger := zaptest.NewLogger(t)

	// Test creating a repository factory
	t.Run("NewRepositoryFactory", func(t *testing.T) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create repository factory
		factory, err := postgres.NewRepositoryFactory(ctx, pgDSN, logger)
		require.NoError(t, err, "Failed to create repository factory")
		require.NotNil(t, factory, "Repository factory should not be nil")

		// Clean up
		err = factory.Close(ctx)
		require.NoError(t, err, "Failed to close repository factory")
	})

	// Test with nil context
	t.Run("NewRepositoryFactoryWithNilContext", func(t *testing.T) {
		// Create repository factory with nil context
		factory, err := postgres.NewRepositoryFactory(nil, pgDSN, logger)
		require.NoError(t, err, "Failed to create repository factory with nil context")
		require.NotNil(t, factory, "Repository factory should not be nil")

		// Clean up
		err = factory.Close(context.Background())
		require.NoError(t, err, "Failed to close repository factory")
	})

	// Test with invalid connection string
	t.Run("NewRepositoryFactoryWithInvalidConnString", func(t *testing.T) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create repository factory with invalid connection string
		factory, err := postgres.NewRepositoryFactory(ctx, "invalid-connection-string", logger)
		require.Error(t, err, "Expected error when creating repository factory with invalid connection string")
		require.Nil(t, factory, "Repository factory should be nil")
		assert.Contains(t, err.Error(), "failed to parse connection string")
	})

	// Test with valid connection string but unreachable database
	t.Run("NewRepositoryFactoryWithUnreachableDatabase", func(t *testing.T) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create repository factory with unreachable database
		factory, err := postgres.NewRepositoryFactory(ctx, "postgres://postgres:password@nonexistent-host:5432/nonexistent_db", logger)
		require.Error(t, err, "Expected error when creating repository factory with unreachable database")
		require.Nil(t, factory, "Repository factory should be nil")
	})

	// Test repository factory methods
	t.Run("RepositoryFactoryMethods", func(t *testing.T) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create repository factory
		factory, err := postgres.NewRepositoryFactory(ctx, pgDSN, logger)
		require.NoError(t, err, "Failed to create repository factory")
		require.NotNil(t, factory, "Repository factory should not be nil")

		// Test NewParentRepository
		parentRepo := factory.NewParentRepository()
		require.NotNil(t, parentRepo, "Parent repository should not be nil")

		// Test NewChildRepository
		childRepo := factory.NewChildRepository()
		require.NotNil(t, childRepo, "Child repository should not be nil")

		// Test GetTransactionManager
		txManager := factory.GetTransactionManager()
		require.NotNil(t, txManager, "Transaction manager should not be nil")

		// Clean up
		err = factory.Close(ctx)
		require.NoError(t, err, "Failed to close repository factory")
	})

	// Test InitSchema
	t.Run("InitSchema", func(t *testing.T) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create repository factory
		factory, err := postgres.NewRepositoryFactory(ctx, pgDSN, logger)
		require.NoError(t, err, "Failed to create repository factory")
		require.NotNil(t, factory, "Repository factory should not be nil")

		// Initialize schema
		err = factory.InitSchema(ctx)
		require.NoError(t, err, "Failed to initialize schema")

		// Verify tables exist by connecting directly to the database
		config, err := pgxpool.ParseConfig(pgDSN)
		require.NoError(t, err, "Failed to parse PostgreSQL config")

		pool, err := pgxpool.NewWithConfig(ctx, config)
		require.NoError(t, err, "Failed to connect to PostgreSQL")
		defer pool.Close()

		// Check if parents table exists
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'parents'
			)
		`).Scan(&exists)
		require.NoError(t, err, "Failed to check if parents table exists")
		assert.True(t, exists, "Parents table should exist")

		// Check if children table exists
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'children'
			)
		`).Scan(&exists)
		require.NoError(t, err, "Failed to check if children table exists")
		assert.True(t, exists, "Children table should exist")

		// Clean up
		_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS children`)
		require.NoError(t, err, "Failed to drop children table")

		_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS parents`)
		require.NoError(t, err, "Failed to drop parents table")

		err = factory.Close(ctx)
		require.NoError(t, err, "Failed to close repository factory")
	})

	// Test Close with nil context
	t.Run("CloseWithNilContext", func(t *testing.T) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create repository factory
		factory, err := postgres.NewRepositoryFactory(ctx, pgDSN, logger)
		require.NoError(t, err, "Failed to create repository factory")
		require.NotNil(t, factory, "Repository factory should not be nil")

		// Close with nil context
		err = factory.Close(nil)
		require.NoError(t, err, "Failed to close repository factory with nil context")
	})
}
