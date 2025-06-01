package migrations

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Registry manages the migrations using the MigrationManager
type Registry struct {
	manager *MigrationManager
	logger  *zap.Logger
}

// NewRegistry creates a new migration registry
func NewRegistry(pool *pgxpool.Pool, logger *zap.Logger) *Registry {
	registry := &Registry{
		manager: NewMigrationManager(pool, logger),
		logger:  logger,
	}

	// Register migrations
	registry.registerMigrations()

	return registry
}

// registerMigrations registers all migrations
func (r *Registry) registerMigrations() {
	// Register initial schema migration
	r.manager.RegisterMigration(1, "Initial schema and sample data", func(ctx context.Context, pool *pgxpool.Pool) error {
		migration := NewInitialSchemaMigration(pool, r.logger)
		return migration.Up(ctx)
	})

	// Add more migrations here as needed
}

// MigrateUp runs all migrations
func (r *Registry) MigrateUp(ctx context.Context) error {
	r.logger.Info("Running PostgreSQL migrations")
	err := r.manager.MigrateUp(ctx)
	if err != nil {
		r.logger.Error("PostgreSQL migrations failed", zap.Error(err))
		return err
	}
	r.logger.Info("PostgreSQL migrations completed successfully")
	return nil
}