// Package mongodb provides MongoDB implementations of the repository interfaces defined in the ports package.
// It contains the data access logic for storing and retrieving domain entities in MongoDB,
// following the repository pattern. This package is part of the adapters layer in the
// hexagonal architecture, connecting the application core to the MongoDB database.
package mongodb

import (
	"context"
	"errors"
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

// ChildRepository implements the ports.ChildRepository interface for MongoDB.
// It provides methods for storing, retrieving, updating, and deleting Child entities
// in a MongoDB database, with support for tracing and logging.
type ChildRepository struct {
	collection *mongo.Collection // MongoDB collection for storing child documents
	logger     *zap.Logger       // Logger for recording repository operations
	tracer     trace.Tracer      // Tracer for distributed tracing
}

// NewChildRepository creates a new MongoDB child repository.
// It initializes the repository with the necessary dependencies and creates indexes
// on the children collection to optimize query performance.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - db: The MongoDB database connection
//   - logger: Logger for recording repository operations
//   - config: Configuration for MongoDB, including timeout settings
//
// Returns:
//   - *ChildRepository: A new instance of the child repository
func NewChildRepository(ctx context.Context, db *mongo.Database, logger *zap.Logger, config ports.MongoDBConfig) *ChildRepository {
	// Validate context
	if ctx == nil {
		ctx = context.Background()
		logger.Warn("Nil context provided to NewChildRepository, using background context")
	}

	collection := db.Collection("children")

	// Create indexes with timeout
	indexTimeout := config.GetIndexTimeout()
	indexCtx, cancel := context.WithTimeout(ctx, indexTimeout)
	defer cancel()

	// Create indexes one by one to handle conflicts
	deletedAtIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "deleted_at", Value: 1}},
		Options: options.Index().SetName("idx_children_deleted_at"),
	}

	_, err := collection.Indexes().CreateOne(indexCtx, deletedAtIndex)
	if err != nil {
		// Check if the error is because the index already exists
		if mongo.IsDuplicateKeyError(err) || (err != nil && (stringutil.ContainsIgnoreCase(err.Error(), "index already exists with a different name") || stringutil.ContainsIgnoreCase(err.Error(), "index already exists"))) {
			// Log as info instead of error since this is not a critical issue
			logger.Info("Index already exists, skipping creation", zap.String("index", "idx_children_deleted_at"))
		} else {
			// Log other errors as they might be critical
			logger.Error("Failed to create index", zap.Error(err), zap.String("index", "idx_children_deleted_at"))
		}
	}

	// Try to create parent_id index with the original name
	parentIdIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "parentId", Value: 1}},
		Options: options.Index().SetName("idx_children_parent_id_new"),
	}

	_, err = collection.Indexes().CreateOne(indexCtx, parentIdIndex)
	if err != nil {
		// Check if the error is because the index already exists
		if mongo.IsDuplicateKeyError(err) || (err != nil && (stringutil.ContainsIgnoreCase(err.Error(), "index already exists with a different name") || stringutil.ContainsIgnoreCase(err.Error(), "index already exists"))) {
			// Log as info instead of error since this is not a critical issue
			logger.Info("Index already exists, skipping creation", zap.String("index", "idx_children_parent_id_new"))
		} else {
			// Log other errors as they might be critical
			logger.Error("Failed to create index", zap.Error(err), zap.String("index", "idx_children_parent_id_new"))
		}
	}

	return &ChildRepository{
		collection: collection,
		logger:     logger,
		tracer:     otel.Tracer("mongodb.child_repository"),
	}
}

// Create creates a new child in the database.
// It first checks if the parent exists before creating the child to maintain referential integrity.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - child: The child entity to create in the database
//
// Returns:
//   - error: An error if the parent doesn't exist or if there's a database error
func (r *ChildRepository) Create(ctx context.Context, child *domain.Child) error {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("child.id", child.ID.String()),
		attribute.String("parent.id", child.ParentID.String()),
	)

	// First check if the parent exists
	db := r.collection.Database()
	parentsCollection := db.Collection("parents")

	parentFilter := bson.M{
		"_id":        child.ParentID,
		"deleted_at": nil,
	}

	var parentCount int64
	parentCount, err := parentsCollection.CountDocuments(ctx, parentFilter)
	if err != nil {
		r.logger.Error("Failed to check parent existence", zap.Error(err), zap.String("parent_id", child.ParentID.String()))
		return fmt.Errorf("child.parent.check.failed: %w", err)
	}

	if parentCount == 0 {
		r.logger.Debug("Parent not found for child creation", zap.String("parent_id", child.ParentID.String()))
		return errors.New("child.parent.notFound")
	}

	_, err = r.collection.InsertOne(ctx, child)
	if err != nil {
		r.logger.Error("Failed to create child", zap.Error(err), zap.String("child_id", child.ID.String()))
		return fmt.Errorf("child.create.failed: %w", err)
	}

	return nil
}

// GetByID retrieves a child by ID from the database.
// It only returns children that are not marked as deleted (soft delete).
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - id: The unique identifier of the child to retrieve
//
// Returns:
//   - *domain.Child: The retrieved child entity if found
//   - error: An error if the child is not found or if there's a database error
func (r *ChildRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("child.id", id.String()))

	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}

	var child domain.Child
	err := r.collection.FindOne(ctx, filter).Decode(&child)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			r.logger.Debug("Child not found", zap.String("child_id", id.String()))
			return nil, fmt.Errorf("child not found")
		}
		r.logger.Error("Failed to get child", zap.Error(err), zap.String("child_id", id.String()))
		return nil, fmt.Errorf("child.get.failed: %w", err)
	}

	return &child, nil
}

// Update updates an existing child in the database.
// It only updates children that are not marked as deleted (soft delete) and
// automatically updates the UpdatedAt timestamp.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - child: The child entity with updated information
//
// Returns:
//   - error: An error if the child is not found or if there's a database error
func (r *ChildRepository) Update(ctx context.Context, child *domain.Child) error {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.Update")
	defer span.End()

	span.SetAttributes(attribute.String("child.id", child.ID.String()))

	child.UpdatedAt = time.Now().UTC()

	filter := bson.M{
		"_id":        child.ID,
		"deleted_at": nil,
	}

	update := bson.M{
		"$set": bson.M{
			"firstName": child.FirstName,
			"lastName":  child.LastName,
			"birthDate": child.BirthDate,
			"updatedAt": child.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Failed to update child", zap.Error(err), zap.String("child_id", child.ID.String()))
		return fmt.Errorf("child.update.failed: %w", err)
	}

	if result.MatchedCount == 0 {
		r.logger.Debug("Child not found for update", zap.String("child_id", child.ID.String()))
		return fmt.Errorf("child not found")
	}

	return nil
}

// Delete marks a child as deleted in the database.
// This is a soft delete operation that sets the DeletedAt timestamp rather than
// removing the document from the database. It also updates the UpdatedAt timestamp.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - id: The unique identifier of the child to mark as deleted
//
// Returns:
//   - error: An error if the child is not found or if there's a database error
func (r *ChildRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.Delete")
	defer span.End()

	span.SetAttributes(attribute.String("child.id", id.String()))

	now := time.Now().UTC()

	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updatedAt":  now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Failed to delete child", zap.Error(err), zap.String("child_id", id.String()))
		return fmt.Errorf("child.delete.failed: %w", err)
	}

	if result.MatchedCount == 0 {
		r.logger.Debug("Child not found for deletion", zap.String("child_id", id.String()))
		return fmt.Errorf("child not found")
	}

	return nil
}

// buildListFilter builds a MongoDB filter document for listing children with filtering.
// It constructs a BSON filter based on the provided filter options and optional parent ID.
// The filter includes conditions for soft delete (deleted_at is nil) and supports
// filtering by first name, last name, and age range.
// Parameters:
//   - filter: The filter options containing criteria for filtering children
//   - parentID: Optional parent ID to filter children by parent
//
// Returns:
//   - bson.M: A MongoDB filter document that can be used in Find and Count operations
func (r *ChildRepository) buildListFilter(filter ports.FilterOptions, parentID *uuid.UUID) bson.M {
	mongoFilter := bson.M{
		"deleted_at": nil,
	}

	if parentID != nil {
		mongoFilter["parentId"] = *parentID
	}

	if filter.FirstName != "" {
		mongoFilter["firstName"] = bson.M{"$regex": filter.FirstName, "$options": "i"}
	}

	if filter.LastName != "" {
		mongoFilter["lastName"] = bson.M{"$regex": filter.LastName, "$options": "i"}
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

// ListByParentID retrieves children for a specific parent with pagination, filtering, and sorting.
// It performs concurrent database operations for finding children and counting the total results
// to optimize performance. The method supports filtering by various criteria, sorting by different
// fields, and pagination with page size and page number.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - parentID: The unique identifier of the parent whose children to retrieve
//   - queryOptions: Options for filtering, sorting, and pagination
//
// Returns:
//   - []*domain.Child: A slice of child entities matching the criteria
//   - *ports.PagedResult: Pagination information including total count and whether there are more pages
//   - error: An error if there's a database error
func (r *ChildRepository) ListByParentID(ctx context.Context, parentID uuid.UUID, queryOptions ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.ListByParentID")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parentID.String()))

	filter := r.buildListFilter(queryOptions.Filter, &parentID)

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

	// Create channels for concurrent operations
	type findResult struct {
		children []*domain.Child
		err      error
	}
	type countResult struct {
		count int64
		err   error
	}
	findCh := make(chan findResult, 1)
	countCh := make(chan countResult, 1)

	// Execute Find operation concurrently
	go func() {
		cursor, err := r.collection.Find(ctx, filter, findOpts)
		if err != nil {
			findCh <- findResult{nil, fmt.Errorf("child.list.byParent.failed: %w", err)}
			return
		}
		defer cursor.Close(ctx)

		children := []*domain.Child{}
		if err := cursor.All(ctx, &children); err != nil {
			findCh <- findResult{nil, fmt.Errorf("child.decode.failed: %w", err)}
			return
		}
		findCh <- findResult{children, nil}
	}()

	// Execute Count operation concurrently
	go func() {
		count, err := r.countByParentID(ctx, parentID, queryOptions.Filter)
		if err != nil {
			countCh <- countResult{0, fmt.Errorf("child.totalCount.failed: %w", err)}
			return
		}
		countCh <- countResult{count, nil}
	}()

	// Wait for both operations to complete
	findRes := <-findCh
	countRes := <-countCh

	// Check for errors
	if findRes.err != nil {
		r.logger.Error("Failed to list children by parent ID", zap.Error(findRes.err), zap.String("parent_id", parentID.String()))
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
		HasNext:    (skip + int64(len(findRes.children))) < countRes.count,
	}

	return findRes.children, pagedResult, nil
}

// List retrieves a list of all children with pagination, filtering, and sorting.
// Unlike ListByParentID, this method retrieves children regardless of their parent.
// It performs concurrent database operations for finding children and counting the total results
// to optimize performance. The method supports filtering by various criteria, sorting by different
// fields, and pagination with page size and page number.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - queryOptions: Options for filtering, sorting, and pagination
//
// Returns:
//   - []*domain.Child: A slice of child entities matching the criteria
//   - *ports.PagedResult: Pagination information including total count and whether there are more pages
//   - error: An error if there's a database error
func (r *ChildRepository) List(ctx context.Context, queryOptions ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.List")
	defer span.End()

	filter := r.buildListFilter(queryOptions.Filter, nil)

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

	// Create channels for concurrent operations
	type findResult struct {
		children []*domain.Child
		err      error
	}
	type countResult struct {
		count int64
		err   error
	}
	findCh := make(chan findResult, 1)
	countCh := make(chan countResult, 1)

	// Execute Find operation concurrently
	go func() {
		cursor, err := r.collection.Find(ctx, filter, findOpts)
		if err != nil {
			findCh <- findResult{nil, fmt.Errorf("child.list.failed: %w", err)}
			return
		}
		defer cursor.Close(ctx)

		children := []*domain.Child{}
		if err := cursor.All(ctx, &children); err != nil {
			findCh <- findResult{nil, fmt.Errorf("child.decode.failed: %w", err)}
			return
		}
		findCh <- findResult{children, nil}
	}()

	// Execute Count operation concurrently
	go func() {
		count, err := r.Count(ctx, queryOptions.Filter)
		if err != nil {
			countCh <- countResult{0, fmt.Errorf("child.totalCount.failed: %w", err)}
			return
		}
		countCh <- countResult{count, nil}
	}()

	// Wait for both operations to complete
	findRes := <-findCh
	countRes := <-countCh

	// Check for errors
	if findRes.err != nil {
		r.logger.Error("Failed to list children", zap.Error(findRes.err))
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
		HasNext:    (skip + int64(len(findRes.children))) < countRes.count,
	}

	return findRes.children, pagedResult, nil
}

// countByParentID returns the total count of children for a specific parent matching the filter.
// This is a helper method used by ListByParentID to get the total count for pagination.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - parentID: The unique identifier of the parent whose children to count
//   - filter: The filter options containing criteria for filtering children
//
// Returns:
//   - int64: The total count of children matching the criteria
//   - error: An error if there's a database error
func (r *ChildRepository) countByParentID(ctx context.Context, parentID uuid.UUID, filter ports.FilterOptions) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.countByParentID")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parentID.String()))

	mongoFilter := r.buildListFilter(filter, &parentID)

	count, err := r.collection.CountDocuments(ctx, mongoFilter)
	if err != nil {
		r.logger.Error("Failed to count children by parent ID", zap.Error(err), zap.String("parent_id", parentID.String()))
		return 0, fmt.Errorf("child.count.byParent.failed: %w", err)
	}

	return count, nil
}

// Count returns the total count of children matching the filter.
// This method counts all children (not deleted) that match the specified filter criteria,
// regardless of their parent. It's used for pagination and statistics.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - filter: The filter options containing criteria for filtering children
//
// Returns:
//   - int64: The total count of children matching the criteria
//   - error: An error if there's a database error
func (r *ChildRepository) Count(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.Count")
	defer span.End()

	mongoFilter := r.buildListFilter(filter, nil)

	count, err := r.collection.CountDocuments(ctx, mongoFilter)
	if err != nil {
		r.logger.Error("Failed to count children", zap.Error(err))
		return 0, fmt.Errorf("child.count.failed: %w", err)
	}

	return count, nil
}
