// Package postgres provides PostgreSQL implementations of the repository interfaces.
// This file contains test helpers for PostgreSQL integration tests.
package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// SetupTestDatabase sets up a PostgreSQL connection for testing.
// It returns a connection pool, context, logger, and cleanup function.
// The cleanup function should be deferred to ensure proper resource cleanup.
func SetupTestDatabase(t *testing.T) (*pgxpool.Pool, context.Context, *zap.Logger, func()) {
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

	// Ensure SSL is properly disabled
	if pgDSN != "" {
		// Parse the DSN to handle SSL parameters properly
		if strings.Contains(pgDSN, "?") {
			// If there's already a query string, make sure we don't add duplicate parameters
			if !strings.Contains(pgDSN, "sslmode=") {
				pgDSN = pgDSN + "&sslmode=disable"
			}
		} else {
			// If there's no query string yet, add one with SSL disabled
			pgDSN = pgDSN + "?sslmode=disable"
		}
	}

	// Connect to PostgreSQL
	config, err := pgxpool.ParseConfig(pgDSN)
	require.NoError(t, err, "Failed to parse PostgreSQL config")

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err, "Failed to connect to PostgreSQL")

	// Ping the database to verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "Failed to ping PostgreSQL")

	// Create logger
	logger := zaptest.NewLogger(t)

	// Initialize schema using the repository factory's InitSchema method
	factory, err := NewRepositoryFactory(ctx, pgDSN, logger)
	require.NoError(t, err, "Failed to create repository factory")

	err = factory.InitSchema(ctx)
	require.NoError(t, err, "Failed to initialize schema")

	// Return cleanup function
	cleanup := func() {
		// Drop tables to clean up
		_, err := pool.Exec(ctx, `DROP TABLE IF EXISTS children`)
		if err != nil {
			t.Logf("Failed to drop children table: %v", err)
		}

		_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS parents`)
		if err != nil {
			t.Logf("Failed to drop parents table: %v", err)
		}

		// Close connection
		pool.Close()

		// Cancel context
		cancel()
	}

	return pool, ctx, logger, cleanup
}

// SetupTestRepositories sets up repositories for testing.
// It returns a repository factory, context, and cleanup function.
// The cleanup function should be deferred to ensure proper resource cleanup.
func SetupTestRepositories(t *testing.T) (ports.RepositoryFactory, context.Context, func()) {
	pool, ctx, logger, cleanup := SetupTestDatabase(t)

	// Create a new repository factory using the existing pool
	factory := &RepositoryFactory{
		pool:               pool,
		logger:             logger,
		transactionManager: NewTransactionManager(pool, logger),
		parentRepository:   NewParentRepository(pool, logger),
		childRepository:    NewChildRepository(pool, logger),
	}

	return factory, ctx, cleanup
}
