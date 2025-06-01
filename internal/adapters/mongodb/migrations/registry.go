package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Registry manages the migrations using the MigrationManager
type Registry struct {
	manager *MigrationManager
	logger  *zap.Logger
}

// NewRegistry creates a new migration registry
func NewRegistry(db *mongo.Database, logger *zap.Logger) *Registry {
	registry := &Registry{
		manager: NewMigrationManager(db, logger),
		logger:  logger,
	}

	// Register migrations
	registry.registerMigrations()

	return registry
}

// registerMigrations registers all migrations
func (r *Registry) registerMigrations() {
	// Register initial schema migration
	r.manager.RegisterMigration(1, "Initial schema and sample data", func(ctx context.Context, db *mongo.Database) error {
		migration := NewInitialSchemaMigration(db, r.logger)
		return migration.Up(ctx)
	})

	// Add more migrations here as needed
}

// MigrateUp runs all migrations
func (r *Registry) MigrateUp(ctx context.Context) error {
	r.logger.Info("Running MongoDB migrations")
	err := r.manager.MigrateUp(ctx)
	if err != nil {
		r.logger.Error("MongoDB migrations failed", zap.Error(err))
		return err
	}
	r.logger.Info("MongoDB migrations completed successfully")
	return nil
}
