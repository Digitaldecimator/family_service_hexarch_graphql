// Package domain contains the core business entities and business rules for the family service.
// It defines the domain models and their behaviors independent of any external concerns.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Child represents a child entity in the family service.
// It contains personal information about the child and a reference to their parent.
// This entity implements the Entity interface for common CRUD operations.
type Child struct {
	ID        uuid.UUID  `json:"id" bson:"_id"`
	FirstName string     `json:"firstName" bson:"firstName" validate:"required"`
	LastName  string     `json:"lastName" bson:"lastName" validate:"required"`
	BirthDate time.Time  `json:"birthDate" bson:"birthDate" validate:"required"`
	ParentID  uuid.UUID  `json:"parentId" bson:"parentId" validate:"required"`
	CreatedAt time.Time  `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt" bson:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
}

// Ensure Child implements Entity interface
var _ Entity = (*Child)(nil)

// GetID returns the child's unique identifier.
// This method implements the Entity interface.
// Returns:
//   - uuid.UUID: The unique identifier of the child
func (c *Child) GetID() uuid.UUID {
	return c.ID
}

// GetCreatedAt returns the child's creation timestamp.
// This method implements the Entity interface.
// Returns:
//   - time.Time: The UTC timestamp when the child was created
func (c *Child) GetCreatedAt() time.Time {
	return c.CreatedAt
}

// GetUpdatedAt returns the child's last update timestamp.
// This method implements the Entity interface.
// Returns:
//   - time.Time: The UTC timestamp when the child was last updated
func (c *Child) GetUpdatedAt() time.Time {
	return c.UpdatedAt
}

// GetDeletedAt returns the child's deletion timestamp, if any.
// This method implements the Entity interface.
// Returns:
//   - *time.Time: The UTC timestamp when the child was marked as deleted, or nil if not deleted
func (c *Child) GetDeletedAt() *time.Time {
	return c.DeletedAt
}

// NewChild creates a new Child instance with a generated UUID and timestamps.
// It initializes a new child with the provided personal information, parent reference,
// and sets creation and update timestamps to the current UTC time.
// Parameters:
//   - firstName: The child's first name
//   - lastName: The child's last name
//   - birthDate: The child's date of birth
//   - parentID: The UUID of the parent this child belongs to
//
// Returns:
//   - *Child: A pointer to the newly created Child instance
func NewChild(firstName, lastName string, birthDate time.Time, parentID uuid.UUID) *Child {
	now := time.Now().UTC()
	return &Child{
		ID:        uuid.New(),
		FirstName: firstName,
		LastName:  lastName,
		BirthDate: birthDate,
		ParentID:  parentID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MarkAsDeleted marks the child as deleted by setting the DeletedAt timestamp.
// This is a soft delete operation that maintains the record but marks it as deleted.
func (c *Child) MarkAsDeleted() {
	now := time.Now().UTC()
	c.DeletedAt = &now
	c.UpdatedAt = now
}

// IsDeleted checks if the child is marked as deleted.
// Returns:
//   - bool: true if the child has been marked as deleted, false otherwise
func (c *Child) IsDeleted() bool {
	return c.DeletedAt != nil
}

// Update updates the child's personal information.
// It sets the new values for the child's attributes and updates the UpdatedAt timestamp.
// Parameters:
//   - firstName: The new first name
//   - lastName: The new last name
//   - birthDate: The new birth date
func (c *Child) Update(firstName, lastName string, birthDate time.Time) {
	c.FirstName = firstName
	c.LastName = lastName
	c.BirthDate = birthDate
	c.UpdatedAt = time.Now().UTC()
}

// FullName returns the full name of the child by concatenating the first and last names.
// Returns:
//   - string: The full name in the format "FirstName LastName"
func (c *Child) FullName() string {
	return c.FirstName + " " + c.LastName
}

// Age calculates the current age of the child based on their birth date.
// The age is calculated as the difference in years between the current date and the birth date,
// with an adjustment if the birthday hasn't occurred yet in the current year.
// Returns:
//   - int: The age in years
func (c *Child) Age() int {
	now := time.Now()
	years := now.Year() - c.BirthDate.Year()

	// Adjust age if birthday hasn't occurred yet this year
	if now.Month() < c.BirthDate.Month() ||
		(now.Month() == c.BirthDate.Month() && now.Day() < c.BirthDate.Day()) {
		years--
	}

	return years
}
