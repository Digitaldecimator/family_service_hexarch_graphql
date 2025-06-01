package domain_test

import (
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewChild(t *testing.T) {
	// Arrange
	firstName := "Jane"
	lastName := "Doe"
	birthDate := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	parentID := uuid.New()

	// Act
	child := domain.NewChild(firstName, lastName, birthDate, parentID)

	// Assert
	assert.NotNil(t, child)
	assert.NotEqual(t, uuid.Nil, child.ID)
	assert.Equal(t, firstName, child.FirstName)
	assert.Equal(t, lastName, child.LastName)
	assert.Equal(t, birthDate, child.BirthDate)
	assert.Equal(t, parentID, child.ParentID)
	assert.False(t, child.CreatedAt.IsZero())
	assert.False(t, child.UpdatedAt.IsZero())
	assert.Nil(t, child.DeletedAt)
}

func TestChild_MarkAsDeleted(t *testing.T) {
	// Arrange
	child := domain.NewChild("Jane", "Doe", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), uuid.New())
	initialUpdatedAt := child.UpdatedAt

	// Wait a moment to ensure UpdatedAt will be different
	time.Sleep(1 * time.Millisecond)

	// Act
	child.MarkAsDeleted()

	// Assert
	assert.NotNil(t, child.DeletedAt)
	assert.True(t, child.UpdatedAt.After(initialUpdatedAt))
	assert.True(t, child.IsDeleted())
}

func TestChild_Update(t *testing.T) {
	// Arrange
	child := domain.NewChild("Jane", "Doe", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), uuid.New())
	initialUpdatedAt := child.UpdatedAt

	newFirstName := "John"
	newLastName := "Smith"
	newBirthDate := time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)

	// Wait a moment to ensure UpdatedAt will be different
	time.Sleep(1 * time.Millisecond)

	// Act
	child.Update(newFirstName, newLastName, newBirthDate)

	// Assert
	assert.Equal(t, newFirstName, child.FirstName)
	assert.Equal(t, newLastName, child.LastName)
	assert.Equal(t, newBirthDate, child.BirthDate)
	assert.True(t, child.UpdatedAt.After(initialUpdatedAt))
}

func TestChild_FullName(t *testing.T) {
	// Arrange
	firstName := "Jane"
	lastName := "Doe"
	child := domain.NewChild(firstName, lastName, time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), uuid.New())

	// Act
	fullName := child.FullName()

	// Assert
	assert.Equal(t, firstName+" "+lastName, fullName)
}

func TestChild_Age(t *testing.T) {
	// Arrange
	now := time.Now()

	// Test cases
	testCases := []struct {
		name        string
		birthDate   time.Time
		expectedAge int
	}{
		{
			name:        "Child born 5 years ago",
			birthDate:   now.AddDate(-5, 0, 0),
			expectedAge: 5,
		},
		{
			name:        "Child born 10 years ago",
			birthDate:   now.AddDate(-10, 0, 0),
			expectedAge: 10,
		},
		{
			name:        "Child born today",
			birthDate:   now,
			expectedAge: 0,
		},
		{
			name:        "Child born 5 years and 1 day ago",
			birthDate:   now.AddDate(-5, 0, -1),
			expectedAge: 5,
		},
		{
			name:        "Child's birthday is tomorrow (born 5 years ago minus 1 day)",
			birthDate:   now.AddDate(-5, 0, 1),
			expectedAge: 4, // Not yet 5 years old
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a child with the test birth date
			child := domain.NewChild("Test", "Child", tc.birthDate, uuid.New())

			// Act
			age := child.Age()

			// Assert
			assert.Equal(t, tc.expectedAge, age)
		})
	}
}

func TestChild_GetID(t *testing.T) {
	// Arrange
	child := domain.NewChild("Jane", "Doe", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), uuid.New())

	// Act
	id := child.GetID()

	// Assert
	assert.Equal(t, child.ID, id)
}

func TestChild_GetCreatedAt(t *testing.T) {
	// Arrange
	child := domain.NewChild("Jane", "Doe", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), uuid.New())

	// Act
	createdAt := child.GetCreatedAt()

	// Assert
	assert.Equal(t, child.CreatedAt, createdAt)
}

func TestChild_GetUpdatedAt(t *testing.T) {
	// Arrange
	child := domain.NewChild("Jane", "Doe", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), uuid.New())

	// Act
	updatedAt := child.GetUpdatedAt()

	// Assert
	assert.Equal(t, child.UpdatedAt, updatedAt)
}

func TestChild_GetDeletedAt(t *testing.T) {
	// Arrange
	child := domain.NewChild("Jane", "Doe", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), uuid.New())

	// Act
	deletedAt := child.GetDeletedAt()

	// Assert
	assert.Equal(t, child.DeletedAt, deletedAt)
	assert.Nil(t, deletedAt)

	// Mark as deleted and check again
	child.MarkAsDeleted()
	deletedAt = child.GetDeletedAt()
	assert.NotNil(t, deletedAt)
	assert.Equal(t, child.DeletedAt, deletedAt)
}
