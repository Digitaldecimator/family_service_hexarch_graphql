package migrations

import (
	"context"
	"time"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
			Keys:    bson.D{{Key: "deleted_at", Value: 1}},
			Options: options.Index().SetName("idx_parents_deleted_at"),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetName("idx_parents_email"),
		},
	}

	_, err := parentsCollection.Indexes().CreateMany(ctx, parentIndexes)
	if err != nil {
		m.logger.Error("Failed to create indexes for parents collection", zap.Error(err))
		return err
	}

	// Create indexes for children collection
	childrenCollection := m.db.Collection("children")

	// First, try to drop any existing index on parentId field to avoid conflicts
	_, err = childrenCollection.Indexes().DropOne(ctx, "idx_children_parent_id")
	if err != nil {
		// Ignore error if index doesn't exist
		m.logger.Info("No existing index idx_children_parent_id to drop, continuing")
	}

	// Also try to drop the index with the new name that's causing the conflict
	_, err = childrenCollection.Indexes().DropOne(ctx, "idx_children_parent_id_new")
	if err != nil {
		// Ignore error if index doesn't exist
		m.logger.Info("No existing index idx_children_parent_id_new to drop, continuing")
	}

	childrenIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "deleted_at", Value: 1}},
			Options: options.Index().SetName("idx_children_deleted_at"),
		},
		{
			Keys:    bson.D{{Key: "parentId", Value: 1}},
			Options: options.Index().SetName("idx_children_parent_id"),
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
