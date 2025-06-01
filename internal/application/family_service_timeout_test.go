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

// setupTimeoutTest sets up a test environment with mocked repositories that can simulate timeouts
func setupTimeoutTest(t *testing.T) (
	*application.FamilyService,
	*mocks.MockRepositoryFactory,
	*mocks.MockParentRepository,
	*mocks.MockChildRepository,
	context.Context,
) {
	// Create mocks
	repoFactory := mocks.NewMockRepositoryFactory()
	parentRepo := repoFactory.GetMockParentRepository()
	childRepo := repoFactory.GetMockChildRepository()
	validate := validator.New()
	logger := zaptest.NewLogger(t)

	// Create service
	service := application.NewFamilyService(
		repoFactory,
		validate,
		logger,
	)

	// Create context with a short timeout
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	return service, repoFactory, parentRepo, childRepo, ctx
}

// TestParentRepositoryTimeout tests the behavior when parent repository operations time out
func TestParentRepositoryTimeout(t *testing.T) {
	// Arrange
	service, _, parentRepo, _, ctx := setupTimeoutTest(t)

	// Set up the mock to simulate a timeout
	parentRepo.GetByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
		return nil, context.DeadlineExceeded
	}

	// Act
	parent, err := service.GetParentByID(ctx, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Nil(t, parent)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(errors.Unwrap(err), context.DeadlineExceeded) ||
		err.Error() == "context deadline exceeded" ||
		err.Error() == "Parent not found")
}

// TestChildRepositoryTimeout tests the behavior when child repository operations time out
func TestChildRepositoryTimeout(t *testing.T) {
	// Arrange
	service, _, _, childRepo, ctx := setupTimeoutTest(t)

	// Set up the mock to simulate a timeout
	childRepo.GetByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
		return nil, context.DeadlineExceeded
	}

	// Act
	child, err := service.GetChildByID(ctx, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Nil(t, child)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(errors.Unwrap(err), context.DeadlineExceeded) ||
		err.Error() == "context deadline exceeded" ||
		err.Error() == "Child not found")
}

// TestCreateParentTimeout tests the behavior when creating a parent times out
func TestCreateParentTimeout(t *testing.T) {
	// Arrange
	service, _, parentRepo, _, ctx := setupTimeoutTest(t)

	// Set up the mock to simulate a timeout
	parentRepo.CreateFunc = func(ctx context.Context, parent *domain.Parent) error {
		return context.DeadlineExceeded
	}

	// Act
	firstName := "John"
	lastName := "Doe"
	email := "john.doe@example.com"
	birthDate := time.Now().AddDate(-30, 0, 0).Format(time.RFC3339)

	parent, err := service.CreateParent(ctx, firstName, lastName, email, birthDate)

	// Assert
	require.Error(t, err)
	assert.Nil(t, parent)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(errors.Unwrap(err), context.DeadlineExceeded) ||
		err.Error() == "context deadline exceeded" ||
		err.Error() == "Failed to create parent")
}

// TestCreateChildTimeout tests the behavior when creating a child times out
func TestCreateChildTimeout(t *testing.T) {
	// Arrange
	service, repoFactory, _, childRepo, ctx := setupTimeoutTest(t)

	// Create a parent first
	parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))
	repoFactory.GetMockParentRepository().AddTestParent(parent)

	// Set up the mock to simulate a timeout
	childRepo.CreateFunc = func(ctx context.Context, child *domain.Child) error {
		return context.DeadlineExceeded
	}

	// Act
	firstName := "Jane"
	lastName := "Doe"
	birthDate := time.Now().AddDate(-5, 0, 0).Format(time.RFC3339)

	child, err := service.CreateChild(ctx, firstName, lastName, birthDate, parent.ID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, child)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(errors.Unwrap(err), context.DeadlineExceeded) ||
		err.Error() == "context deadline exceeded" ||
		err.Error() == "Failed to create child")
}

// TestListParentsTimeout tests the behavior when listing parents times out
func TestListParentsTimeout(t *testing.T) {
	// Arrange
	service, _, parentRepo, _, ctx := setupTimeoutTest(t)

	// Set up the mock to simulate a timeout
	parentRepo.ListFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
		return nil, nil, context.DeadlineExceeded
	}

	// Act
	options := ports.QueryOptions{
		Pagination: ports.PaginationOptions{
			Page:     0,
			PageSize: 10,
		},
	}

	parents, pagedResult, err := service.ListParents(ctx, options)

	// Assert
	require.Error(t, err)
	assert.Nil(t, parents)
	assert.Nil(t, pagedResult)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(errors.Unwrap(err), context.DeadlineExceeded) ||
		err.Error() == "context deadline exceeded" ||
		err.Error() == "Failed to list parents")
}

// TestListChildrenTimeout tests the behavior when listing children times out
func TestListChildrenTimeout(t *testing.T) {
	// Arrange
	service, _, _, childRepo, ctx := setupTimeoutTest(t)

	// Set up the mock to simulate a timeout
	childRepo.ListFunc = func(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
		return nil, nil, context.DeadlineExceeded
	}

	// Act
	options := ports.QueryOptions{
		Pagination: ports.PaginationOptions{
			Page:     0,
			PageSize: 10,
		},
	}

	children, pagedResult, err := service.ListChildren(ctx, options)

	// Assert
	require.Error(t, err)
	assert.Nil(t, children)
	assert.Nil(t, pagedResult)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(errors.Unwrap(err), context.DeadlineExceeded) ||
		err.Error() == "context deadline exceeded" ||
		err.Error() == "Failed to list children")
}

// TestTransactionTimeout tests the behavior when a transaction times out
func TestTransactionTimeout(t *testing.T) {
	// Arrange
	service, repoFactory, _, _, ctx := setupTimeoutTest(t)

	// Set up the mock transaction manager to simulate a timeout
	txManager := repoFactory.GetTransactionManager().(*mocks.MockTransactionManager)
	txManager.BeginTxFunc = func(ctx context.Context) (context.Context, error) {
		return ctx, context.DeadlineExceeded
	}

	// Act
	err := service.DeleteParent(ctx, uuid.New())

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(errors.Unwrap(err), context.DeadlineExceeded) ||
		err.Error() == "context deadline exceeded" ||
		err.Error() == "Failed to begin transaction")
}
