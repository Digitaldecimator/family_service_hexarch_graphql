package graphql_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/adapters/graphql"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/mocks"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupResolverTest(t *testing.T) (*graphql.Resolver, *mocks.MockFamilyService, *mocks.MockAuthorizationService) {
	// Create mocks
	mockFamilyService := mocks.NewMockFamilyService()
	mockAuthService := mocks.NewMockAuthorizationService()
	logger := zaptest.NewLogger(t)

	// Create resolver
	resolver := graphql.NewResolver(mockFamilyService, mockAuthService, logger)

	return resolver, mockFamilyService, mockAuthService
}

func TestNewResolver(t *testing.T) {
	// Create mocks
	mockFamilyService := mocks.NewMockFamilyService()
	mockAuthService := mocks.NewMockAuthorizationService()
	logger := zaptest.NewLogger(t)

	// Create resolver
	resolver := graphql.NewResolver(mockFamilyService, mockAuthService, logger)

	// Assert
	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.Query())
	assert.NotNil(t, resolver.Mutation())
	assert.NotNil(t, resolver.Parent())
	assert.NotNil(t, resolver.Child())
	assert.NotNil(t, resolver.ParentConnection())
	assert.NotNil(t, resolver.ChildConnection())
}

func TestQueryResolver_Parent(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	testParent.ID = parentID

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "parent:read", permission)
		return true, nil
	}

	mockFamilyService.GetParentByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
		assert.Equal(t, parentID, id)
		return testParent, nil
	}

	// Execute
	result, err := resolver.Query().Parent(ctx, parentIDStr)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testParent, result)
}

func TestQueryResolver_Parent_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Query().Parent(ctx, parentIDStr)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestQueryResolver_Parent_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Query().Parent(ctx, parentIDStr)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestQueryResolver_Parent_InvalidID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Query().Parent(ctx, "invalid-uuid")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid parent ID")
}

func TestQueryResolver_Parent_GetError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.GetParentByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
		return nil, errors.New("get error")
	}

	// Execute
	result, err := resolver.Query().Parent(ctx, parentIDStr)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get parent")
}

func TestQueryResolver_Parent_NilContext(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Execute
	result, err := resolver.Query().Parent(nil, parentIDStr)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "nil context")
}

func TestQueryResolver_Child(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()
	parentID := uuid.New()

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	testChild.ID = childID

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "child:read", permission)
		return true, nil
	}

	mockFamilyService.GetChildByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
		assert.Equal(t, childID, id)
		return testChild, nil
	}

	// Execute
	result, err := resolver.Query().Child(ctx, childIDStr)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testChild, result)
}

func TestQueryResolver_Child_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Query().Child(ctx, childIDStr)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestQueryResolver_Child_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Query().Child(ctx, childIDStr)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestQueryResolver_Child_InvalidID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Query().Child(ctx, "invalid-uuid")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid child ID")
}

func TestQueryResolver_Child_GetError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.GetChildByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
		return nil, errors.New("get error")
	}

	// Execute
	result, err := resolver.Query().Child(ctx, childIDStr)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get child")
}

func TestQueryResolver_Child_NilContext(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	childID := uuid.New()
	childIDStr := childID.String()

	// Execute
	result, err := resolver.Query().Child(nil, childIDStr)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "nil context")
}

func TestParentResolver_ID(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	testParent.ID = parentID

	// Execute
	result, err := resolver.Parent().ID(ctx, testParent)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, parentID.String(), result)
}

func TestChildResolver_ID(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	parentID := uuid.New()

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	testChild.ID = childID

	// Execute
	result, err := resolver.Child().ID(ctx, testChild)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, childID.String(), result)
}

func TestParentResolver_BirthDate(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	birthDate := time.Now().AddDate(-30, 0, 0)

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", birthDate)

	// Execute
	result, err := resolver.Parent().BirthDate(ctx, testParent)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, birthDate.Format(time.RFC3339), result)
}

func TestChildResolver_BirthDate(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	birthDate := time.Now().AddDate(-5, 0, 0)
	parentID := uuid.New()

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", birthDate, parentID)

	// Execute
	result, err := resolver.Child().BirthDate(ctx, testChild)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, birthDate.Format(time.RFC3339), result)
}

func TestChildResolver_ParentID(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)

	// Execute
	result, err := resolver.Child().ParentID(ctx, testChild)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, parentID.String(), result)
}

func TestParentResolver_CreatedAt(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	createdAt := time.Now()

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	testParent.CreatedAt = createdAt

	// Execute
	result, err := resolver.Parent().CreatedAt(ctx, testParent)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, createdAt.Format(time.RFC3339), result)
}

func TestParentResolver_UpdatedAt(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	updatedAt := time.Now()

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	testParent.UpdatedAt = updatedAt

	// Execute
	result, err := resolver.Parent().UpdatedAt(ctx, testParent)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, updatedAt.Format(time.RFC3339), result)
}

func TestChildResolver_CreatedAt(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	createdAt := time.Now()
	parentID := uuid.New()

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	testChild.CreatedAt = createdAt

	// Execute
	result, err := resolver.Child().CreatedAt(ctx, testChild)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, createdAt.Format(time.RFC3339), result)
}

func TestChildResolver_UpdatedAt(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	updatedAt := time.Now()
	parentID := uuid.New()

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	testChild.UpdatedAt = updatedAt

	// Execute
	result, err := resolver.Child().UpdatedAt(ctx, testChild)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, updatedAt.Format(time.RFC3339), result)
}

func TestParentResolver_Children(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()

	// Create a test parent with children
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), testParent.ID)
	child2 := domain.NewChild("Jack", "Doe", time.Now().AddDate(-3, 0, 0), testParent.ID)
	testParent.Children = []domain.Child{*child1, *child2}

	// Execute
	result, err := resolver.Parent().Children(ctx, testParent)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, testParent.Children, result)
}

func TestMutationResolver_CreateParent(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Input data
	input := graphql.CreateParentInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		BirthDate: time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
	}

	// Create a test parent
	testParent := domain.NewParent(input.FirstName, input.LastName, input.Email, time.Now().AddDate(-30, 0, 0))

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "parent:create", permission)
		return true, nil
	}

	mockFamilyService.CreateParentFunc = func(ctx context.Context, firstName, lastName, email, birthDate string) (*domain.Parent, error) {
		assert.Equal(t, input.FirstName, firstName)
		assert.Equal(t, input.LastName, lastName)
		assert.Equal(t, input.Email, email)
		assert.Equal(t, input.BirthDate, birthDate)
		return testParent, nil
	}

	// Execute
	result, err := resolver.Mutation().CreateParent(ctx, input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testParent, result)
}

func TestMutationResolver_CreateParent_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Input data
	input := graphql.CreateParentInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		BirthDate: time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Mutation().CreateParent(ctx, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestMutationResolver_CreateParent_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Input data
	input := graphql.CreateParentInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		BirthDate: time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Mutation().CreateParent(ctx, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestMutationResolver_CreateParent_ServiceError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Input data
	input := graphql.CreateParentInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		BirthDate: time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.CreateParentFunc = func(ctx context.Context, firstName, lastName, email, birthDate string) (*domain.Parent, error) {
		return nil, errors.New("service error")
	}

	// Execute
	result, err := resolver.Mutation().CreateParent(ctx, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create parent")
}

func TestMutationResolver_CreateParent_NilContext(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)

	// Input data
	input := graphql.CreateParentInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		BirthDate: time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
	}

	// Execute
	result, err := resolver.Mutation().CreateParent(nil, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "nil context")
}

func TestMutationResolver_CreateChild(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()

	// Input data
	input := graphql.CreateChildInput{
		FirstName: "Jane",
		LastName:  "Doe",
		BirthDate: time.Now().AddDate(-5, 0, 0).Format(time.RFC3339),
		ParentID:  parentID.String(),
	}

	// Create a test child
	testChild := domain.NewChild(input.FirstName, input.LastName, time.Now().AddDate(-5, 0, 0), parentID)

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "child:create", permission)
		return true, nil
	}

	mockFamilyService.CreateChildFunc = func(ctx context.Context, firstName, lastName, birthDate string, parentID uuid.UUID) (*domain.Child, error) {
		assert.Equal(t, input.FirstName, firstName)
		assert.Equal(t, input.LastName, lastName)
		assert.Equal(t, input.BirthDate, birthDate)
		assert.Equal(t, parentID, parentID)
		return testChild, nil
	}

	// Execute
	result, err := resolver.Mutation().CreateChild(ctx, input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testChild, result)
}

func TestMutationResolver_UpdateParent(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	testParent.ID = parentID

	// Create updated parent
	updatedFirstName := "Jane"
	updatedLastName := "Smith"
	updatedEmail := "jane.smith@example.com"
	updatedBirthDate := time.Now().AddDate(-25, 0, 0).Format(time.RFC3339)

	updatedParent := domain.NewParent(updatedFirstName, updatedLastName, updatedEmail, time.Now().AddDate(-25, 0, 0))
	updatedParent.ID = parentID

	// Input data
	input := graphql.UpdateParentInput{
		FirstName: &updatedFirstName,
		LastName:  &updatedLastName,
		Email:     &updatedEmail,
		BirthDate: &updatedBirthDate,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "parent:update", permission)
		return true, nil
	}

	mockFamilyService.GetParentByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
		assert.Equal(t, parentID, id)
		return testParent, nil
	}

	mockFamilyService.UpdateParentFunc = func(ctx context.Context, id uuid.UUID, firstName, lastName, email, birthDate string) (*domain.Parent, error) {
		assert.Equal(t, parentID, id)
		assert.Equal(t, updatedFirstName, firstName)
		assert.Equal(t, updatedLastName, lastName)
		assert.Equal(t, updatedEmail, email)
		assert.Equal(t, updatedBirthDate, birthDate)
		return updatedParent, nil
	}

	// Execute
	result, err := resolver.Mutation().UpdateParent(ctx, parentIDStr, input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, updatedParent, result)
}

func TestMutationResolver_UpdateParent_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Input data
	updatedFirstName := "Jane"
	input := graphql.UpdateParentInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Mutation().UpdateParent(ctx, parentIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestMutationResolver_UpdateParent_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Input data
	updatedFirstName := "Jane"
	input := graphql.UpdateParentInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Mutation().UpdateParent(ctx, parentIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestMutationResolver_UpdateParent_InvalidID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Input data
	updatedFirstName := "Jane"
	input := graphql.UpdateParentInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Mutation().UpdateParent(ctx, "invalid-uuid", input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid parent ID")
}

func TestMutationResolver_UpdateParent_GetError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Input data
	updatedFirstName := "Jane"
	input := graphql.UpdateParentInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.GetParentByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
		return nil, errors.New("get error")
	}

	// Execute
	result, err := resolver.Mutation().UpdateParent(ctx, parentIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get parent")
}

func TestMutationResolver_UpdateParent_UpdateError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Create a test parent
	testParent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	testParent.ID = parentID

	// Input data
	updatedFirstName := "Jane"
	input := graphql.UpdateParentInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.GetParentByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
		return testParent, nil
	}

	mockFamilyService.UpdateParentFunc = func(ctx context.Context, id uuid.UUID, firstName, lastName, email, birthDate string) (*domain.Parent, error) {
		return nil, errors.New("update error")
	}

	// Execute
	result, err := resolver.Mutation().UpdateParent(ctx, parentIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to update parent")
}

func TestMutationResolver_UpdateParent_NilContext(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Input data
	updatedFirstName := "Jane"
	input := graphql.UpdateParentInput{
		FirstName: &updatedFirstName,
	}

	// Execute
	result, err := resolver.Mutation().UpdateParent(nil, parentIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "nil context")
}

func TestMutationResolver_DeleteParent(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "parent:delete", permission)
		return true, nil
	}

	mockFamilyService.DeleteParentFunc = func(ctx context.Context, id uuid.UUID) error {
		assert.Equal(t, parentID, id)
		return nil
	}

	// Execute
	result, err := resolver.Mutation().DeleteParent(ctx, parentIDStr)

	// Assert
	require.NoError(t, err)
	assert.True(t, result)
}

func TestMutationResolver_DeleteParent_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Mutation().DeleteParent(ctx, parentIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestMutationResolver_DeleteParent_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Mutation().DeleteParent(ctx, parentIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestMutationResolver_DeleteParent_InvalidID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Mutation().DeleteParent(ctx, "invalid-uuid")

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "invalid parent ID")
}

func TestMutationResolver_DeleteParent_DeleteError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.DeleteParentFunc = func(ctx context.Context, id uuid.UUID) error {
		return errors.New("delete error")
	}

	// Execute
	result, err := resolver.Mutation().DeleteParent(ctx, parentIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to delete parent")
}

func TestMutationResolver_DeleteParent_NilContext(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Execute
	result, err := resolver.Mutation().DeleteParent(nil, parentIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "nil context")
}

func TestMutationResolver_UpdateChild(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()
	parentID := uuid.New()

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	testChild.ID = childID

	// Create updated child
	updatedFirstName := "John"
	updatedLastName := "Smith"
	updatedBirthDate := time.Now().AddDate(-4, 0, 0).Format(time.RFC3339)

	updatedChild := domain.NewChild(updatedFirstName, updatedLastName, time.Now().AddDate(-4, 0, 0), parentID)
	updatedChild.ID = childID

	// Input data
	input := graphql.UpdateChildInput{
		FirstName: &updatedFirstName,
		LastName:  &updatedLastName,
		BirthDate: &updatedBirthDate,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "child:update", permission)
		return true, nil
	}

	mockFamilyService.GetChildByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
		assert.Equal(t, childID, id)
		return testChild, nil
	}

	mockFamilyService.UpdateChildFunc = func(ctx context.Context, id uuid.UUID, firstName, lastName, birthDate string) (*domain.Child, error) {
		assert.Equal(t, childID, id)
		assert.Equal(t, updatedFirstName, firstName)
		assert.Equal(t, updatedLastName, lastName)
		assert.Equal(t, updatedBirthDate, birthDate)
		return updatedChild, nil
	}

	// Execute
	result, err := resolver.Mutation().UpdateChild(ctx, childIDStr, input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, updatedChild, result)
}

func TestMutationResolver_UpdateChild_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Input data
	updatedFirstName := "John"
	input := graphql.UpdateChildInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Mutation().UpdateChild(ctx, childIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestMutationResolver_UpdateChild_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Input data
	updatedFirstName := "John"
	input := graphql.UpdateChildInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Mutation().UpdateChild(ctx, childIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestMutationResolver_UpdateChild_InvalidID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Input data
	updatedFirstName := "John"
	input := graphql.UpdateChildInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Mutation().UpdateChild(ctx, "invalid-uuid", input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid child ID")
}

func TestMutationResolver_UpdateChild_GetError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Input data
	updatedFirstName := "John"
	input := graphql.UpdateChildInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.GetChildByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
		return nil, errors.New("get error")
	}

	// Execute
	result, err := resolver.Mutation().UpdateChild(ctx, childIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get child")
}

func TestMutationResolver_UpdateChild_UpdateError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()
	parentID := uuid.New()

	// Create a test child
	testChild := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	testChild.ID = childID

	// Input data
	updatedFirstName := "John"
	input := graphql.UpdateChildInput{
		FirstName: &updatedFirstName,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.GetChildByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
		return testChild, nil
	}

	mockFamilyService.UpdateChildFunc = func(ctx context.Context, id uuid.UUID, firstName, lastName, birthDate string) (*domain.Child, error) {
		return nil, errors.New("update error")
	}

	// Execute
	result, err := resolver.Mutation().UpdateChild(ctx, childIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to update child")
}

func TestMutationResolver_UpdateChild_NilContext(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	childID := uuid.New()
	childIDStr := childID.String()

	// Input data
	updatedFirstName := "John"
	input := graphql.UpdateChildInput{
		FirstName: &updatedFirstName,
	}

	// Execute
	result, err := resolver.Mutation().UpdateChild(nil, childIDStr, input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "nil context")
}

func TestMutationResolver_DeleteChild(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "child:delete", permission)
		return true, nil
	}

	mockFamilyService.DeleteChildFunc = func(ctx context.Context, id uuid.UUID) error {
		assert.Equal(t, childID, id)
		return nil
	}

	// Execute
	result, err := resolver.Mutation().DeleteChild(ctx, childIDStr)

	// Assert
	require.NoError(t, err)
	assert.True(t, result)
}

func TestMutationResolver_DeleteChild_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Mutation().DeleteChild(ctx, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestMutationResolver_DeleteChild_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Mutation().DeleteChild(ctx, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestMutationResolver_DeleteChild_InvalidID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Mutation().DeleteChild(ctx, "invalid-uuid")

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "invalid child ID")
}

func TestMutationResolver_DeleteChild_DeleteError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.DeleteChildFunc = func(ctx context.Context, id uuid.UUID) error {
		return errors.New("delete error")
	}

	// Execute
	result, err := resolver.Mutation().DeleteChild(ctx, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to delete child")
}

func TestMutationResolver_DeleteChild_NilContext(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	childID := uuid.New()
	childIDStr := childID.String()

	// Execute
	result, err := resolver.Mutation().DeleteChild(nil, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "nil context")
}

func TestMutationResolver_AddChildToParent(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "parent:update", permission)
		return true, nil
	}

	mockFamilyService.AddChildToParentFunc = func(ctx context.Context, pID, cID uuid.UUID) error {
		assert.Equal(t, parentID, pID)
		assert.Equal(t, childID, cID)
		return nil
	}

	// Execute
	result, err := resolver.Mutation().AddChildToParent(ctx, parentIDStr, childIDStr)

	// Assert
	require.NoError(t, err)
	assert.True(t, result)
}

func TestMutationResolver_AddChildToParent_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Mutation().AddChildToParent(ctx, parentIDStr, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestMutationResolver_AddChildToParent_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Mutation().AddChildToParent(ctx, parentIDStr, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestMutationResolver_AddChildToParent_InvalidParentID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Mutation().AddChildToParent(ctx, "invalid-uuid", childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "invalid parent ID")
}

func TestMutationResolver_AddChildToParent_InvalidChildID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Mutation().AddChildToParent(ctx, parentIDStr, "invalid-uuid")

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "invalid child ID")
}

func TestMutationResolver_AddChildToParent_ServiceError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.AddChildToParentFunc = func(ctx context.Context, pID, cID uuid.UUID) error {
		return errors.New("service error")
	}

	// Execute
	result, err := resolver.Mutation().AddChildToParent(ctx, parentIDStr, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to add child to parent")
}

func TestMutationResolver_AddChildToParent_NilContext(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Execute
	result, err := resolver.Mutation().AddChildToParent(nil, parentIDStr, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "nil context")
}

func TestMutationResolver_RemoveChildFromParent(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "parent:update", permission)
		return true, nil
	}

	mockFamilyService.RemoveChildFromParentFunc = func(ctx context.Context, pID, cID uuid.UUID) error {
		assert.Equal(t, parentID, pID)
		assert.Equal(t, childID, cID)
		return nil
	}

	// Execute
	result, err := resolver.Mutation().RemoveChildFromParent(ctx, parentIDStr, childIDStr)

	// Assert
	require.NoError(t, err)
	assert.True(t, result)
}

func TestMutationResolver_RemoveChildFromParent_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Mutation().RemoveChildFromParent(ctx, parentIDStr, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestMutationResolver_RemoveChildFromParent_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Mutation().RemoveChildFromParent(ctx, parentIDStr, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestMutationResolver_RemoveChildFromParent_InvalidParentID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	childID := uuid.New()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Mutation().RemoveChildFromParent(ctx, "invalid-uuid", childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "invalid parent ID")
}

func TestMutationResolver_RemoveChildFromParent_InvalidChildID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Mutation().RemoveChildFromParent(ctx, parentIDStr, "invalid-uuid")

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "invalid child ID")
}

func TestMutationResolver_RemoveChildFromParent_ServiceError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.RemoveChildFromParentFunc = func(ctx context.Context, pID, cID uuid.UUID) error {
		return errors.New("service error")
	}

	// Execute
	result, err := resolver.Mutation().RemoveChildFromParent(ctx, parentIDStr, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to remove child from parent")
}

func TestMutationResolver_RemoveChildFromParent_NilContext(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	parentID := uuid.New()
	childID := uuid.New()
	parentIDStr := parentID.String()
	childIDStr := childID.String()

	// Execute
	result, err := resolver.Mutation().RemoveChildFromParent(nil, parentIDStr, childIDStr)

	// Assert
	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "nil context")
}

func TestQueryResolver_Parents(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Create test parents
	parent1 := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	parent2 := domain.NewParent("Jane", "Smith", "jane.smith@example.com", time.Now().AddDate(-25, 0, 0))
	parents := []*domain.Parent{parent1, parent2}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       0,
		PageSize:   10,
		TotalCount: 2,
		HasNext:    false,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "parent:list", permission)
		return true, nil
	}

	mockFamilyService.ListParentsFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
		// Verify options
		assert.Equal(t, 0, options.Pagination.Page)
		assert.Equal(t, 10, options.Pagination.PageSize)
		return parents, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().Parents(ctx, nil, nil, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Edges, 2)
	assert.Equal(t, parent1, result.Edges[0].Node)
	assert.Equal(t, parent2, result.Edges[1].Node)
}

func TestQueryResolver_Parents_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Query().Parents(ctx, nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestQueryResolver_Parents_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Query().Parents(ctx, nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestQueryResolver_Parents_WithFilter(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Create test parents
	parent1 := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	parents := []*domain.Parent{parent1}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       0,
		PageSize:   10,
		TotalCount: 1,
		HasNext:    false,
	}

	// Create filter
	firstName := "John"
	lastName := "Doe"
	email := "john.doe@example.com"
	minAge := 25
	maxAge := 35
	filter := &graphql.ParentFilter{
		FirstName: &firstName,
		LastName:  &lastName,
		Email:     &email,
		MinAge:    &minAge,
		MaxAge:    &maxAge,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListParentsFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
		// Verify filter options
		assert.Equal(t, firstName, options.Filter.FirstName)
		assert.Equal(t, lastName, options.Filter.LastName)
		assert.Equal(t, email, options.Filter.Email)
		assert.Equal(t, minAge, options.Filter.MinAge)
		assert.Equal(t, maxAge, options.Filter.MaxAge)
		return parents, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().Parents(ctx, filter, nil, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, parent1, result.Edges[0].Node)
}

func TestQueryResolver_Parents_WithPagination(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Create test parents
	parent1 := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	parents := []*domain.Parent{parent1}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       2,
		PageSize:   5,
		TotalCount: 15,
		HasNext:    true,
	}

	// Create pagination
	page := 2
	pageSize := 5
	pagination := &graphql.PaginationInput{
		Page:     &page,
		PageSize: &pageSize,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListParentsFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
		// Verify pagination options
		assert.Equal(t, page, options.Pagination.Page)
		assert.Equal(t, pageSize, options.Pagination.PageSize)
		return parents, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().Parents(ctx, nil, pagination, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 15, result.TotalCount)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, parent1, result.Edges[0].Node)
	assert.True(t, result.PageInfo.HasNextPage)
	assert.True(t, result.PageInfo.HasPreviousPage)
}

func TestQueryResolver_Parents_WithSort(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Create test parents
	parent1 := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	parents := []*domain.Parent{parent1}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       0,
		PageSize:   10,
		TotalCount: 1,
		HasNext:    false,
	}

	// Create sort
	field := "firstName"
	direction := graphql.SortDirectionAsc
	sort := &graphql.SortInput{
		Field:     &field,
		Direction: &direction,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListParentsFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
		// Verify sort options
		assert.Equal(t, field, options.Sort.Field)
		assert.Equal(t, "asc", options.Sort.Direction)
		return parents, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().Parents(ctx, nil, nil, sort)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, parent1, result.Edges[0].Node)
}

func TestQueryResolver_Parents_ListError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListParentsFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
		return nil, nil, errors.New("list error")
	}

	// Execute
	result, err := resolver.Query().Parents(ctx, nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list parents")
}

func TestQueryResolver_Children(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	child2 := domain.NewChild("Jack", "Doe", time.Now().AddDate(-3, 0, 0), parentID)
	children := []*domain.Child{child1, child2}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       0,
		PageSize:   10,
		TotalCount: 2,
		HasNext:    false,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "child:list", permission)
		return true, nil
	}

	mockFamilyService.ListChildrenFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		// Verify options
		assert.Equal(t, 0, options.Pagination.Page)
		assert.Equal(t, 10, options.Pagination.PageSize)
		return children, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().Children(ctx, nil, nil, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Edges, 2)
	assert.Equal(t, child1, result.Edges[0].Node)
	assert.Equal(t, child2, result.Edges[1].Node)
}

func TestQueryResolver_Children_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Query().Children(ctx, nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestQueryResolver_Children_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Query().Children(ctx, nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestQueryResolver_Children_WithFilter(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	children := []*domain.Child{child1}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       0,
		PageSize:   10,
		TotalCount: 1,
		HasNext:    false,
	}

	// Create filter
	firstName := "Jane"
	lastName := "Doe"
	minAge := 3
	maxAge := 10
	filter := &graphql.ChildFilter{
		FirstName: &firstName,
		LastName:  &lastName,
		MinAge:    &minAge,
		MaxAge:    &maxAge,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListChildrenFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		// Verify filter options
		assert.Equal(t, firstName, options.Filter.FirstName)
		assert.Equal(t, lastName, options.Filter.LastName)
		assert.Equal(t, minAge, options.Filter.MinAge)
		assert.Equal(t, maxAge, options.Filter.MaxAge)
		return children, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().Children(ctx, filter, nil, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, child1, result.Edges[0].Node)
}

func TestQueryResolver_Children_WithPagination(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	children := []*domain.Child{child1}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       2,
		PageSize:   5,
		TotalCount: 15,
		HasNext:    true,
	}

	// Create pagination
	page := 2
	pageSize := 5
	pagination := &graphql.PaginationInput{
		Page:     &page,
		PageSize: &pageSize,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListChildrenFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		// Verify pagination options
		assert.Equal(t, page, options.Pagination.Page)
		assert.Equal(t, pageSize, options.Pagination.PageSize)
		return children, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().Children(ctx, nil, pagination, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 15, result.TotalCount)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, child1, result.Edges[0].Node)
	assert.True(t, result.PageInfo.HasNextPage)
	assert.True(t, result.PageInfo.HasPreviousPage)
}

func TestQueryResolver_Children_WithSort(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	children := []*domain.Child{child1}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       0,
		PageSize:   10,
		TotalCount: 1,
		HasNext:    false,
	}

	// Create sort
	field := "firstName"
	direction := graphql.SortDirectionAsc
	sort := &graphql.SortInput{
		Field:     &field,
		Direction: &direction,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListChildrenFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		// Verify sort options
		assert.Equal(t, field, options.Sort.Field)
		assert.Equal(t, "asc", options.Sort.Direction)
		return children, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().Children(ctx, nil, nil, sort)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, child1, result.Edges[0].Node)
}

func TestQueryResolver_Children_ListError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListChildrenFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		return nil, nil, errors.New("list error")
	}

	// Execute
	result, err := resolver.Query().Children(ctx, nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list children")
}

func TestQueryResolver_ChildrenByParent(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	child2 := domain.NewChild("Jack", "Doe", time.Now().AddDate(-3, 0, 0), parentID)
	children := []*domain.Child{child1, child2}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       0,
		PageSize:   10,
		TotalCount: 2,
		HasNext:    false,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		assert.Equal(t, "child:list", permission)
		return true, nil
	}

	mockFamilyService.ListChildrenByParentIDFunc = func(ctx context.Context, pID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		// Verify parent ID and options
		assert.Equal(t, parentID, pID)
		assert.Equal(t, 0, options.Pagination.Page)
		assert.Equal(t, 10, options.Pagination.PageSize)
		return children, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().ChildrenByParent(ctx, parentIDStr, nil, nil, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Edges, 2)
	assert.Equal(t, child1, result.Edges[0].Node)
	assert.Equal(t, child2, result.Edges[1].Node)
}

func TestQueryResolver_ChildrenByParent_AuthError(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, errors.New("authorization error")
	}

	// Execute
	result, err := resolver.Query().ChildrenByParent(ctx, parentIDStr, nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check authorization")
}

func TestQueryResolver_ChildrenByParent_Unauthorized(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return false, nil
	}

	// Execute
	result, err := resolver.Query().ChildrenByParent(ctx, parentIDStr, nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestQueryResolver_ChildrenByParent_InvalidParentID(t *testing.T) {
	// Setup
	resolver, _, mockAuthService := setupResolverTest(t)
	ctx := context.Background()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	// Execute
	result, err := resolver.Query().ChildrenByParent(ctx, "invalid-uuid", nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid parent ID")
}

func TestQueryResolver_ChildrenByParent_WithFilter(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	children := []*domain.Child{child1}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       0,
		PageSize:   10,
		TotalCount: 1,
		HasNext:    false,
	}

	// Create filter
	firstName := "Jane"
	lastName := "Doe"
	minAge := 3
	maxAge := 10
	filter := &graphql.ChildFilter{
		FirstName: &firstName,
		LastName:  &lastName,
		MinAge:    &minAge,
		MaxAge:    &maxAge,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListChildrenByParentIDFunc = func(ctx context.Context, pID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		// Verify parent ID and filter options
		assert.Equal(t, parentID, pID)
		assert.Equal(t, firstName, options.Filter.FirstName)
		assert.Equal(t, lastName, options.Filter.LastName)
		assert.Equal(t, minAge, options.Filter.MinAge)
		assert.Equal(t, maxAge, options.Filter.MaxAge)
		return children, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().ChildrenByParent(ctx, parentIDStr, filter, nil, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, child1, result.Edges[0].Node)
}

func TestQueryResolver_ChildrenByParent_WithPagination(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	children := []*domain.Child{child1}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       2,
		PageSize:   5,
		TotalCount: 15,
		HasNext:    true,
	}

	// Create pagination
	page := 2
	pageSize := 5
	pagination := &graphql.PaginationInput{
		Page:     &page,
		PageSize: &pageSize,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListChildrenByParentIDFunc = func(ctx context.Context, pID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		// Verify parent ID and pagination options
		assert.Equal(t, parentID, pID)
		assert.Equal(t, page, options.Pagination.Page)
		assert.Equal(t, pageSize, options.Pagination.PageSize)
		return children, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().ChildrenByParent(ctx, parentIDStr, nil, pagination, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 15, result.TotalCount)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, child1, result.Edges[0].Node)
	assert.True(t, result.PageInfo.HasNextPage)
	assert.True(t, result.PageInfo.HasPreviousPage)
}

func TestQueryResolver_ChildrenByParent_WithSort(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Create test children
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	children := []*domain.Child{child1}

	// Create paged result
	pagedResult := &ports.PagedResult{
		Page:       0,
		PageSize:   10,
		TotalCount: 1,
		HasNext:    false,
	}

	// Create sort
	field := "firstName"
	direction := graphql.SortDirectionAsc
	sort := &graphql.SortInput{
		Field:     &field,
		Direction: &direction,
	}

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListChildrenByParentIDFunc = func(ctx context.Context, pID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		// Verify parent ID and sort options
		assert.Equal(t, parentID, pID)
		assert.Equal(t, field, options.Sort.Field)
		assert.Equal(t, "asc", options.Sort.Direction)
		return children, pagedResult, nil
	}

	// Execute
	result, err := resolver.Query().ChildrenByParent(ctx, parentIDStr, nil, nil, sort)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, child1, result.Edges[0].Node)
}

func TestQueryResolver_ChildrenByParent_ListError(t *testing.T) {
	// Setup
	resolver, mockFamilyService, mockAuthService := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	// Configure mocks
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, permission string) (bool, error) {
		return true, nil
	}

	mockFamilyService.ListChildrenByParentIDFunc = func(ctx context.Context, pID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		return nil, nil, errors.New("list error")
	}

	// Execute
	result, err := resolver.Query().ChildrenByParent(ctx, parentIDStr, nil, nil, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list children by parent")
}

func TestParentConnectionResolver_Edges(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()

	// Create test parent connection
	parent1 := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	parent2 := domain.NewParent("Jane", "Smith", "jane.smith@example.com", time.Now().AddDate(-25, 0, 0))

	edges := []graphql.ParentEdge{
		{Node: parent1, Cursor: "cursor1"},
		{Node: parent2, Cursor: "cursor2"},
	}

	connection := &graphql.ParentConnection{
		Edges: edges,
		PageInfo: &graphql.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
		},
		TotalCount: 2,
	}

	// Execute
	result, err := resolver.ParentConnection().Edges(ctx, connection)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, edges, result)
}

func TestParentConnectionResolver_PageInfo(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()

	// Create test parent connection
	pageInfo := &graphql.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
	}

	connection := &graphql.ParentConnection{
		PageInfo:   pageInfo,
		TotalCount: 2,
	}

	// Execute
	result, err := resolver.ParentConnection().PageInfo(ctx, connection)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, pageInfo, result)
}

func TestParentConnectionResolver_TotalCount(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()

	// Create test parent connection
	connection := &graphql.ParentConnection{
		TotalCount: 42,
	}

	// Execute
	result, err := resolver.ParentConnection().TotalCount(ctx, connection)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestChildConnectionResolver_Edges(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()
	parentID := uuid.New()

	// Create test child connection
	child1 := domain.NewChild("Jane", "Doe", time.Now().AddDate(-5, 0, 0), parentID)
	child2 := domain.NewChild("Jack", "Doe", time.Now().AddDate(-3, 0, 0), parentID)

	edges := []graphql.ChildEdge{
		{Node: child1, Cursor: "cursor1"},
		{Node: child2, Cursor: "cursor2"},
	}

	connection := &graphql.ChildConnection{
		Edges: edges,
		PageInfo: &graphql.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
		},
		TotalCount: 2,
	}

	// Execute
	result, err := resolver.ChildConnection().Edges(ctx, connection)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, edges, result)
}

func TestChildConnectionResolver_PageInfo(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()

	// Create test child connection
	pageInfo := &graphql.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
	}

	connection := &graphql.ChildConnection{
		PageInfo:   pageInfo,
		TotalCount: 2,
	}

	// Execute
	result, err := resolver.ChildConnection().PageInfo(ctx, connection)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, pageInfo, result)
}

func TestChildConnectionResolver_TotalCount(t *testing.T) {
	// Setup
	resolver, _, _ := setupResolverTest(t)
	ctx := context.Background()

	// Create test child connection
	connection := &graphql.ChildConnection{
		TotalCount: 42,
	}

	// Execute
	result, err := resolver.ChildConnection().TotalCount(ctx, connection)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 42, result)
}
