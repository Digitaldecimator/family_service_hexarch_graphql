// Package mongodb provides MongoDB implementations of the repository interfaces.
package mongodb

import (
	"context"
	"fmt"
	"github.com/abitofhelp/family_service_hexarch_graphql/pkg/stringutil"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ParentRepository implements the ports.ParentRepository interface for MongoDB.
// It provides methods for CRUD operations on parent entities and supports
// filtering, pagination, and sorting when listing parents.
type ParentRepository struct {
	collection *mongo.Collection // MongoDB collection for parent documents
	logger     *zap.Logger       // Logger for recording repository events
	tracer     trace.Tracer      // Tracer for OpenTelemetry tracing
}

// NewParentRepository creates a new MongoDB parent repository.
// It initializes the repository with the provided database connection and logger,
// and ensures that necessary indexes exist in the MongoDB collection.
//
// Parameters:
//   - ctx: Context for the initialization operations, can be nil (background context will be used)
//   - db: MongoDB database connection
//   - logger: Logger for recording repository events
//   - config: MongoDB configuration settings
//
// Returns:
//   - A pointer to a new ParentRepository instance
func NewParentRepository(ctx context.Context, db *mongo.Database, logger *zap.Logger, config ports.MongoDBConfig) *ParentRepository {
	// Validate context
	if ctx == nil {
		ctx = context.Background()
		logger.Warn("Nil context provided to NewParentRepository, using background context")
	}

	collection := db.Collection("parents")

	// Create indexes with timeout
	indexTimeout := config.GetIndexTimeout()
	indexCtx, cancel := context.WithTimeout(ctx, indexTimeout)
	defer cancel()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "deleted_at", Value: 1}},
		Options: options.Index().SetName("idx_parents_deleted_at"),
	}

	_, err := collection.Indexes().CreateOne(indexCtx, indexModel)
	if err != nil {
		// Check if the error is because the index already exists
		if mongo.IsDuplicateKeyError(err) || (err != nil && (stringutil.ContainsIgnoreCase(err.Error(), "index already exists with a different name") || stringutil.ContainsIgnoreCase(err.Error(), "index already exists"))) {
			// Log as info instead of error since this is not a critical issue
			logger.Info("Index already exists, skipping creation", zap.String("index", "idx_parents_deleted_at"))
		} else {
			// Log other errors as they might be critical
			logger.Error("Failed to create index", zap.Error(err))
		}
	}

	return &ParentRepository{
		collection: collection,
		logger:     logger,
		tracer:     otel.Tracer("mongodb.parent_repository"),
	}
}

// Create creates a new parent in the database.
// It inserts the parent document into the MongoDB collection.
//
// Parameters:
//   - ctx: Context for the database operation
//   - parent: The parent entity to create
//
// Returns:
//   - An error if the creation fails, or nil on success
func (r *ParentRepository) Create(ctx context.Context, parent *domain.Parent) error {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.Create")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parent.ID.String()))

	_, err := r.collection.InsertOne(ctx, parent)
	if err != nil {
		r.logger.Error("Failed to create parent", zap.Error(err), zap.String("parent_id", parent.ID.String()))
		return fmt.Errorf("parent.create.failed: %w", err)
	}

	return nil
}

// GetByID retrieves a parent by ID from the database.
// It only returns non-deleted parents (where deleted_at is nil).
//
// Parameters:
//   - ctx: Context for the database operation
//   - id: The UUID of the parent to retrieve
//
// Returns:
//   - The parent entity if found
//   - An error if the parent is not found or if retrieval fails
func (r *ParentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", id.String()))

	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}

	var parent domain.Parent
	err := r.collection.FindOne(ctx, filter).Decode(&parent)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			r.logger.Debug("Parent not found", zap.String("parent_id", id.String()))
			return nil, fmt.Errorf("parent not found")
		}
		r.logger.Error("Failed to get parent", zap.Error(err), zap.String("parent_id", id.String()))
		return nil, fmt.Errorf("parent.get.failed: %w", err)
	}

	return &parent, nil
}

// Update updates an existing parent in the database.
// It only updates non-deleted parents and sets the updated_at timestamp.
// After updating, it verifies the update by retrieving the updated document.
//
// Parameters:
//   - ctx: Context for the database operation
//   - parent: The parent entity with updated fields
//
// Returns:
//   - An error if the parent is not found or if the update fails, or nil on success
func (r *ParentRepository) Update(ctx context.Context, parent *domain.Parent) error {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.Update")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parent.ID.String()))

	parent.UpdatedAt = time.Now().UTC()

	filter := bson.M{
		"_id":        parent.ID,
		"deleted_at": nil,
	}

	// Log the parent before update
	r.logger.Debug("Updating parent",
		zap.String("parent_id", parent.ID.String()),
		zap.Int("children_count", len(parent.Children)),
		zap.Any("parent", parent))

	update := bson.M{
		"$set": bson.M{
			"firstName": parent.FirstName,
			"lastName":  parent.LastName,
			"email":     parent.Email,
			"birthDate": parent.BirthDate,
			"children":  parent.Children,
			"updatedAt": parent.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Failed to update parent", zap.Error(err), zap.String("parent_id", parent.ID.String()))
		return fmt.Errorf("parent.update.failed: %w", err)
	}

	if result.MatchedCount == 0 {
		r.logger.Debug("Parent not found for update", zap.String("parent_id", parent.ID.String()))
		return fmt.Errorf("parent not found")
	}

	// Verify the update
	var updatedParent domain.Parent
	err = r.collection.FindOne(ctx, filter).Decode(&updatedParent)
	if err != nil {
		r.logger.Error("Failed to verify parent update", zap.Error(err), zap.String("parent_id", parent.ID.String()))
	} else {
		r.logger.Debug("Parent updated successfully",
			zap.String("parent_id", parent.ID.String()),
			zap.Int("children_count", len(updatedParent.Children)),
			zap.Any("parent", updatedParent))
	}

	return nil
}

// Delete marks a parent as deleted in the database.
// It performs a soft delete by setting the deleted_at timestamp rather than removing the document.
// It also marks all children of the parent as deleted.
//
// Parameters:
//   - ctx: Context for the database operation
//   - id: The UUID of the parent to delete
//
// Returns:
//   - An error if the parent is not found or if the deletion fails, or nil on success
func (r *ParentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.Delete")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", id.String()))

	now := time.Now().UTC()

	// Mark parent as deleted
	parentFilter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}

	parentUpdate := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updatedAt":  now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, parentFilter, parentUpdate)
	if err != nil {
		r.logger.Error("Failed to delete parent", zap.Error(err), zap.String("parent_id", id.String()))
		return fmt.Errorf("parent.delete.failed: %w", err)
	}

	if result.MatchedCount == 0 {
		r.logger.Debug("Parent not found for deletion", zap.String("parent_id", id.String()))
		return fmt.Errorf("parent not found")
	}

	// Mark all children as deleted
	childrenFilter := bson.M{
		"parentId":   id,
		"deleted_at": nil,
	}

	childrenUpdate := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updatedAt":  now,
		},
	}

	db := r.collection.Database()
	childrenCollection := db.Collection("children")

	_, err = childrenCollection.UpdateMany(ctx, childrenFilter, childrenUpdate)
	if err != nil {
		r.logger.Error("Failed to delete children", zap.Error(err), zap.String("parent_id", id.String()))
		return fmt.Errorf("parent.children.delete.failed: %w", err)
	}

	return nil
}

// buildListFilter builds a MongoDB filter document for listing parents with filtering.
// It converts the generic FilterOptions into a MongoDB-specific filter document.
// The filter always excludes deleted parents (where deleted_at is not nil).
// It supports filtering by first name, last name, email, and age range.
//
// Parameters:
//   - filter: The generic filter options containing filter criteria
//
// Returns:
//   - A MongoDB filter document (bson.M) that can be used in queries
func (r *ParentRepository) buildListFilter(filter ports.FilterOptions) bson.M {
	mongoFilter := bson.M{
		"deleted_at": nil,
	}

	if filter.FirstName != "" {
		mongoFilter["firstName"] = bson.M{"$regex": filter.FirstName, "$options": "i"}
	}

	if filter.LastName != "" {
		mongoFilter["lastName"] = bson.M{"$regex": filter.LastName, "$options": "i"}
	}

	if filter.Email != "" {
		mongoFilter["email"] = bson.M{"$regex": filter.Email, "$options": "i"}
	}

	if filter.MinAge > 0 {
		mongoFilter["birthDate"] = bson.M{"$lte": time.Now().AddDate(-filter.MinAge, 0, 0)}
	}

	if filter.MaxAge > 0 {
		if _, ok := mongoFilter["birthDate"]; ok {
			mongoFilter["birthDate"].(bson.M)["$gte"] = time.Now().AddDate(-filter.MaxAge, 0, 0)
		} else {
			mongoFilter["birthDate"] = bson.M{"$gte": time.Now().AddDate(-filter.MaxAge, 0, 0)}
		}
	}

	return mongoFilter
}

// List retrieves a list of parents with pagination, filtering, and sorting.
// It performs the find and count operations concurrently for better performance.
// The method handles context cancellation at various stages of the operation.
//
// Parameters:
//   - ctx: Context for the database operations
//   - queryOptions: Options for filtering, sorting, and pagination
//
// Returns:
//   - A slice of parent entities matching the filter criteria
//   - Pagination information including total count and whether there are more pages
//   - An error if the operation fails, or nil on success
func (r *ParentRepository) List(ctx context.Context, queryOptions ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.List")
	defer span.End()

	// Check for context cancellation
	if ctx.Err() != nil {
		r.logger.Warn("Context already cancelled before listing parents", zap.Error(ctx.Err()))
		return nil, nil, domain.NewDatabaseError("list", "Parent", ctx.Err())
	}

	filter := r.buildListFilter(queryOptions.Filter)

	// Build sort options
	sortOptions := bson.D{}
	if queryOptions.Sort.Field != "" {
		direction := 1 // ascending
		if queryOptions.Sort.Direction == "desc" {
			direction = -1
		}

		var sortField string
		switch queryOptions.Sort.Field {
		case "firstName", "first_name":
			sortField = "firstName"
		case "lastName", "last_name":
			sortField = "lastName"
		case "email":
			sortField = "email"
		case "birthDate", "birth_date":
			sortField = "birthDate"
		case "createdAt", "created_at":
			sortField = "createdAt"
		case "updatedAt", "updated_at":
			sortField = "updatedAt"
		default:
			sortField = "createdAt"
		}

		sortOptions = append(sortOptions, bson.E{Key: sortField, Value: direction})
	} else {
		sortOptions = append(sortOptions, bson.E{Key: "createdAt", Value: -1})
	}

	// Build pagination options
	limit := int64(queryOptions.Pagination.PageSize)
	if limit <= 0 {
		limit = 10 // Default page size
	}

	skip := int64(queryOptions.Pagination.Page * queryOptions.Pagination.PageSize)
	if skip < 0 {
		skip = 0
	}

	findOpts := options.Find()
	findOpts.SetSort(sortOptions)
	findOpts.SetLimit(limit)
	findOpts.SetSkip(skip)

	// Create a context with cancellation for the goroutines
	// This ensures we can cancel the goroutines if one of them fails
	goCtx, goCancel := context.WithCancel(ctx)
	defer goCancel()

	// Create channels for concurrent operations
	type findResult struct {
		parents []*domain.Parent
		err     error
	}
	type countResult struct {
		count int64
		err   error
	}
	findCh := make(chan findResult, 1)
	countCh := make(chan countResult, 1)

	// Execute Find operation concurrently
	go func() {
		cursor, err := r.collection.Find(goCtx, filter, findOpts)
		if err != nil {
			if goCtx.Err() != nil {
				// Context was cancelled
				findCh <- findResult{nil, domain.NewDatabaseError("list", "Parent", goCtx.Err())}
			} else {
				findCh <- findResult{nil, domain.NewDatabaseError("list", "Parent", err)}
			}
			return
		}
		defer cursor.Close(goCtx)

		parents := []*domain.Parent{}
		if err := cursor.All(goCtx, &parents); err != nil {
			if goCtx.Err() != nil {
				// Context was cancelled
				findCh <- findResult{nil, domain.NewDatabaseError("decode", "Parent", goCtx.Err())}
			} else {
				findCh <- findResult{nil, domain.NewDatabaseError("decode", "Parent", err)}
			}
			return
		}
		findCh <- findResult{parents, nil}
	}()

	// Execute Count operation concurrently
	go func() {
		count, err := r.Count(goCtx, queryOptions.Filter)
		if err != nil {
			if goCtx.Err() != nil {
				// Context was cancelled
				countCh <- countResult{0, domain.NewDatabaseError("count", "Parent", goCtx.Err())}
			} else {
				countCh <- countResult{0, domain.NewDatabaseError("count", "Parent", err)}
			}
			return
		}
		countCh <- countResult{count, nil}
	}()

	// Wait for both operations to complete or context cancellation
	var findRes findResult
	var countRes countResult

	// Use a select with a done channel to handle context cancellation
	findDone := make(chan struct{})
	countDone := make(chan struct{})

	// Start a goroutine to receive find results
	go func() {
		findRes = <-findCh
		close(findDone)
	}()

	// Start a goroutine to receive count results
	go func() {
		countRes = <-countCh
		close(countDone)
	}()

	// Wait for both operations to complete or context cancellation
	select {
	case <-findDone:
		select {
		case <-countDone:
			// Both operations completed
		case <-ctx.Done():
			// Context cancelled while waiting for count
			goCancel() // Cancel the count operation
			r.logger.Warn("Context cancelled while waiting for count operation", zap.Error(ctx.Err()))
			return nil, nil, domain.NewDatabaseError("list", "Parent", ctx.Err())
		}
	case <-countDone:
		select {
		case <-findDone:
			// Both operations completed
		case <-ctx.Done():
			// Context cancelled while waiting for find
			goCancel() // Cancel the find operation
			r.logger.Warn("Context cancelled while waiting for find operation", zap.Error(ctx.Err()))
			return nil, nil, domain.NewDatabaseError("list", "Parent", ctx.Err())
		}
	case <-ctx.Done():
		// Context cancelled before either operation completed
		goCancel() // Cancel both operations
		r.logger.Warn("Context cancelled before list operations completed", zap.Error(ctx.Err()))
		return nil, nil, domain.NewDatabaseError("list", "Parent", ctx.Err())
	}

	// Check for errors
	if findRes.err != nil {
		r.logger.Error("Failed to list parents", zap.Error(findRes.err))
		return nil, nil, findRes.err
	}
	if countRes.err != nil {
		r.logger.Error("Failed to get total count", zap.Error(countRes.err))
		return nil, nil, countRes.err
	}

	pagedResult := &ports.PagedResult{
		TotalCount: countRes.count,
		Page:       queryOptions.Pagination.Page,
		PageSize:   int(limit),
		HasNext:    (skip + int64(len(findRes.parents))) < countRes.count,
	}

	return findRes.parents, pagedResult, nil
}

// Count returns the total count of parents matching the filter.
// It converts the generic filter options to a MongoDB filter and counts matching documents.
//
// Parameters:
//   - ctx: Context for the database operation
//   - filter: Filter options to apply when counting parents
//
// Returns:
//   - The count of parents matching the filter
//   - An error if the count operation fails, or nil on success
func (r *ParentRepository) Count(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.Count")
	defer span.End()

	mongoFilter := r.buildListFilter(filter)

	count, err := r.collection.CountDocuments(ctx, mongoFilter)
	if err != nil {
		r.logger.Error("Failed to count parents", zap.Error(err))
		return 0, fmt.Errorf("parent.count.failed: %w", err)
	}

	return count, nil
}
