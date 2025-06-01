// Package ports defines the interfaces that connect the application's core business logic
// to external adapters. It follows the ports and adapters (hexagonal) architecture pattern,
// where ports are the interfaces that the application core exposes to interact with external systems.
// This package contains repository interfaces for data access, service interfaces for business logic,
// and configuration interfaces for application settings.
package ports

import (
	"context"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/google/uuid"
)

// FilterOptions represents options for filtering list queries
type FilterOptions struct {
	FirstName string
	LastName  string
	Email     string
	MinAge    int
	MaxAge    int
}

// PaginationOptions represents options for paginating list queries
type PaginationOptions struct {
	Page     int
	PageSize int
}

// SortOptions represents options for sorting list queries
type SortOptions struct {
	Field     string
	Direction string // "asc" or "desc"
}

// QueryOptions combines all query options
type QueryOptions struct {
	Filter     FilterOptions
	Pagination PaginationOptions
	Sort       SortOptions
}

// PagedResult represents a paginated result
type PagedResult struct {
	TotalCount int64
	Page       int
	PageSize   int
	HasNext    bool
}

// ParentRepository defines the interface for parent data access
type ParentRepository interface {
	// Create creates a new parent
	Create(ctx context.Context, parent *domain.Parent) error

	// GetByID retrieves a parent by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error)

	// Update updates an existing parent
	Update(ctx context.Context, parent *domain.Parent) error

	// Delete marks a parent as deleted
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves a list of parents with pagination, filtering, and sorting
	List(ctx context.Context, options QueryOptions) ([]*domain.Parent, *PagedResult, error)

	// Count returns the total count of parents matching the filter
	Count(ctx context.Context, filter FilterOptions) (int64, error)
}

// ChildRepository defines the interface for child data access
type ChildRepository interface {
	// Create creates a new child
	Create(ctx context.Context, child *domain.Child) error

	// GetByID retrieves a child by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error)

	// Update updates an existing child
	Update(ctx context.Context, child *domain.Child) error

	// Delete marks a child as deleted
	Delete(ctx context.Context, id uuid.UUID) error

	// ListByParentID retrieves children for a specific parent with pagination, filtering, and sorting
	ListByParentID(ctx context.Context, parentID uuid.UUID, options QueryOptions) ([]*domain.Child, *PagedResult, error)

	// List retrieves a list of children with pagination, filtering, and sorting
	List(ctx context.Context, options QueryOptions) ([]*domain.Child, *PagedResult, error)

	// Count returns the total count of children matching the filter
	Count(ctx context.Context, filter FilterOptions) (int64, error)
}

// TransactionManager defines the interface for managing database transactions
type TransactionManager interface {
	// BeginTx begins a new transaction
	BeginTx(ctx context.Context) (context.Context, error)

	// CommitTx commits the current transaction
	CommitTx(ctx context.Context) error

	// RollbackTx rolls back the current transaction
	RollbackTx(ctx context.Context) error
}

// RepositoryFactory defines the interface for creating repositories
type RepositoryFactory interface {
	// NewParentRepository creates a new parent repository
	NewParentRepository() ParentRepository

	// NewChildRepository creates a new child repository
	NewChildRepository() ChildRepository

	// GetTransactionManager returns the transaction manager
	GetTransactionManager() TransactionManager
}
