package migrations

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Migration represents a database migration
type Migration struct {
	Version     int       `json:"version"`
	Description string    `json:"description"`
	AppliedAt   time.Time `json:"applied_at"`
}

// MigrationFunc is a function that performs a migration
type MigrationFunc func(ctx context.Context, pool *pgxpool.Pool) error

// MigrationDefinition defines a migration with its version, description, and function
type MigrationDefinition struct {
	Version     int
	Description string
	Migrate     MigrationFunc
}

// MigrationManager manages database migrations
type MigrationManager struct {
	pool       *pgxpool.Pool
	logger     *zap.Logger
	migrations map[int]MigrationDefinition
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(pool *pgxpool.Pool, logger *zap.Logger) *MigrationManager {
	return &MigrationManager{
		pool:       pool,
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

// EnsureMigrationsTable ensures that the migrations table exists
func (m *MigrationManager) EnsureMigrationsTable(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS migrations (
			version INT PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL
		);
	`

	_, err := m.pool.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// GetAppliedMigrations gets all applied migrations
func (m *MigrationManager) GetAppliedMigrations(ctx context.Context) ([]Migration, error) {
	// Get all migrations from the migrations table
	rows, err := m.pool.Query(ctx, "SELECT version, description, applied_at FROM migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	defer rows.Close()

	// Decode migrations
	var migrations []Migration
	for rows.Next() {
		var migration Migration
		if err := rows.Scan(&migration.Version, &migration.Description, &migration.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to decode migration: %w", err)
		}
		migrations = append(migrations, migration)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migrations rows: %w", err)
	}

	return migrations, nil
}

// MigrateUp applies all pending migrations
func (m *MigrationManager) MigrateUp(ctx context.Context) error {
	// Ensure migrations table exists
	if err := m.EnsureMigrationsTable(ctx); err != nil {
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

		// Begin transaction
		tx, err := m.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", version, err)
		}

		// Apply migration
		if err := migration.Migrate(ctx, m.pool); err != nil {
			// Rollback transaction
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				m.logger.Error("Failed to rollback transaction", zap.Error(rbErr))
			}
			return fmt.Errorf("failed to apply migration %d: %w", version, err)
		}

		// Record migration
		_, err = tx.Exec(ctx, `
			INSERT INTO migrations (version, description, applied_at)
			VALUES ($1, $2, $3)
		`, version, migration.Description, time.Now().UTC())
		if err != nil {
			// Rollback transaction
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				m.logger.Error("Failed to rollback transaction", zap.Error(rbErr))
			}
			return fmt.Errorf("failed to record migration %d: %w", version, err)
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction for migration %d: %w", version, err)
		}

		m.logger.Info("Migration applied successfully", zap.Int("version", version))
	}

	return nil
}