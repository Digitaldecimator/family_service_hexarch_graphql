package migrations

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Migration represents a database migration
type Migration struct {
	Version     int       `bson:"version"`
	Description string    `bson:"description"`
	AppliedAt   time.Time `bson:"applied_at"`
}

// MigrationFunc is a function that performs a migration
type MigrationFunc func(ctx context.Context, db *mongo.Database) error

// MigrationDefinition defines a migration with its version, description, and function
type MigrationDefinition struct {
	Version     int
	Description string
	Migrate     MigrationFunc
}

// MigrationManager manages database migrations
type MigrationManager struct {
	db         *mongo.Database
	logger     *zap.Logger
	migrations map[int]MigrationDefinition
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *mongo.Database, logger *zap.Logger) *MigrationManager {
	return &MigrationManager{
		db:         db,
		logger:     logger,
		migrations: make(map[int]MigrationDefinition),
	}
}

// RegisterMigration registers a migration
func (m *MigrationManager) RegisterMigration(version int, description string, migrate MigrationFunc) {
	m.migrations[version] = MigrationDefinition{
		Version:     version,
		Description: description,
		Migrate:     migrate,
	}
}

// EnsureMigrationsCollection ensures that the migrations collection exists
func (m *MigrationManager) EnsureMigrationsCollection(ctx context.Context) error {
	// Check if migrations collection exists
	collections, err := m.db.ListCollectionNames(ctx, bson.M{"name": "migrations"})
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	// If migrations collection doesn't exist, create it
	if len(collections) == 0 {
		// Create migrations collection
		err = m.db.CreateCollection(ctx, "migrations")
		if err != nil {
			return fmt.Errorf("failed to create migrations collection: %w", err)
		}

		// Create index on version field
		indexModel := mongo.IndexModel{
			Keys:    bson.D{{Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true),
		}

		_, err = m.db.Collection("migrations").Indexes().CreateOne(ctx, indexModel)
		if err != nil {
			return fmt.Errorf("failed to create index on migrations collection: %w", err)
		}
	}

	return nil
}

// GetAppliedMigrations gets all applied migrations
func (m *MigrationManager) GetAppliedMigrations(ctx context.Context) ([]Migration, error) {
	// Get all migrations from the migrations collection
	cursor, err := m.db.Collection("migrations").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode migrations
	var migrations []Migration
	if err := cursor.All(ctx, &migrations); err != nil {
		return nil, fmt.Errorf("failed to decode migrations: %w", err)
	}

	return migrations, nil
}

// MigrateUp applies all pending migrations
func (m *MigrationManager) MigrateUp(ctx context.Context) error {
	// Ensure migrations collection exists
	if err := m.EnsureMigrationsCollection(ctx); err != nil {
		return err
	}

	// Get applied migrations
	appliedMigrations, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Create a map of applied migrations for quick lookup
	appliedMap := make(map[int]bool)
	for _, migration := range appliedMigrations {
		appliedMap[migration.Version] = true
	}

	// Get all migration versions
	var versions []int
	for version := range m.migrations {
		versions = append(versions, version)
	}

	// Sort versions
	sort.Ints(versions)

	// Apply pending migrations
	for _, version := range versions {
		// Skip if already applied
		if appliedMap[version] {
			m.logger.Info("Migration already applied", zap.Int("version", version))
			continue
		}

		migration := m.migrations[version]
		m.logger.Info("Applying migration", zap.Int("version", version), zap.String("description", migration.Description))

		// Apply migration
		if err := migration.Migrate(ctx, m.db); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", version, err)
		}

		// Record migration
		_, err := m.db.Collection("migrations").InsertOne(ctx, Migration{
			Version:     version,
			Description: migration.Description,
			AppliedAt:   time.Now().UTC(),
		})
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %w", version, err)
		}

		m.logger.Info("Migration applied successfully", zap.Int("version", version))
	}

	return nil
}
