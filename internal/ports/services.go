// Package ports defines the interfaces that connect the application's core business logic
// to external adapters. It follows the ports and adapters (hexagonal architecture) pattern
// where these interfaces act as "ports" that external components can implement.
// This package contains service interfaces for business operations and repository interfaces
// for data access, allowing the core application to remain independent of external concerns.
package ports

import (
	"context"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"github.com/google/uuid"
)

// ParentService defines the interface for parent business operations.
// It provides methods for creating, retrieving, updating, and deleting parent entities,
// as well as listing and counting parents with various filtering, pagination, and sorting options.
// This interface is implemented by the application layer and used by adapters like GraphQL resolvers.
type ParentService interface {
	// CreateParent creates a new parent with the provided information.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - firstName: The parent's first name
	//   - lastName: The parent's last name
	//   - email: The parent's email address
	//   - birthDate: The parent's birth date as a string in RFC3339 format
	//
	// Returns:
	//   - *domain.Parent: The newly created parent entity if successful
	//   - error: An error if validation fails or if there's a database error
	CreateParent(ctx context.Context, firstName, lastName, email string, birthDate string) (*domain.Parent, error)

	// GetParentByID retrieves a parent by their unique identifier.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - id: The unique identifier of the parent to retrieve
	//
	// Returns:
	//   - *domain.Parent: The retrieved parent entity if found
	//   - error: An error if the parent doesn't exist or if there's a database error
	GetParentByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error)

	// UpdateParent updates an existing parent with the provided information.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - id: The unique identifier of the parent to update
	//   - firstName: The new first name
	//   - lastName: The new last name
	//   - email: The new email address
	//   - birthDate: The new birth date as a string in RFC3339 format
	//
	// Returns:
	//   - *domain.Parent: The updated parent entity if successful
	//   - error: An error if the parent doesn't exist, validation fails, or if there's a database error
	UpdateParent(ctx context.Context, id uuid.UUID, firstName, lastName, email string, birthDate string) (*domain.Parent, error)

	// DeleteParent marks a parent as deleted.
	// This is typically a soft delete operation that maintains the record but marks it as deleted.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - id: The unique identifier of the parent to delete
	//
	// Returns:
	//   - error: An error if the parent doesn't exist or if there's a database error
	DeleteParent(ctx context.Context, id uuid.UUID) error

	// ListParents retrieves a list of parents with pagination, filtering, and sorting.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - options: Query options including filtering, pagination, and sorting parameters
	//
	// Returns:
	//   - []*domain.Parent: A slice of parent entities matching the query criteria
	//   - *PagedResult: Pagination metadata including total count and page information
	//   - error: An error if there's a database error or if the query options are invalid
	ListParents(ctx context.Context, options QueryOptions) ([]*domain.Parent, *PagedResult, error)

	// CountParents returns the total count of parents matching the filter.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - filter: Filter options to apply when counting parents
	//
	// Returns:
	//   - int64: The number of parents matching the filter criteria
	//   - error: An error if there's a database error or if the filter options are invalid
	CountParents(ctx context.Context, filter FilterOptions) (int64, error)
}

// ChildService defines the interface for child business operations.
// It provides methods for creating, retrieving, updating, and deleting child entities,
// as well as listing and counting children with various filtering, pagination, and sorting options.
// This interface is implemented by the application layer and used by adapters like GraphQL resolvers.
type ChildService interface {
	// CreateChild creates a new child with the provided information and associates it with a parent.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - firstName: The child's first name
	//   - lastName: The child's last name
	//   - birthDate: The child's birth date as a string in RFC3339 format
	//   - parentID: The unique identifier of the parent to associate with this child
	//
	// Returns:
	//   - *domain.Child: The newly created child entity if successful
	//   - error: An error if validation fails, if the parent doesn't exist, or if there's a database error
	CreateChild(ctx context.Context, firstName, lastName string, birthDate string, parentID uuid.UUID) (*domain.Child, error)

	// GetChildByID retrieves a child by their unique identifier.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - id: The unique identifier of the child to retrieve
	//
	// Returns:
	//   - *domain.Child: The retrieved child entity if found
	//   - error: An error if the child doesn't exist or if there's a database error
	GetChildByID(ctx context.Context, id uuid.UUID) (*domain.Child, error)

	// UpdateChild updates an existing child with the provided information.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - id: The unique identifier of the child to update
	//   - firstName: The new first name
	//   - lastName: The new last name
	//   - birthDate: The new birth date as a string in RFC3339 format
	//
	// Returns:
	//   - *domain.Child: The updated child entity if successful
	//   - error: An error if the child doesn't exist, validation fails, or if there's a database error
	UpdateChild(ctx context.Context, id uuid.UUID, firstName, lastName string, birthDate string) (*domain.Child, error)

	// DeleteChild marks a child as deleted.
	// This is typically a soft delete operation that maintains the record but marks it as deleted.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - id: The unique identifier of the child to delete
	//
	// Returns:
	//   - error: An error if the child doesn't exist or if there's a database error
	DeleteChild(ctx context.Context, id uuid.UUID) error

	// ListChildrenByParentID retrieves children for a specific parent with pagination, filtering, and sorting.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - parentID: The unique identifier of the parent whose children to retrieve
	//   - options: Query options including filtering, pagination, and sorting parameters
	//
	// Returns:
	//   - []*domain.Child: A slice of child entities matching the query criteria
	//   - *PagedResult: Pagination metadata including total count and page information
	//   - error: An error if there's a database error or if the query options are invalid
	ListChildrenByParentID(ctx context.Context, parentID uuid.UUID, options QueryOptions) ([]*domain.Child, *PagedResult, error)

	// ListChildren retrieves a list of all children with pagination, filtering, and sorting.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - options: Query options including filtering, pagination, and sorting parameters
	//
	// Returns:
	//   - []*domain.Child: A slice of child entities matching the query criteria
	//   - *PagedResult: Pagination metadata including total count and page information
	//   - error: An error if there's a database error or if the query options are invalid
	ListChildren(ctx context.Context, options QueryOptions) ([]*domain.Child, *PagedResult, error)

	// CountChildren returns the total count of children matching the filter.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - filter: Filter options to apply when counting children
	//
	// Returns:
	//   - int64: The number of children matching the filter criteria
	//   - error: An error if there's a database error or if the filter options are invalid
	CountChildren(ctx context.Context, filter FilterOptions) (int64, error)
}

// FamilyService combines parent and child services into a unified interface.
// It extends both ParentService and ChildService interfaces and adds methods
// for managing relationships between parents and children. This interface
// provides a comprehensive API for family management operations.
type FamilyService interface {
	ParentService
	ChildService

	// AddChildToParent adds a child to a parent by establishing a parent-child relationship.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - parentID: The unique identifier of the parent
	//   - childID: The unique identifier of the child to add to the parent
	//
	// Returns:
	//   - error: An error if either the parent or child doesn't exist, or if there's a database error
	AddChildToParent(ctx context.Context, parentID, childID uuid.UUID) error

	// RemoveChildFromParent removes a child from a parent by breaking the parent-child relationship.
	// Parameters:
	//   - ctx: The context for the operation, used for tracing and cancellation
	//   - parentID: The unique identifier of the parent
	//   - childID: The unique identifier of the child to remove from the parent
	//
	// Returns:
	//   - error: An error if either the parent or child doesn't exist, if the child is not
	//     associated with the parent, or if there's a database error
	RemoveChildFromParent(ctx context.Context, parentID, childID uuid.UUID) error
}

// AuthorizationService defines the interface for authorization operations.
// It provides methods for checking user permissions, roles, and retrieving
// user information from the context. This interface is used to implement
// authorization checks throughout the application.
type AuthorizationService interface {
	// IsAuthorized checks if the user is authorized to perform the specified operation.
	// Parameters:
	//   - ctx: The context containing user authentication information
	//   - operation: The name of the operation to check authorization for
	//
	// Returns:
	//   - bool: true if the user is authorized to perform the operation, false otherwise
	//   - error: An error if the authorization check fails due to technical reasons
	IsAuthorized(ctx context.Context, operation string) (bool, error)

	// IsAdmin checks if the user has admin role.
	// Parameters:
	//   - ctx: The context containing user authentication information
	//
	// Returns:
	//   - bool: true if the user has admin role, false otherwise
	//   - error: An error if the role check fails due to technical reasons
	IsAdmin(ctx context.Context) (bool, error)

	// GetUserID retrieves the user ID from the context.
	// Parameters:
	//   - ctx: The context containing user authentication information
	//
	// Returns:
	//   - string: The unique identifier of the authenticated user
	//   - error: An error if the user ID cannot be retrieved or if the user is not authenticated
	GetUserID(ctx context.Context) (string, error)

	// GetUserRoles retrieves the user roles from the context.
	// Parameters:
	//   - ctx: The context containing user authentication information
	//
	// Returns:
	//   - []string: A slice of role names assigned to the authenticated user
	//   - error: An error if the user roles cannot be retrieved or if the user is not authenticated
	GetUserRoles(ctx context.Context) ([]string, error)
}
