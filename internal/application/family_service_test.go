package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/application"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/mocks"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupFamilyServiceTest(t *testing.T) (
	*application.FamilyService,
	*mocks.MockRepositoryFactory,
	*validator.Validate,
	*mocks.MockLocalizer,
	context.Context,
) {
	// Create mocks
	repoFactory := mocks.NewMockRepositoryFactory()
	validate := validator.New()
	localizer := mocks.NewMockLocalizer()
	logger := zaptest.NewLogger(t)

	// Create service
	service := application.NewFamilyService(
		repoFactory,
		validate,
		logger,
	)

	// Create context
	ctx := context.Background()

	return service, repoFactory, validate, localizer, ctx
}

func TestCreateParent_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	firstName := "John"
	lastName := "Doe"
	email := "john.doe@example.com"
	birthDate := time.Now().AddDate(-30, 0, 0).Format(time.RFC3339)

	// Act
	parent, err := service.CreateParent(ctx, firstName, lastName, email, birthDate)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, parent)
	assert.Equal(t, firstName, parent.FirstName)
	assert.Equal(t, lastName, parent.LastName)
	assert.Equal(t, email, parent.Email)

	// Verify parent was saved to repository
	savedParent, err := repoFactory.GetMockParentRepository().GetByID(ctx, parent.ID)
	require.NoError(t, err)
	assert.Equal(t, parent.ID, savedParent.ID)
}

func TestCreateParent_ValidationFailure(t *testing.T) {
	// Arrange
	_, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a custom validator that always fails
	customValidator := validator.New()

	// Create a new service with the custom validator
	customService := application.NewFamilyService(
		repoFactory,
		customValidator,
		zaptest.NewLogger(t),
	)

	// No need to set up custom error messages anymore

	// Use an invalid email to trigger validation failure
	firstName := "John"
	lastName := "Doe"
	email := "invalid-email" // Invalid email format
	birthDate := time.Now().AddDate(-30, 0, 0).Format(time.RFC3339)

	// Act
	parent, err := customService.CreateParent(ctx, firstName, lastName, email, birthDate)

	// Assert
	require.Error(t, err)
	assert.Nil(t, parent)
}

func TestCreateParent_MissingRequiredFields(t *testing.T) {
	// Arrange
	service, _, _, _, ctx := setupFamilyServiceTest(t)

	testCases := []struct {
		name      string
		firstName string
		lastName  string
		email     string
		birthDate string
		errorMsg  string
	}{
		{
			name:      "Missing first name",
			firstName: "",
			lastName:  "Doe",
			email:     "john.doe@example.com",
			birthDate: time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
			errorMsg:  "validation failed for Parent: field first name is required",
		},
		{
			name:      "Missing last name",
			firstName: "John",
			lastName:  "",
			email:     "john.doe@example.com",
			birthDate: time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
			errorMsg:  "validation failed for Parent: field last name is required",
		},
		{
			name:      "Missing email",
			firstName: "John",
			lastName:  "Doe",
			email:     "",
			birthDate: time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
			errorMsg:  "validation failed for Parent: field email is required",
		},
		{
			name:      "Missing birth date",
			firstName: "John",
			lastName:  "Doe",
			email:     "john.doe@example.com",
			birthDate: "",
			errorMsg:  "validation failed for Parent: field birth date is required",
		},
		{
			name:      "Invalid birth date",
			firstName: "John",
			lastName:  "Doe",
			email:     "john.doe@example.com",
			birthDate: "invalid-date",
			errorMsg:  "validation failed for Parent: field birth date invalid format, expected RFC3339",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			parent, err := service.CreateParent(ctx, tc.firstName, tc.lastName, tc.email, tc.birthDate)

			// Assert
			require.Error(t, err)
			assert.Nil(t, parent)
			assert.Contains(t, err.Error(), tc.errorMsg)
		})
	}
}

func TestGetParentByID_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(testParent)

	// Act
	parent, err := service.GetParentByID(ctx, testParent.ID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, parent)
	assert.Equal(t, testParent.ID, parent.ID)
	assert.Equal(t, testParent.FirstName, parent.FirstName)
	assert.Equal(t, testParent.LastName, parent.LastName)
	assert.Equal(t, testParent.Email, parent.Email)
}

func TestGetParentByID_NotFound(t *testing.T) {
	// Arrange
	service, _, _, _, ctx := setupFamilyServiceTest(t)

	// Act
	parent, err := service.GetParentByID(ctx, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Nil(t, parent)
	assert.Contains(t, err.Error(), "Parent with ID")
}

func TestUpdateParent_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(testParent)

	// New values
	newFirstName := "Jane"
	newLastName := "Smith"
	newEmail := "jane.smith@example.com"
	newBirthDate := time.Now().AddDate(-25, 0, 0).Format(time.RFC3339)

	// Act
	updatedParent, err := service.UpdateParent(ctx, testParent.ID, newFirstName, newLastName, newEmail, newBirthDate)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, updatedParent)
	assert.Equal(t, testParent.ID, updatedParent.ID)
	assert.Equal(t, newFirstName, updatedParent.FirstName)
	assert.Equal(t, newLastName, updatedParent.LastName)
	assert.Equal(t, newEmail, updatedParent.Email)

	// Verify parent was updated in repository
	savedParent, err := repoFactory.GetMockParentRepository().GetByID(ctx, testParent.ID)
	require.NoError(t, err)
	assert.Equal(t, newFirstName, savedParent.FirstName)
	assert.Equal(t, newLastName, savedParent.LastName)
	assert.Equal(t, newEmail, savedParent.Email)
}

func TestUpdateParent_NotFound(t *testing.T) {
	// Arrange
	service, _, _, _, ctx := setupFamilyServiceTest(t)

	// Act
	updatedParent, err := service.UpdateParent(
		ctx,
		uuid.New(),
		"Jane",
		"Smith",
		"jane.smith@example.com",
		time.Now().AddDate(-25, 0, 0).Format(time.RFC3339),
	)

	// Assert
	require.Error(t, err)
	assert.Nil(t, updatedParent)
	assert.Contains(t, err.Error(), "Parent with ID")
}

func TestDeleteParent_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(testParent)

	// Act
	err := service.DeleteParent(ctx, testParent.ID)

	// Assert
	require.NoError(t, err)

	// Verify parent was deleted in repository
	_, err = repoFactory.GetMockParentRepository().GetByID(ctx, testParent.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parent not found")
}

func TestDeleteParent_NotFound(t *testing.T) {
	// Arrange
	service, _, _, _, ctx := setupFamilyServiceTest(t)

	// Act
	err := service.DeleteParent(ctx, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Parent with ID")
}

func TestListParents_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create test parents
	parent1 := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	parent2 := domain.NewParent("Jane", "Smith", "jane.smith@example.com", time.Now().AddDate(-25, 0, 0))
	parent3 := domain.NewParent("Bob", "Johnson", "bob.johnson@example.com", time.Now().AddDate(-40, 0, 0))

	repoFactory.GetMockParentRepository().AddTestParent(parent1)
	repoFactory.GetMockParentRepository().AddTestParent(parent2)
	repoFactory.GetMockParentRepository().AddTestParent(parent3)

	// Create query options
	options := ports.QueryOptions{
		Pagination: ports.PaginationOptions{
			Page:     0,
			PageSize: 10,
		},
	}

	// Act
	parents, pagedResult, err := service.ListParents(ctx, options)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, parents)
	assert.NotNil(t, pagedResult)
	assert.Equal(t, int64(3), pagedResult.TotalCount)
	assert.Len(t, parents, 3)
}

func TestListParents_WithFiltering(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create test parents
	parent1 := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	parent2 := domain.NewParent("Jane", "Smith", "jane.smith@example.com", time.Now().AddDate(-25, 0, 0))
	parent3 := domain.NewParent("John", "Johnson", "john.johnson@example.com", time.Now().AddDate(-40, 0, 0))

	repoFactory.GetMockParentRepository().AddTestParent(parent1)
	repoFactory.GetMockParentRepository().AddTestParent(parent2)
	repoFactory.GetMockParentRepository().AddTestParent(parent3)

	// Create query options with filter
	options := ports.QueryOptions{
		Filter: ports.FilterOptions{
			FirstName: "John",
		},
		Pagination: ports.PaginationOptions{
			Page:     0,
			PageSize: 10,
		},
	}

	// Act
	parents, pagedResult, err := service.ListParents(ctx, options)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, parents)
	assert.NotNil(t, pagedResult)
	assert.Equal(t, int64(2), pagedResult.TotalCount)
	assert.Len(t, parents, 2)

	// Verify all returned parents have first name "John"
	for _, parent := range parents {
		assert.Equal(t, "John", parent.FirstName)
	}
}

// Child-related tests

func TestCreateChild_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent first
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	firstName := "Jane"
	lastName := "Doe"
	birthDate := time.Now().AddDate(-5, 0, 0).Format(time.RFC3339)

	// Act
	child, err := service.CreateChild(ctx, firstName, lastName, birthDate, parent.ID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, child)
	assert.Equal(t, firstName, child.FirstName)
	assert.Equal(t, lastName, child.LastName)
	assert.Equal(t, parent.ID, child.ParentID)

	// Verify child was saved to repository
	savedChild, err := repoFactory.GetMockChildRepository().GetByID(ctx, child.ID)
	require.NoError(t, err)
	assert.Equal(t, child.ID, savedChild.ID)
}

func TestCreateChild_ValidationFailure(t *testing.T) {
	// Arrange
	_, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent first
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Create a custom validator
	customValidator := validator.New()

	// Create a new service with the custom validator
	customService := application.NewFamilyService(
		repoFactory,
		customValidator,
		zaptest.NewLogger(t),
	)

	// Use an invalid birth date to trigger validation failure
	firstName := "Jane"
	lastName := "Doe"
	birthDate := "invalid-date"

	// Act
	child, err := customService.CreateChild(ctx, firstName, lastName, birthDate, parent.ID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, child)
	assert.Contains(t, err.Error(), "validation failed for Child: field birth date invalid format, expected RFC3339")
}

func TestCreateChild_MissingRequiredFields(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent first
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	testCases := []struct {
		name      string
		firstName string
		lastName  string
		birthDate string
		parentID  uuid.UUID
		errorMsg  string
	}{
		{
			name:      "Missing first name",
			firstName: "",
			lastName:  "Doe",
			birthDate: time.Now().AddDate(-5, 0, 0).Format(time.RFC3339),
			parentID:  parent.ID,
			errorMsg:  "validation failed for Child: field first name is required",
		},
		{
			name:      "Missing last name",
			firstName: "Jane",
			lastName:  "",
			birthDate: time.Now().AddDate(-5, 0, 0).Format(time.RFC3339),
			parentID:  parent.ID,
			errorMsg:  "validation failed for Child: field last name is required",
		},
		{
			name:      "Missing birth date",
			firstName: "Jane",
			lastName:  "Doe",
			birthDate: "",
			parentID:  parent.ID,
			errorMsg:  "validation failed for Child: field birth date is required",
		},
		{
			name:      "Invalid birth date",
			firstName: "Jane",
			lastName:  "Doe",
			birthDate: "invalid-date",
			parentID:  parent.ID,
			errorMsg:  "validation failed for Child: field birth date invalid format, expected RFC3339",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			child, err := service.CreateChild(ctx, tc.firstName, tc.lastName, tc.birthDate, tc.parentID)

			// Assert
			require.Error(t, err)
			assert.Nil(t, child)
			assert.Contains(t, err.Error(), tc.errorMsg)
		})
	}
}

func TestGetChildByID_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parent.ID)
	repoFactory.GetMockChildRepository().AddTestChild(testChild)

	// Act
	child, err := service.GetChildByID(ctx, testChild.ID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, child)
	assert.Equal(t, testChild.ID, child.ID)
	assert.Equal(t, testChild.FirstName, child.FirstName)
	assert.Equal(t, testChild.LastName, child.LastName)
	assert.Equal(t, testChild.ParentID, child.ParentID)
}

func TestGetChildByID_NotFound(t *testing.T) {
	// Arrange
	service, _, _, _, ctx := setupFamilyServiceTest(t)

	// Act
	child, err := service.GetChildByID(ctx, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Nil(t, child)
	assert.Contains(t, err.Error(), "Child with ID")
}

func TestUpdateChild_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parent.ID)
	repoFactory.GetMockChildRepository().AddTestChild(testChild)

	// New values
	newFirstName := "John"
	newLastName := "Smith"
	newBirthDate := time.Now().AddDate(-6, 0, 0).Format(time.RFC3339)

	// Act
	updatedChild, err := service.UpdateChild(ctx, testChild.ID, newFirstName, newLastName, newBirthDate)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, updatedChild)
	assert.Equal(t, testChild.ID, updatedChild.ID)
	assert.Equal(t, newFirstName, updatedChild.FirstName)
	assert.Equal(t, newLastName, updatedChild.LastName)

	// Verify child was updated in repository
	savedChild, err := repoFactory.GetMockChildRepository().GetByID(ctx, testChild.ID)
	require.NoError(t, err)
	assert.Equal(t, newFirstName, savedChild.FirstName)
	assert.Equal(t, newLastName, savedChild.LastName)
}

func TestUpdateChild_NotFound(t *testing.T) {
	// Arrange
	service, _, _, _, ctx := setupFamilyServiceTest(t)

	// Act
	updatedChild, err := service.UpdateChild(
		ctx,
		uuid.New(),
		"John",
		"Smith",
		time.Now().AddDate(-6, 0, 0).Format(time.RFC3339),
	)

	// Assert
	require.Error(t, err)
	assert.Nil(t, updatedChild)
	assert.Contains(t, err.Error(), "Child with ID")
}

func TestDeleteChild_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parent.ID)
	repoFactory.GetMockChildRepository().AddTestChild(testChild)

	// Act
	err := service.DeleteChild(ctx, testChild.ID)

	// Assert
	require.NoError(t, err)

	// Verify child was deleted in repository
	_, err = repoFactory.GetMockChildRepository().GetByID(ctx, testChild.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "child not found")
}

func TestDeleteChild_NotFound(t *testing.T) {
	// Arrange
	service, _, _, _, ctx := setupFamilyServiceTest(t)

	// Act
	err := service.DeleteChild(ctx, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Child with ID")
}

func TestListChildrenByParentID_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parent.ID)
	child2 := domain.NewChild("Jack", "Doe", time.Now().AddDate(-3, 0, 0), parent.ID)
	child3 := domain.NewChild("Jill", "Doe", time.Now().AddDate(-1, 0, 0), parent.ID)

	repoFactory.GetMockChildRepository().AddTestChild(child1)
	repoFactory.GetMockChildRepository().AddTestChild(child2)
	repoFactory.GetMockChildRepository().AddTestChild(child3)

	// Create query options
	options := ports.QueryOptions{
		Pagination: ports.PaginationOptions{
			Page:     0,
			PageSize: 10,
		},
	}

	// Act
	children, pagedResult, err := service.ListChildrenByParentID(ctx, parent.ID, options)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, children)
	assert.NotNil(t, pagedResult)
	assert.Equal(t, int64(3), pagedResult.TotalCount)
	assert.Len(t, children, 3)

	// Verify all children have the correct parent ID
	for _, child := range children {
		assert.Equal(t, parent.ID, child.ParentID)
	}
}

func TestCountParents_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create test parents
	parent1 := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	parent2 := domain.NewParent("Jane", "Smith", "jane.smith@example.com", time.Now().AddDate(-25, 0, 0))
	parent3 := domain.NewParent("Bob", "Johnson", "bob.johnson@example.com", time.Now().AddDate(-40, 0, 0))

	repoFactory.GetMockParentRepository().AddTestParent(parent1)
	repoFactory.GetMockParentRepository().AddTestParent(parent2)
	repoFactory.GetMockParentRepository().AddTestParent(parent3)

	// Create filter options
	filter := ports.FilterOptions{
		FirstName: "John", // Exact match for John
	}

	// Act
	count, err := service.CountParents(ctx, filter)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestCountParents_Error(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Set up the mock to return an error
	mockRepo := repoFactory.GetMockParentRepository()
	mockRepo.CountFunc = func(ctx context.Context, filter ports.FilterOptions) (int64, error) {
		return 0, errors.New("mock count error")
	}

	// Act
	count, err := service.CountParents(ctx, ports.FilterOptions{})

	// Assert
	require.Error(t, err)
	assert.Equal(t, int64(0), count)
}

func TestListChildren_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parent.ID)
	child2 := domain.NewChild("Jack", "Doe", time.Now().AddDate(-3, 0, 0), parent.ID)
	child3 := domain.NewChild("Jill", "Doe", time.Now().AddDate(-1, 0, 0), parent.ID)

	repoFactory.GetMockChildRepository().AddTestChild(child1)
	repoFactory.GetMockChildRepository().AddTestChild(child2)
	repoFactory.GetMockChildRepository().AddTestChild(child3)

	// Create query options
	options := ports.QueryOptions{
		Pagination: ports.PaginationOptions{
			Page:     0,
			PageSize: 10,
		},
	}

	// Act
	children, pagedResult, err := service.ListChildren(ctx, options)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, children)
	assert.NotNil(t, pagedResult)
	assert.Equal(t, int64(3), pagedResult.TotalCount)
	assert.Len(t, children, 3)
}

func TestListChildren_Error(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Set up the mock to return an error
	mockRepo := repoFactory.GetMockChildRepository()
	mockRepo.ListFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		return nil, nil, errors.New("mock list error")
	}

	// Act
	children, pagedResult, err := service.ListChildren(ctx, ports.QueryOptions{})

	// Assert
	require.Error(t, err)
	assert.Nil(t, children)
	assert.Nil(t, pagedResult)
}

func TestCountChildren_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parent.ID)
	child2 := domain.NewChild("Jack", "Doe", time.Now().AddDate(-3, 0, 0), parent.ID)
	child3 := domain.NewChild("Jill", "Doe", time.Now().AddDate(-1, 0, 0), parent.ID)

	repoFactory.GetMockChildRepository().AddTestChild(child1)
	repoFactory.GetMockChildRepository().AddTestChild(child2)
	repoFactory.GetMockChildRepository().AddTestChild(child3)

	// Create filter options
	filter := ports.FilterOptions{
		FirstName: "Jane", // Exact match for Jane
	}

	// Act
	count, err := service.CountChildren(ctx, filter)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestCountChildren_Error(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Set up the mock to return an error
	mockRepo := repoFactory.GetMockChildRepository()
	mockRepo.CountFunc = func(ctx context.Context, filter ports.FilterOptions) (int64, error) {
		return 0, errors.New("mock count error")
	}

	// Act
	count, err := service.CountChildren(ctx, ports.FilterOptions{})

	// Assert
	require.Error(t, err)
	assert.Equal(t, int64(0), count)
}

func TestAddChildToParent_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Create a child
	child := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), uuid.New())
	repoFactory.GetMockChildRepository().AddTestChild(child)

	// Act
	err := service.AddChildToParent(ctx, parent.ID, child.ID)

	// Assert
	require.NoError(t, err)

	// Verify child was added to parent
	updatedParent, err := repoFactory.GetMockParentRepository().GetByID(ctx, parent.ID)
	require.NoError(t, err)
	assert.Len(t, updatedParent.Children, 1)
	assert.Equal(t, child.ID, updatedParent.Children[0].ID)

	// Verify child's parent ID was updated
	updatedChild, err := repoFactory.GetMockChildRepository().GetByID(ctx, child.ID)
	require.NoError(t, err)
	assert.Equal(t, parent.ID, updatedChild.ParentID)
}

func TestAddChildToParent_ParentNotFound(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a child
	child := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), uuid.New())
	repoFactory.GetMockChildRepository().AddTestChild(child)

	// Act
	err := service.AddChildToParent(ctx, uuid.New(), child.ID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Parent with ID")
}

func TestAddChildToParent_ChildNotFound(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Act
	err := service.AddChildToParent(ctx, parent.ID, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Child with ID")
}

func TestRemoveChildFromParent_Success(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))

	// Create a child
	child := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parent.ID)

	// Add child to parent
	parent.AddChild(*child)

	// Add to repositories
	repoFactory.GetMockParentRepository().AddTestParent(parent)
	repoFactory.GetMockChildRepository().AddTestChild(child)

	// Act
	err := service.RemoveChildFromParent(ctx, parent.ID, child.ID)

	// Assert
	require.NoError(t, err)

	// Verify child was removed from parent
	updatedParent, err := repoFactory.GetMockParentRepository().GetByID(ctx, parent.ID)
	require.NoError(t, err)
	assert.Len(t, updatedParent.Children, 0)
}

func TestRemoveChildFromParent_ParentNotFound(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a child
	child := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), uuid.New())
	repoFactory.GetMockChildRepository().AddTestChild(child)

	// Act
	err := service.RemoveChildFromParent(ctx, uuid.New(), child.ID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Parent with ID")
}

func TestRemoveChildFromParent_ChildNotFoundInParent(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupFamilyServiceTest(t)

	// Create a parent
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Act
	err := service.RemoveChildFromParent(ctx, parent.ID, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Child with ID")
}
