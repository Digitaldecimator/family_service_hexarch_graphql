package migrations

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// InitialSchemaMigration creates the initial schema and sample data
type InitialSchemaMigration struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewInitialSchemaMigration creates a new initial schema migration
func NewInitialSchemaMigration(pool *pgxpool.Pool, logger *zap.Logger) *InitialSchemaMigration {
	return &InitialSchemaMigration{
		pool:   pool,
		logger: logger,
	}
}

// Up runs the migration
func (m *InitialSchemaMigration) Up(ctx context.Context) error {
	m.logger.Info("Running initial schema migration for PostgreSQL")

	// Create tables
	err := m.createTables(ctx)
	if err != nil {
		m.logger.Error("Failed to create tables", zap.Error(err))
		return err
	}

	// Insert sample data
	err = m.insertSampleData(ctx)
	if err != nil {
		m.logger.Error("Failed to insert sample data", zap.Error(err))
		return err
	}

	m.logger.Info("Initial schema migration for PostgreSQL completed successfully")
	return nil
}

// Down rolls back the migration
func (m *InitialSchemaMigration) Down(ctx context.Context) error {
	m.logger.Info("Rolling back initial schema migration for PostgreSQL")

	// Drop tables
	dropTablesSQL := `
		DROP TABLE IF EXISTS children;
		DROP TABLE IF EXISTS parents;
	`

	_, err := m.pool.Exec(ctx, dropTablesSQL)
	if err != nil {
		m.logger.Error("Failed to drop tables", zap.Error(err))
		return err
	}

	m.logger.Info("Initial schema migration for PostgreSQL rolled back successfully")
	return nil
}

// createTables creates the database tables
func (m *InitialSchemaMigration) createTables(ctx context.Context) error {
	// Create tables
	createTablesSQL := `
		CREATE TABLE IF NOT EXISTS parents (
			id UUID PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			email TEXT NOT NULL,
			birth_date TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			deleted_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS children (
			id UUID PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			birth_date TIMESTAMP NOT NULL,
			parent_id UUID NOT NULL REFERENCES parents(id),
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			deleted_at TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_parents_deleted_at ON parents(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_parents_email ON parents(email);
		CREATE INDEX IF NOT EXISTS idx_children_deleted_at ON children(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_children_parent_id ON children(parent_id);
	`

	_, err := m.pool.Exec(ctx, createTablesSQL)
	if err != nil {
		m.logger.Error("Failed to create tables", zap.Error(err))
		return err
	}

	return nil
}

// insertSampleData inserts sample data into the database
func (m *InitialSchemaMigration) insertSampleData(ctx context.Context) error {
	// Insert sample parents
	parents := []struct {
		ID        uuid.UUID
		FirstName string
		LastName  string
		Email     string
		BirthDate time.Time
	}{
		{
			ID:        uuid.New(),
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@example.com",
			BirthDate: parseDate("1980-01-15"),
		},
		{
			ID:        uuid.New(),
			FirstName: "Jane",
			LastName:  "Smith",
			Email:     "jane.smith@example.com",
			BirthDate: parseDate("1985-05-20"),
		},
		{
			ID:        uuid.New(),
			FirstName: "Bob",
			LastName:  "Johnson",
			Email:     "bob.johnson@example.com",
			BirthDate: parseDate("1975-11-08"),
		},
	}

	// Begin transaction
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		m.logger.Error("Failed to begin transaction", zap.Error(err))
		return err
	}
	defer tx.Rollback(ctx)

	// Insert parents
	now := time.Now().UTC()
	for _, parent := range parents {
		_, err := tx.Exec(ctx, `
			INSERT INTO parents (id, first_name, last_name, email, birth_date, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, parent.ID, parent.FirstName, parent.LastName, parent.Email, parent.BirthDate, now, now)
		if err != nil {
			m.logger.Error("Failed to insert parent", zap.Error(err), zap.String("email", parent.Email))
			return err
		}

		// Insert 2 children for each parent
		child1ID := uuid.New()
		child1BirthDate := parseDate("2010-03-12")
		_, err = tx.Exec(ctx, `
			INSERT INTO children (id, first_name, last_name, birth_date, parent_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, child1ID, "Child1", parent.LastName, child1BirthDate, parent.ID, now, now)
		if err != nil {
			m.logger.Error("Failed to insert child", zap.Error(err))
			return err
		}

		child2ID := uuid.New()
		child2BirthDate := parseDate("2012-07-25")
		_, err = tx.Exec(ctx, `
			INSERT INTO children (id, first_name, last_name, birth_date, parent_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, child2ID, "Child2", parent.LastName, child2BirthDate, parent.ID, now, now)
		if err != nil {
			m.logger.Error("Failed to insert child", zap.Error(err))
			return err
		}
	}

	// Commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		m.logger.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	m.logger.Info("Sample data inserted successfully",
		zap.Int("parents", len(parents)),
		zap.Int("children", len(parents)*2))
	return nil
}

// Helper function to parse date stringutil
func parseDate(dateStr string) time.Time {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		// Return a default date if parsing fails
		// TODO: Replace with localized logging once the localization system is accessible here
		// Message ID: system.date.parseError
		fmt.Printf("Error parsing date %s: %v\n", dateStr, err)
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return date
}
