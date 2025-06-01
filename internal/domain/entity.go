// Package domain contains the core business entities and business rules for the family service.
// It defines the domain models and their behaviors independent of any external concerns.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entity defines the common interface for all domain entities.
// This interface provides a standard set of methods that all entities must implement,
// ensuring consistent behavior for CRUD operations, soft deletion, and timestamp tracking.
type Entity interface {
	// GetID returns the entity's unique identifier.
	// Returns:
	//   - uuid.UUID: The unique identifier of the entity
	GetID() uuid.UUID

	// IsDeleted checks if the entity is marked as deleted.
	// Returns:
	//   - bool: true if the entity has been marked as deleted, false otherwise
	IsDeleted() bool

	// MarkAsDeleted marks the entity as deleted by setting the DeletedAt timestamp.
	// This is a soft delete operation that maintains the record but marks it as deleted.
	MarkAsDeleted()

	// GetCreatedAt returns the entity's creation timestamp.
	// Returns:
	//   - time.Time: The UTC timestamp when the entity was created
	GetCreatedAt() time.Time

	// GetUpdatedAt returns the entity's last update timestamp.
	// Returns:
	//   - time.Time: The UTC timestamp when the entity was last updated
	GetUpdatedAt() time.Time

	// GetDeletedAt returns the entity's deletion timestamp, if any.
	// Returns:
	//   - *time.Time: The UTC timestamp when the entity was marked as deleted, or nil if not deleted
	GetDeletedAt() *time.Time
}
