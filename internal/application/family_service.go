// Package application implements the application layer of the family service.
// It contains the business logic and orchestrates the flow of data between the domain layer
// and the external interfaces. This layer is responsible for implementing use cases
// by coordinating domain entities and infrastructure services.
package application

import (
	"context"
	"strings"
	"time"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/abitofhelp/family-service2/internal/ports"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// FamilyService implements the ports.FamilyService interface.
// It provides methods for managing parents and children in the family service,
// including CRUD operations and relationship management between parents and children.
// The service uses repositories for data access, a transaction manager for ensuring
// data consistency, a validator for input validation, a logger for logging,
// a tracer for distributed tracing, and a localizer for error message localization.
type FamilyService struct {
	parentRepo         ports.ParentRepository   // Repository for parent entities
	childRepo          ports.ChildRepository    // Repository for child entities
	transactionManager ports.TransactionManager // Manages database transactions
	validator          *validator.Validate      // Validates input data
	logger             *zap.Logger              // Logs service operations
	tracer             trace.Tracer             // Provides distributed tracing
}

// NewFamilyService creates a new family service with the necessary dependencies.
// Parameters:
//   - repoFactory: Factory for creating repositories and transaction manager
//   - validator: Validator for input validation
//   - logger: Logger for logging service operations
//
// Returns:
//   - *FamilyService: A new instance of the family service
func NewFamilyService(
	repoFactory ports.RepositoryFactory,
	validator *validator.Validate,
	logger *zap.Logger,
) *FamilyService {
	return &FamilyService{
		parentRepo:         repoFactory.NewParentRepository(),
		childRepo:          repoFactory.NewChildRepository(),
		transactionManager: repoFactory.GetTransactionManager(),
		validator:          validator,
		logger:             logger,
		tracer:             otel.Tracer("application.family_service"),
	}
}

// CreateParent creates a new parent in the system.
// It validates the input data, creates a new Parent entity, and persists it to the database.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - firstName: The parent's first name
//   - lastName: The parent's last name
//   - email: The parent's email address
//   - birthDateStr: The parent's birth date as a string in RFC3339 format (e.g., "2006-01-02T15:04:05Z")
//
// Returns:
//   - *domain.Parent: The newly created parent entity if successful
//   - error: An error if validation fails or if there's a database error
func (s *FamilyService) CreateParent(ctx context.Context, firstName, lastName, email, birthDateStr string) (*domain.Parent, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.CreateParent")
	defer span.End()

	// Validate input
	if firstName == "" {
		return nil, domain.NewValidationError("Parent", "firstName", "is required")
	}
	if lastName == "" {
		return nil, domain.NewValidationError("Parent", "lastName", "is required")
	}
	if email == "" {
		return nil, domain.NewValidationError("Parent", "email", "is required")
	}
	if birthDateStr == "" {
		return nil, domain.NewValidationError("Parent", "birthDate", "is required")
	}

	// Parse birth date
	birthDate, err := time.Parse(time.RFC3339, birthDateStr)
	if err != nil {
		s.logger.Error("Failed to parse birth date", zap.Error(err), zap.String("birthDate", birthDateStr))
		return nil, domain.NewValidationError("Parent", "birthDate", "invalid format, expected RFC3339")
	}

	// Create parent
	parent := domain.NewParent(firstName, lastName, email, birthDate)

	// Validate parent
	if err := s.validator.Struct(parent); err != nil {
		s.logger.Error("Parent validation failed", zap.Error(err))

		// Check for specific validation errors
		if strings.Contains(err.Error(), "Email") {
			return nil, domain.NewValidationError("Parent", "email", "invalid format")
		}

		return nil, domain.NewValidationError("Parent", "", err.Error())
	}

	// Save parent
	err = s.parentRepo.Create(ctx, parent)
	if err != nil {
		s.logger.Error("Failed to create parent", zap.Error(err))
		return nil, domain.NewDatabaseError("create", "Parent", err)
	}

	return parent, nil
}

// GetParentByID retrieves a parent by ID from the repository.
// It attempts to find a parent with the specified ID that is not marked as deleted.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - id: The unique identifier of the parent to retrieve
//
// Returns:
//   - *domain.Parent: The retrieved parent entity if found
//   - error: A NotFoundError if the parent doesn't exist, or a database error
func (s *FamilyService) GetParentByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.GetParentByID")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", id.String()))

	parent, err := s.parentRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get parent", zap.Error(err), zap.String("parent_id", id.String()))
		// Check for context deadline exceeded error
		if err == context.DeadlineExceeded {
			return nil, err
		}
		// For other errors, return a NotFoundError
		return nil, domain.NewNotFoundError("Parent", id.String())
	}

	return parent, nil
}

// UpdateParent updates an existing parent with new information.
// It retrieves the parent, updates its attributes, validates the updated entity,
// and persists the changes to the database. The method handles validation of input data,
// including parsing the birth date string.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - id: The unique identifier of the parent to update
//   - firstName: The new first name for the parent
//   - lastName: The new last name for the parent
//   - email: The new email address for the parent
//   - birthDateStr: The new birth date as a string in RFC3339 format (e.g., "2006-01-02T15:04:05Z")
//
// Returns:
//   - *domain.Parent: The updated parent entity if successful
//   - error: A NotFoundError if the parent doesn't exist, a ValidationError if validation fails,
//            or a database error
func (s *FamilyService) UpdateParent(ctx context.Context, id uuid.UUID, firstName, lastName, email, birthDateStr string) (*domain.Parent, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.UpdateParent")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", id.String()))

	// Get existing parent
	parent, err := s.parentRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get parent for update", zap.Error(err), zap.String("parent_id", id.String()))
		return nil, domain.NewNotFoundError("Parent", id.String())
	}

	// Parse birth date
	birthDate, err := time.Parse(time.RFC3339, birthDateStr)
	if err != nil {
		s.logger.Error("Failed to parse birth date", zap.Error(err), zap.String("birthDate", birthDateStr))
		return nil, domain.NewValidationError("Parent", "birthDate", "invalid format, expected RFC3339")
	}

	// Update parent
	parent.Update(firstName, lastName, email, birthDate)

	// Validate parent
	if err := s.validator.Struct(parent); err != nil {
		s.logger.Error("Parent validation failed", zap.Error(err))

		// Check for specific validation errors
		if strings.Contains(err.Error(), "Email") {
			return nil, domain.NewValidationError("Parent", "email", "invalid format")
		}

		return nil, domain.NewValidationError("Parent", "", err.Error())
	}

	// Save parent
	err = s.parentRepo.Update(ctx, parent)
	if err != nil {
		s.logger.Error("Failed to update parent", zap.Error(err), zap.String("parent_id", id.String()))
		return nil, domain.NewDatabaseError("update", "Parent", err)
	}

	return parent, nil
}

// DeleteParent marks a parent as deleted in the database.
// This is a soft delete operation that sets the DeletedAt timestamp rather than
// removing the record from the database. The operation is performed within a transaction
// to ensure data consistency.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - id: The unique identifier of the parent to mark as deleted
//
// Returns:
//   - error: A NotFoundError if the parent doesn't exist, a TransactionError if the transaction
//            fails, or a database error
func (s *FamilyService) DeleteParent(ctx context.Context, id uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "FamilyService.DeleteParent")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", id.String()))

	// Begin transaction
	ctx, err := s.transactionManager.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return domain.NewTransactionError("begin", err)
	}

	// Delete parent
	err = s.parentRepo.Delete(ctx, id)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to delete parent", zap.Error(err), zap.String("parent_id", id.String()))

		// Check if this is a "not found" error
		if strings.Contains(err.Error(), "not found") {
			return domain.NewNotFoundError("Parent", id.String())
		}

		return domain.NewDatabaseError("delete", "Parent", err)
	}

	// Commit transaction
	err = s.transactionManager.CommitTx(ctx)
	if err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return domain.NewTransactionError("commit", err)
	}

	return nil
}

// ListParents retrieves a list of parents with pagination, filtering, and sorting.
// It delegates to the parent repository to fetch the data and handles any errors.
// The method supports filtering by various criteria, sorting by different fields,
// and pagination with page size and page number.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - options: Options for filtering, sorting, and pagination
//
// Returns:
//   - []*domain.Parent: A slice of parent entities matching the criteria
//   - *ports.PagedResult: Pagination information including total count and whether there are more pages
//   - error: A database error if the operation fails
func (s *FamilyService) ListParents(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.ListParents")
	defer span.End()

	parents, pagedResult, err := s.parentRepo.List(ctx, options)
	if err != nil {
		s.logger.Error("Failed to list parents", zap.Error(err))
		return nil, nil, domain.NewDatabaseError("list", "Parent", err)
	}

	return parents, pagedResult, nil
}

// CountParents returns the total count of parents matching the filter.
// It delegates to the parent repository to count the data and handles any errors.
// This method is useful for statistics and for clients that need to know the total
// count without retrieving the actual parent entities.
// The method uses OpenTelemetry for tracing and logs relevant information during the operation.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - filter: The filter options containing criteria for filtering parents
//
// Returns:
//   - int64: The total count of parents matching the criteria
//   - error: A database error if the operation fails
func (s *FamilyService) CountParents(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.CountParents")
	defer span.End()

	count, err := s.parentRepo.Count(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to count parents", zap.Error(err))
		return 0, domain.NewDatabaseError("count", "Parent", err)
	}

	return count, nil
}

// CreateChild creates a new child in the system and associates it with a parent.
// It validates the input data, creates a new Child entity, and persists it to the database
// within a transaction to ensure data consistency.
// Parameters:
//   - ctx: The context for the operation, used for tracing and cancellation
//   - firstName: The child's first name
//   - lastName: The child's last name
//   - birthDateStr: The child's birth date as a string in RFC3339 format (e.g., "2006-01-02T15:04:05Z")
//   - parentID: The UUID of the parent to associate with this child
//
// Returns:
//   - *domain.Child: The newly created child entity if successful
//   - error: An error if validation fails, if the parent doesn't exist, or if there's a database error
func (s *FamilyService) CreateChild(ctx context.Context, firstName, lastName, birthDateStr string, parentID uuid.UUID) (*domain.Child, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.CreateChild")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parentID.String()))

	// Validate input
	if firstName == "" {
		return nil, domain.NewValidationError("Child", "firstName", "is required")
	}
	if lastName == "" {
		return nil, domain.NewValidationError("Child", "lastName", "is required")
	}
	if birthDateStr == "" {
		return nil, domain.NewValidationError("Child", "birthDate", "is required")
	}

	// Parse birth date
	birthDate, err := time.Parse(time.RFC3339, birthDateStr)
	if err != nil {
		s.logger.Error("Failed to parse birth date", zap.Error(err), zap.String("birthDate", birthDateStr))
		return nil, domain.NewValidationError("Child", "birthDate", "invalid format, expected RFC3339")
	}

	// Create child
	child := domain.NewChild(firstName, lastName, birthDate, parentID)

	// Validate child
	if err := s.validator.Struct(child); err != nil {
		s.logger.Error("Child validation failed", zap.Error(err))
		return nil, domain.NewValidationError("Child", "", err.Error())
	}

	// Begin transaction
	ctx, err = s.transactionManager.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, domain.NewTransactionError("begin", err)
	}

	// Get parent to update its children array
	parent, err := s.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to get parent", zap.Error(err), zap.String("parent_id", parentID.String()))
		return nil, domain.NewNotFoundError("Parent", parentID.String())
	}

	// Save child
	err = s.childRepo.Create(ctx, child)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to create child", zap.Error(err))
		return nil, domain.NewDatabaseError("create", "Child", err)
	}

	// Add child to parent's children array
	s.logger.Debug("Before adding child to parent",
		zap.String("parent_id", parentID.String()),
		zap.Int("children_count", len(parent.Children)),
		zap.Any("parent", parent))

	parent.AddChild(*child)

	s.logger.Debug("After adding child to parent",
		zap.String("parent_id", parentID.String()),
		zap.Int("children_count", len(parent.Children)),
		zap.Any("parent", parent))

	err = s.parentRepo.Update(ctx, parent)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to update parent with child", zap.Error(err), zap.String("parent_id", parentID.String()), zap.String("child_id", child.ID.String()))
		return nil, domain.NewDatabaseError("update", "Parent", err)
	}

	// Verify the parent was updated correctly
	updatedParent, err := s.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		s.logger.Error("Failed to get updated parent", zap.Error(err), zap.String("parent_id", parentID.String()))
	} else {
		s.logger.Debug("After updating parent in database",
			zap.String("parent_id", parentID.String()),
			zap.Int("children_count", len(updatedParent.Children)),
			zap.Any("parent", updatedParent))
	}

	// Commit transaction
	err = s.transactionManager.CommitTx(ctx)
	if err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return nil, domain.NewTransactionError("commit", err)
	}

	return child, nil
}

// GetChildByID retrieves a child by ID
func (s *FamilyService) GetChildByID(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.GetChildByID")
	defer span.End()

	span.SetAttributes(attribute.String("child.id", id.String()))

	child, err := s.childRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get child", zap.Error(err), zap.String("child_id", id.String()))
		// Check for context deadline exceeded error
		if err == context.DeadlineExceeded {
			return nil, err
		}
		// For other errors, return a NotFoundError
		return nil, domain.NewNotFoundError("Child", id.String())
	}

	return child, nil
}

// UpdateChild updates an existing child
func (s *FamilyService) UpdateChild(ctx context.Context, id uuid.UUID, firstName, lastName, birthDateStr string) (*domain.Child, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.UpdateChild")
	defer span.End()

	span.SetAttributes(attribute.String("child.id", id.String()))

	// Get existing child
	child, err := s.childRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get child for update", zap.Error(err), zap.String("child_id", id.String()))
		return nil, domain.NewNotFoundError("Child", id.String())
	}

	// Parse birth date
	birthDate, err := time.Parse(time.RFC3339, birthDateStr)
	if err != nil {
		s.logger.Error("Failed to parse birth date", zap.Error(err), zap.String("birthDate", birthDateStr))
		return nil, domain.NewValidationError("Child", "birthDate", "invalid format, expected RFC3339")
	}

	// Update child
	child.Update(firstName, lastName, birthDate)

	// Validate child
	if err := s.validator.Struct(child); err != nil {
		s.logger.Error("Child validation failed", zap.Error(err))
		return nil, domain.NewValidationError("Child", "", err.Error())
	}

	// Save child
	err = s.childRepo.Update(ctx, child)
	if err != nil {
		s.logger.Error("Failed to update child", zap.Error(err), zap.String("child_id", id.String()))
		return nil, domain.NewDatabaseError("update", "Child", err)
	}

	return child, nil
}

// DeleteChild marks a child as deleted
func (s *FamilyService) DeleteChild(ctx context.Context, id uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "FamilyService.DeleteChild")
	defer span.End()

	span.SetAttributes(attribute.String("child.id", id.String()))

	// Begin transaction
	ctx, err := s.transactionManager.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return domain.NewTransactionError("begin", err)
	}

	// Get the child to find its parent
	child, err := s.childRepo.GetByID(ctx, id)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to get child for deletion", zap.Error(err), zap.String("child_id", id.String()))

		// Check if this is a "not found" error
		if strings.Contains(err.Error(), "not found") {
			return domain.NewNotFoundError("Child", id.String())
		}

		return domain.NewDatabaseError("get", "Child", err)
	}

	// Get the parent to update its children array
	parent, err := s.parentRepo.GetByID(ctx, child.ParentID)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to get parent for child deletion", zap.Error(err), zap.String("parent_id", child.ParentID.String()))

		// If parent not found, we can still proceed with deleting the child
		if !strings.Contains(err.Error(), "not found") {
			return domain.NewDatabaseError("get", "Parent", err)
		}
	}

	// Delete child
	err = s.childRepo.Delete(ctx, id)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to delete child", zap.Error(err), zap.String("child_id", id.String()))

		// Check if this is a "not found" error
		if strings.Contains(err.Error(), "not found") {
			return domain.NewNotFoundError("Child", id.String())
		}

		return domain.NewDatabaseError("delete", "Child", err)
	}

	// If parent was found, remove the child from its children array
	if parent != nil {
		s.logger.Debug("Before removing child from parent",
			zap.String("parent_id", parent.ID.String()),
			zap.String("child_id", id.String()),
			zap.Int("children_count", len(parent.Children)))

		parent.RemoveChild(id)

		s.logger.Debug("After removing child from parent",
			zap.String("parent_id", parent.ID.String()),
			zap.String("child_id", id.String()),
			zap.Int("children_count", len(parent.Children)))

		err = s.parentRepo.Update(ctx, parent)
		if err != nil {
			// Rollback transaction
			rollbackErr := s.transactionManager.RollbackTx(ctx)
			if rollbackErr != nil {
				s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
				// We don't return the rollback error as the original error is more important
			}

			s.logger.Error("Failed to update parent after child deletion", zap.Error(err), zap.String("parent_id", parent.ID.String()))
			return domain.NewDatabaseError("update", "Parent", err)
		}
	}

	// Commit transaction
	err = s.transactionManager.CommitTx(ctx)
	if err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return domain.NewTransactionError("commit", err)
	}

	return nil
}

// ListChildrenByParentID retrieves children for a specific parent with pagination, filtering, and sorting
func (s *FamilyService) ListChildrenByParentID(ctx context.Context, parentID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.ListChildrenByParentID")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parentID.String()))

	children, pagedResult, err := s.childRepo.ListByParentID(ctx, parentID, options)
	if err != nil {
		s.logger.Error("Failed to list children by parent", zap.Error(err), zap.String("parent_id", parentID.String()))
		return nil, nil, domain.NewDatabaseError("listByParentID", "Child", err)
	}

	return children, pagedResult, nil
}

// ListChildren retrieves a list of children with pagination, filtering, and sorting
func (s *FamilyService) ListChildren(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.ListChildren")
	defer span.End()

	children, pagedResult, err := s.childRepo.List(ctx, options)
	if err != nil {
		s.logger.Error("Failed to list children", zap.Error(err))
		return nil, nil, domain.NewDatabaseError("list", "Child", err)
	}

	return children, pagedResult, nil
}

// CountChildren returns the total count of children matching the filter
func (s *FamilyService) CountChildren(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "FamilyService.CountChildren")
	defer span.End()

	count, err := s.childRepo.Count(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to count children", zap.Error(err))
		return 0, domain.NewDatabaseError("count", "Child", err)
	}

	return count, nil
}

// AddChildToParent adds a child to a parent
func (s *FamilyService) AddChildToParent(ctx context.Context, parentID, childID uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "FamilyService.AddChildToParent")
	defer span.End()

	span.SetAttributes(
		attribute.String("parent.id", parentID.String()),
		attribute.String("child.id", childID.String()),
	)

	// Begin transaction
	ctx, err := s.transactionManager.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return domain.NewTransactionError("begin", err)
	}

	// Get parent
	parent, err := s.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to get parent", zap.Error(err), zap.String("parent_id", parentID.String()))
		return domain.NewNotFoundError("Parent", parentID.String())
	}

	// Get child
	child, err := s.childRepo.GetByID(ctx, childID)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to get child", zap.Error(err), zap.String("child_id", childID.String()))
		return domain.NewNotFoundError("Child", childID.String())
	}

	// Update child's parent ID
	child.ParentID = parentID
	err = s.childRepo.Update(ctx, child)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to update child", zap.Error(err), zap.String("child_id", childID.String()))
		return domain.NewDatabaseError("update", "Child", err)
	}

	// Add child to parent
	parent.AddChild(*child)
	err = s.parentRepo.Update(ctx, parent)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to update parent", zap.Error(err), zap.String("parent_id", parentID.String()))
		return domain.NewDatabaseError("update", "Parent", err)
	}

	// Commit transaction
	err = s.transactionManager.CommitTx(ctx)
	if err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return domain.NewTransactionError("commit", err)
	}

	return nil
}

// RemoveChildFromParent removes a child from a parent
func (s *FamilyService) RemoveChildFromParent(ctx context.Context, parentID, childID uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "FamilyService.RemoveChildFromParent")
	defer span.End()

	span.SetAttributes(
		attribute.String("parent.id", parentID.String()),
		attribute.String("child.id", childID.String()),
	)

	// Begin transaction
	ctx, err := s.transactionManager.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return domain.NewTransactionError("begin", err)
	}

	// Get parent
	parent, err := s.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to get parent", zap.Error(err), zap.String("parent_id", parentID.String()))
		return domain.NewNotFoundError("Parent", parentID.String())
	}

	// Remove child from parent
	if !parent.RemoveChild(childID) {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Child not found in parent", zap.String("parent_id", parentID.String()), zap.String("child_id", childID.String()))
		return domain.NewNotFoundError("Child", childID.String())
	}

	// Update parent
	err = s.parentRepo.Update(ctx, parent)
	if err != nil {
		// Rollback transaction
		rollbackErr := s.transactionManager.RollbackTx(ctx)
		if rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			// We don't return the rollback error as the original error is more important
		}

		s.logger.Error("Failed to update parent", zap.Error(err), zap.String("parent_id", parentID.String()))
		return domain.NewDatabaseError("update", "Parent", err)
	}

	// Commit transaction
	err = s.transactionManager.CommitTx(ctx)
	if err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return domain.NewTransactionError("commit", err)
	}

	return nil
}
