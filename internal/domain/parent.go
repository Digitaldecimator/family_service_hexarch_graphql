// Package domain contains the core business entities and business rules for the family service.
// It defines the domain models and their behaviors independent of any external concerns.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Parent represents a parent entity in the family service.
// It contains personal information about the parent and references to their children.
// This entity implements the Entity interface for common CRUD operations.
type Parent struct {
	ID        uuid.UUID  `json:"id" bson:"_id"`
	FirstName string     `json:"firstName" bson:"firstName" validate:"required"`
	LastName  string     `json:"lastName" bson:"lastName" validate:"required"`
	Email     string     `json:"email" bson:"email" validate:"required,email"`
	BirthDate time.Time  `json:"birthDate" bson:"birthDate" validate:"required"`
	Children  []Child    `json:"children,omitempty" bson:"children,omitempty"`
	CreatedAt time.Time  `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt" bson:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
}

// Ensure Parent implements Entity interface
var _ Entity = (*Parent)(nil)

// GetID returns the parent's unique identifier.
// This method implements the Entity interface.
// Returns:
//   - uuid.UUID: The unique identifier of the parent
func (p *Parent) GetID() uuid.UUID {
	return p.ID
}

// GetCreatedAt returns the parent's creation timestamp.
// This method implements the Entity interface.
// Returns:
//   - time.Time: The UTC timestamp when the parent was created
func (p *Parent) GetCreatedAt() time.Time {
	return p.CreatedAt
}

// GetUpdatedAt returns the parent's last update timestamp.
// This method implements the Entity interface.
// Returns:
//   - time.Time: The UTC timestamp when the parent was last updated
func (p *Parent) GetUpdatedAt() time.Time {
	return p.UpdatedAt
}

// GetDeletedAt returns the parent's deletion timestamp, if any.
// This method implements the Entity interface.
// Returns:
//   - *time.Time: The UTC timestamp when the parent was marked as deleted, or nil if not deleted
func (p *Parent) GetDeletedAt() *time.Time {
	return p.DeletedAt
}

// NewParent creates a new Parent instance with a generated UUID and timestamps.
// It initializes a new parent with the provided personal information and sets
// creation and update timestamps to the current UTC time.
// Parameters:
//   - firstName: The parent's first name
//   - lastName: The parent's last name
//   - email: The parent's email address
//   - birthDate: The parent's date of birth
//
// Returns:
//   - *Parent: A pointer to the newly created Parent instance
func NewParent(firstName, lastName, email string, birthDate time.Time) *Parent {
	now := time.Now().UTC()
	return &Parent{
		ID:        uuid.New(),
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		BirthDate: birthDate,
		Children:  []Child{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddChild adds a child to the parent's list of children.
// It appends the child to the parent's Children slice and updates the UpdatedAt timestamp.
// Parameters:
//   - child: The Child entity to add to this parent
func (p *Parent) AddChild(child Child) {
	p.Children = append(p.Children, child)
	p.UpdatedAt = time.Now().UTC()
}

// RemoveChild removes a child from the parent's list of children.
// It searches for a child with the given ID and removes it if found.
// Parameters:
//   - childID: The UUID of the child to remove
//
// Returns:
//   - bool: true if the child was found and removed, false otherwise
func (p *Parent) RemoveChild(childID uuid.UUID) bool {
	for i, child := range p.Children {
		if child.ID == childID {
			// Remove the child at index i by appending the slices before and after it
			p.Children = append(p.Children[:i], p.Children[i+1:]...)
			p.UpdatedAt = time.Now().UTC()
			return true
		}
	}
	return false
}

// MarkAsDeleted marks the parent as deleted by setting the DeletedAt timestamp.
// This is a soft delete operation that maintains the record but marks it as deleted.
func (p *Parent) MarkAsDeleted() {
	now := time.Now().UTC()
	p.DeletedAt = &now
	p.UpdatedAt = now
}

// IsDeleted checks if the parent is marked as deleted.
// Returns:
//   - bool: true if the parent has been marked as deleted, false otherwise
func (p *Parent) IsDeleted() bool {
	return p.DeletedAt != nil
}

// Update updates the parent's personal information.
// It sets the new values for the parent's attributes and updates the UpdatedAt timestamp.
// Parameters:
//   - firstName: The new first name
//   - lastName: The new last name
//   - email: The new email address
//   - birthDate: The new birth date
func (p *Parent) Update(firstName, lastName, email string, birthDate time.Time) {
	p.FirstName = firstName
	p.LastName = lastName
	p.Email = email
	p.BirthDate = birthDate
	p.UpdatedAt = time.Now().UTC()
}

// FullName returns the full name of the parent by concatenating the first and last names.
// Returns:
//   - string: The full name in the format "FirstName LastName"
func (p *Parent) FullName() string {
	return p.FirstName + " " + p.LastName
}
