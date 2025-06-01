package domain_test

import (
	"testing"
	"time"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewParent(t *testing.T) {
	// Arrange
	firstName := "John"
	lastName := "Doe"
	email := "john.doe@example.com"
	birthDate := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

	// Act
	parent := domain.NewParent(firstName, lastName, email, birthDate)

	// Assert
	assert.NotNil(t, parent)
	assert.NotEqual(t, uuid.Nil, parent.ID)
	assert.Equal(t, firstName, parent.FirstName)
	assert.Equal(t, lastName, parent.LastName)
	assert.Equal(t, email, parent.Email)
	assert.Equal(t, birthDate, parent.BirthDate)
	assert.Empty(t, parent.Children)
	assert.False(t, parent.CreatedAt.IsZero())
	assert.False(t, parent.UpdatedAt.IsZero())
	assert.Nil(t, parent.DeletedAt)
}

func TestParent_AddChild(t *testing.T) {
	// Arrange
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
	child := domain.NewChild("Jane", "Doe", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), parent.ID)
	initialUpdatedAt := parent.UpdatedAt

	// Wait a moment to ensure UpdatedAt will be different
	time.Sleep(1 * time.Millisecond)

	// Act
	parent.AddChild(*child)

	// Assert
	assert.Len(t, parent.Children, 1)
	assert.Equal(t, child.ID, parent.Children[0].ID)
	assert.True(t, parent.UpdatedAt.After(initialUpdatedAt))
}

func TestParent_RemoveChild(t *testing.T) {
	// Arrange
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
	child1 := domain.NewChild("Jane", "Doe", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), parent.ID)
	child2 := domain.NewChild("Jack", "Doe", time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC), parent.ID)

	parent.AddChild(*child1)
	parent.AddChild(*child2)

	initialUpdatedAt := parent.UpdatedAt

	// Wait a moment to ensure UpdatedAt will be different
	time.Sleep(1 * time.Millisecond)

	// Act - Remove existing child
	result1 := parent.RemoveChild(child1.ID)

	// Assert
	assert.True(t, result1)
	assert.Len(t, parent.Children, 1)
	assert.Equal(t, child2.ID, parent.Children[0].ID)
	assert.True(t, parent.UpdatedAt.After(initialUpdatedAt))

	// Act - Remove non-existing child
	result2 := parent.RemoveChild(uuid.New())

	// Assert
	assert.False(t, result2)
	assert.Len(t, parent.Children, 1)
}

func TestParent_MarkAsDeleted(t *testing.T) {
	// Arrange
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
	initialUpdatedAt := parent.UpdatedAt

	// Wait a moment to ensure UpdatedAt will be different
	time.Sleep(1 * time.Millisecond)

	// Act
	parent.MarkAsDeleted()

	// Assert
	assert.NotNil(t, parent.DeletedAt)
	assert.True(t, parent.UpdatedAt.After(initialUpdatedAt))
	assert.True(t, parent.IsDeleted())
}

func TestParent_Update(t *testing.T) {
	// Arrange
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
	initialUpdatedAt := parent.UpdatedAt

	newFirstName := "Jane"
	newLastName := "Smith"
	newEmail := "jane.smith@example.com"
	newBirthDate := time.Date(1985, 1, 1, 0, 0, 0, 0, time.UTC)

	// Wait a moment to ensure UpdatedAt will be different
	time.Sleep(1 * time.Millisecond)

	// Act
	parent.Update(newFirstName, newLastName, newEmail, newBirthDate)

	// Assert
	assert.Equal(t, newFirstName, parent.FirstName)
	assert.Equal(t, newLastName, parent.LastName)
	assert.Equal(t, newEmail, parent.Email)
	assert.Equal(t, newBirthDate, parent.BirthDate)
	assert.True(t, parent.UpdatedAt.After(initialUpdatedAt))
}

func TestParent_FullName(t *testing.T) {
	// Arrange
	firstName := "John"
	lastName := "Doe"
	parent := domain.NewParent(firstName, lastName, "john.doe@example.com", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))

	// Act
	fullName := parent.FullName()

	// Assert
	assert.Equal(t, firstName+" "+lastName, fullName)
}

func TestParent_GetID(t *testing.T) {
	// Arrange
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))

	// Act
	id := parent.GetID()

	// Assert
	assert.Equal(t, parent.ID, id)
}

func TestParent_GetCreatedAt(t *testing.T) {
	// Arrange
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))

	// Act
	createdAt := parent.GetCreatedAt()

	// Assert
	assert.Equal(t, parent.CreatedAt, createdAt)
}

func TestParent_GetUpdatedAt(t *testing.T) {
	// Arrange
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))

	// Act
	updatedAt := parent.GetUpdatedAt()

	// Assert
	assert.Equal(t, parent.UpdatedAt, updatedAt)
}

func TestParent_GetDeletedAt(t *testing.T) {
	// Arrange
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))

	// Act
	deletedAt := parent.GetDeletedAt()

	// Assert
	assert.Equal(t, parent.DeletedAt, deletedAt)
	assert.Nil(t, deletedAt)

	// Mark as deleted and check again
	parent.MarkAsDeleted()
	deletedAt = parent.GetDeletedAt()
	assert.NotNil(t, deletedAt)
	assert.Equal(t, parent.DeletedAt, deletedAt)
}
