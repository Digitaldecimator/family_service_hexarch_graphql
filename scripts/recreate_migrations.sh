#!/bin/bash

# Script to recreate migration scripts for MongoDB and PostgreSQL

echo "Recreating migration scripts..."

# MongoDB
echo "Recreating MongoDB migration script..."
mkdir -p ./internal/adapters/mongodb/migrations/new

cat > ./internal/adapters/mongodb/migrations/new/001_initial_schema.go << 'EOF'
package migrations

import (
	"context"
	"time"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// InitialSchemaMigration creates the initial schema and sample data
type InitialSchemaMigration struct {
	db     *mongo.Database
	logger *zap.Logger
}

// NewInitialSchemaMigration creates a new initial schema migration
func NewInitialSchemaMigration(db *mongo.Database, logger *zap.Logger) *InitialSchemaMigration {
	return &InitialSchemaMigration{
		db:     db,
		logger: logger,
	}
}

// Up runs the migration
func (m *InitialSchemaMigration) Up(ctx context.Context) error {
	m.logger.Info("Running initial schema migration for MongoDB")

	// Create indexes for parents collection
	parentsCollection := m.db.Collection("parents")
	parentIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "deleted_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "email", Value: 1}},
		},
	}

	_, err := parentsCollection.Indexes().CreateMany(ctx, parentIndexes)
	if err != nil {
		m.logger.Error("Failed to create indexes for parents collection", zap.Error(err))
		return err
	}

	// Create indexes for children collection
	childrenCollection := m.db.Collection("children")
	childrenIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "deleted_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "parent_id", Value: 1}},
		},
	}

	_, err = childrenCollection.Indexes().CreateMany(ctx, childrenIndexes)
	if err != nil {
		m.logger.Error("Failed to create indexes for children collection", zap.Error(err))
		return err
	}

	// Insert sample data
	err = m.insertSampleData(ctx)
	if err != nil {
		m.logger.Error("Failed to insert sample data", zap.Error(err))
		return err
	}

	m.logger.Info("Initial schema migration for MongoDB completed successfully")
	return nil
}

// Down rolls back the migration
func (m *InitialSchemaMigration) Down(ctx context.Context) error {
	m.logger.Info("Rolling back initial schema migration for MongoDB")

	// Drop collections
	err := m.db.Collection("children").Drop(ctx)
	if err != nil {
		m.logger.Error("Failed to drop children collection", zap.Error(err))
		return err
	}

	err = m.db.Collection("parents").Drop(ctx)
	if err != nil {
		m.logger.Error("Failed to drop parents collection", zap.Error(err))
		return err
	}

	m.logger.Info("Initial schema migration for MongoDB rolled back successfully")
	return nil
}

// insertSampleData inserts sample data into the database
func (m *InitialSchemaMigration) insertSampleData(ctx context.Context) error {
	// Create sample parents
	parents := []interface{}{
		createSampleParent("John", "Doe", "john.doe@example.com", "1980-01-15"),
		createSampleParent("Jane", "Smith", "jane.smith@example.com", "1985-05-20"),
		createSampleParent("Bob", "Johnson", "bob.johnson@example.com", "1975-11-08"),
	}

	// Insert parents
	parentsCollection := m.db.Collection("parents")
	_, err := parentsCollection.InsertMany(ctx, parents)
	if err != nil {
		m.logger.Error("Failed to insert sample parents", zap.Error(err))
		return err
	}

	// Get the inserted parents to get their IDs
	cursor, err := parentsCollection.Find(ctx, bson.M{})
	if err != nil {
		m.logger.Error("Failed to find sample parents", zap.Error(err))
		return err
	}
	defer cursor.Close(ctx)

	var insertedParents []domain.Parent
	if err = cursor.All(ctx, &insertedParents); err != nil {
		m.logger.Error("Failed to decode sample parents", zap.Error(err))
		return err
	}

	// Create sample children for each parent
	var children []interface{}
	for _, parent := range insertedParents {
		// Add 2 children for each parent
		children = append(children, createSampleChild("Child1", parent.LastName, "2010-03-12", parent.ID))
		children = append(children, createSampleChild("Child2", parent.LastName, "2012-07-25", parent.ID))
	}

	// Insert children
	childrenCollection := m.db.Collection("children")
	_, err = childrenCollection.InsertMany(ctx, children)
	if err != nil {
		m.logger.Error("Failed to insert sample children", zap.Error(err))
		return err
	}

	m.logger.Info("Sample data inserted successfully",
		zap.Int("parents", len(parents)),
		zap.Int("children", len(children)))
	return nil
}

// Helper functions to create sample data

func createSampleParent(firstName, lastName, email, birthDateStr string) *domain.Parent {
	birthDate, _ := time.Parse("2006-01-02", birthDateStr)
	now := time.Now().UTC()
	return &domain.Parent{
		ID:        uuid.New(),
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		BirthDate: birthDate,
		Children:  []domain.Child{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func createSampleChild(firstName, lastName, birthDateStr string, parentID uuid.UUID) *domain.Child {
	birthDate, _ := time.Parse("2006-01-02", birthDateStr)
	now := time.Now().UTC()
	return &domain.Child{
		ID:        uuid.New(),
		FirstName: firstName,
		LastName:  lastName,
		BirthDate: birthDate,
		ParentID:  parentID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
EOF

# PostgreSQL
echo "Recreating PostgreSQL migration script..."
mkdir -p ./internal/adapters/postgres/migrations/new

cat > ./internal/adapters/postgres/migrations/new/001_initial_schema.go << 'EOF'
package migrations

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// InitialSchemaMigration represents the initial schema migration
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

	// Insert parents
	for _, parent := range parents {
		_, err := m.pool.Exec(ctx, `
			INSERT INTO parents (id, first_name, last_name, email, birth_date, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
			parent.ID,
			parent.FirstName,
			parent.LastName,
			parent.Email,
			parent.BirthDate,
			time.Now().UTC(),
			time.Now().UTC(),
		)
		if err != nil {
			m.logger.Error("Failed to insert parent", zap.Error(err))
			return fmt.Errorf("failed to insert parent: %w", err)
		}

		// Insert 2 children for each parent
		child1ID := uuid.New()
		_, err = m.pool.Exec(ctx, `
			INSERT INTO children (id, first_name, last_name, birth_date, parent_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
			child1ID,
			"Child1",
			parent.LastName,
			parseDate("2010-03-12"),
			parent.ID,
			time.Now().UTC(),
			time.Now().UTC(),
		)
		if err != nil {
			m.logger.Error("Failed to insert child1", zap.Error(err))
			return fmt.Errorf("failed to insert child1: %w", err)
		}

		child2ID := uuid.New()
		_, err = m.pool.Exec(ctx, `
			INSERT INTO children (id, first_name, last_name, birth_date, parent_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
			child2ID,
			"Child2",
			parent.LastName,
			parseDate("2012-07-25"),
			parent.ID,
			time.Now().UTC(),
			time.Now().UTC(),
		)
		if err != nil {
			m.logger.Error("Failed to insert child2", zap.Error(err))
			return fmt.Errorf("failed to insert child2: %w", err)
		}
	}

	m.logger.Info("Sample data inserted successfully",
		zap.Int("parents", len(parents)),
		zap.Int("children", len(parents)*2))
	return nil
}

// Helper function to parse date strings
func parseDate(dateStr string) time.Time {
	date, _ := time.Parse("2006-01-02", dateStr)
	return date
}
EOF

echo "Moving new migration scripts to replace old ones..."
mv ./internal/adapters/mongodb/migrations/new/001_initial_schema.go ./internal/adapters/mongodb/migrations/001_initial_schema.go
mv ./internal/adapters/postgres/migrations/new/001_initial_schema.go ./internal/adapters/postgres/migrations/001_initial_schema.go

echo "Migration scripts recreated successfully."