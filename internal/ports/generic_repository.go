// Package ports defines the interfaces that connect the application's core business logic
// to external adapters. It follows the ports and adapters (hexagonal architecture) pattern,
// where ports are the interfaces that the application core exposes to interact with external systems.
// This package contains repository interfaces for data access, service interfaces for business logic,
// and configuration interfaces for application settings.
package ports

import (
	"context"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/google/uuid"
)

// Repository is a generic repository interface for CRUD operations on entities
type Repository[T domain.Entity] interface {
	// Create creates a new entity
	Create(ctx context.Context, entity T) error

	// GetByID retrieves an entity by ID
	GetByID(ctx context.Context, id uuid.UUID) (T, error)

	// Update updates an existing entity
	Update(ctx context.Context, entity T) error

	// Delete marks an entity as deleted
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves a list of entities with pagination, filtering, and sorting
	List(ctx context.Context, options QueryOptions) ([]T, *PagedResult, error)

	// Count returns the total count of entities matching the filter
	Count(ctx context.Context, filter FilterOptions) (int64, error)
}
