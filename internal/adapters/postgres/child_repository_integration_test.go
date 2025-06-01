package postgres_test

import (
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/adapters/postgres"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChildRepositoryIntegration tests the PostgreSQL child repository with a real PostgreSQL database
func TestChildRepositoryIntegration(t *testing.T) {
	// Skip if short flag is set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test repositories using the helper
	factory, ctx, cleanup := postgres.SetupTestRepositories(t)
	defer cleanup()

	// Get repositories from the factory
	parentRepo := factory.NewParentRepository()
	childRepo := factory.NewChildRepository()

	// Create a parent for testing
	parent := domain.NewParent("Test", "Parent", "test.parent@example.com", time.Now().AddDate(-30, 0, 0))
	err := parentRepo.Create(ctx, parent)
	require.NoError(t, err, "Failed to create test parent")

	// Test creating a child
	t.Run("Create", func(t *testing.T) {
		// Create a child
		child := domain.NewChild("John", "Doe", time.Now().AddDate(-5, 0, 0), parent.ID)

		// Save child to repository
		err := childRepo.Create(ctx, child)
		require.NoError(t, err, "Failed to create child")

		// Retrieve child from repository
		retrievedChild, err := childRepo.GetByID(ctx, child.ID)
		require.NoError(t, err, "Failed to retrieve child")
		assert.Equal(t, child.ID, retrievedChild.ID)
		assert.Equal(t, child.FirstName, retrievedChild.FirstName)
		assert.Equal(t, child.LastName, retrievedChild.LastName)
		assert.Equal(t, child.ParentID, retrievedChild.ParentID)
	})

	// Test updating a child
	t.Run("Update", func(t *testing.T) {
		// Create a child
		child := domain.NewChild("Jane", "Smith", time.Now().AddDate(-3, 0, 0), parent.ID)

		// Save child to repository
		err := childRepo.Create(ctx, child)
		require.NoError(t, err, "Failed to create child")

		// Update child
		child.FirstName = "Janet"
		child.LastName = "Johnson"

		// Save updated child to repository
		err = childRepo.Update(ctx, child)
		require.NoError(t, err, "Failed to update child")

		// Retrieve updated child from repository
		retrievedChild, err := childRepo.GetByID(ctx, child.ID)
		require.NoError(t, err, "Failed to retrieve child")
		assert.Equal(t, child.ID, retrievedChild.ID)
		assert.Equal(t, "Janet", retrievedChild.FirstName)
		assert.Equal(t, "Johnson", retrievedChild.LastName)
	})

	// Test deleting a child
	t.Run("Delete", func(t *testing.T) {
		// Create a child
		child := domain.NewChild("Bob", "Brown", time.Now().AddDate(-2, 0, 0), parent.ID)

		// Save child to repository
		err := childRepo.Create(ctx, child)
		require.NoError(t, err, "Failed to create child")

		// Delete child
		err = childRepo.Delete(ctx, child.ID)
		require.NoError(t, err, "Failed to delete child")

		// Try to retrieve deleted child
		_, err = childRepo.GetByID(ctx, child.ID)
		require.Error(t, err, "Expected error when retrieving deleted child")
		assert.Contains(t, err.Error(), "child not found")
	})

	// Test listing children
	t.Run("List", func(t *testing.T) {
		// Create multiple children
		child1 := domain.NewChild("Alice", "Anderson", time.Now().AddDate(-1, 0, 0), parent.ID)
		child2 := domain.NewChild("Bob", "Baker", time.Now().AddDate(-2, 0, 0), parent.ID)
		child3 := domain.NewChild("Charlie", "Clark", time.Now().AddDate(-3, 0, 0), parent.ID)

		// Save children to repository
		err := childRepo.Create(ctx, child1)
		require.NoError(t, err, "Failed to create child1")
		err = childRepo.Create(ctx, child2)
		require.NoError(t, err, "Failed to create child2")
		err = childRepo.Create(ctx, child3)
		require.NoError(t, err, "Failed to create child3")

		// List all children
		options := ports.QueryOptions{
			Pagination: ports.PaginationOptions{
				Page:     0,
				PageSize: 10,
			},
		}

		children, pagedResult, err := childRepo.List(ctx, options)
		require.NoError(t, err, "Failed to list children")
		assert.GreaterOrEqual(t, len(children), 3, "Expected at least 3 children")
		assert.GreaterOrEqual(t, pagedResult.TotalCount, int64(3), "Expected total count of at least 3")
	})

	// Test filtering children
	t.Run("Filter", func(t *testing.T) {
		// Create a unique child for this test
		child := domain.NewChild("Unique", "Name", time.Now().AddDate(-4, 0, 0), parent.ID)

		// Save child to repository
		err := childRepo.Create(ctx, child)
		require.NoError(t, err, "Failed to create child")

		// Filter by first name
		options := ports.QueryOptions{
			Filter: ports.FilterOptions{
				FirstName: "Unique",
			},
			Pagination: ports.PaginationOptions{
				Page:     0,
				PageSize: 10,
			},
		}

		children, pagedResult, err := childRepo.List(ctx, options)
		require.NoError(t, err, "Failed to filter children")
		assert.Equal(t, 1, len(children), "Expected 1 child")
		assert.Equal(t, int64(1), pagedResult.TotalCount, "Expected total count of 1")
		assert.Equal(t, "Unique", children[0].FirstName, "Expected first name to be 'Unique'")
	})

	// Test listing children by parent ID
	t.Run("ListByParentID", func(t *testing.T) {
		// Create a new parent
		newParent := domain.NewParent("New", "Parent", "new.parent@example.com", time.Now().AddDate(-35, 0, 0))
		err := parentRepo.Create(ctx, newParent)
		require.NoError(t, err, "Failed to create new parent")

		// Create children for the new parent
		child1 := domain.NewChild("Child1", "ForNewParent", time.Now().AddDate(-1, 0, 0), newParent.ID)
		child2 := domain.NewChild("Child2", "ForNewParent", time.Now().AddDate(-2, 0, 0), newParent.ID)

		// Save children to repository
		err = childRepo.Create(ctx, child1)
		require.NoError(t, err, "Failed to create child1 for new parent")
		err = childRepo.Create(ctx, child2)
		require.NoError(t, err, "Failed to create child2 for new parent")

		// List children by parent ID
		options := ports.QueryOptions{
			Pagination: ports.PaginationOptions{
				Page:     0,
				PageSize: 10,
			},
		}

		children, pagedResult, err := childRepo.ListByParentID(ctx, newParent.ID, options)
		require.NoError(t, err, "Failed to list children by parent ID")
		assert.Equal(t, 2, len(children), "Expected 2 children for the new parent")
		assert.Equal(t, int64(2), pagedResult.TotalCount, "Expected total count of 2")

		// Verify all children have the correct parent ID
		for _, child := range children {
			assert.Equal(t, newParent.ID, child.ParentID, "Expected child to have the correct parent ID")
		}
	})

	// Test counting children
	t.Run("Count", func(t *testing.T) {
		// Count all children
		count, err := childRepo.Count(ctx, ports.FilterOptions{})
		require.NoError(t, err, "Failed to count children")
		assert.GreaterOrEqual(t, count, int64(6), "Expected at least 6 children")

		// Count children with specific first name
		count, err = childRepo.Count(ctx, ports.FilterOptions{
			FirstName: "Unique",
		})
		require.NoError(t, err, "Failed to count children with filter")
		assert.Equal(t, int64(1), count, "Expected 1 child with first name 'Unique'")

		// Count children for a specific parent using ListByParentID
		options := ports.QueryOptions{
			Pagination: ports.PaginationOptions{
				Page:     0,
				PageSize: 10,
			},
		}
		_, pagedResult, err := childRepo.ListByParentID(ctx, parent.ID, options)
		require.NoError(t, err, "Failed to list children for parent")
		assert.GreaterOrEqual(t, pagedResult.TotalCount, int64(5), "Expected at least 5 children for the test parent")
	})

	// Test getting a non-existent child
	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := childRepo.GetByID(ctx, uuid.New())
		require.Error(t, err, "Expected error when retrieving non-existent child")
		assert.Contains(t, err.Error(), "child not found")
	})

	// Test updating a non-existent child
	t.Run("UpdateNonExistent", func(t *testing.T) {
		child := domain.NewChild("NonExistent", "Child", time.Now().AddDate(-5, 0, 0), parent.ID)
		err := childRepo.Update(ctx, child)
		require.Error(t, err, "Expected error when updating non-existent child")
		assert.Contains(t, err.Error(), "child not found")
	})

	// Test deleting a non-existent child
	t.Run("DeleteNonExistent", func(t *testing.T) {
		err := childRepo.Delete(ctx, uuid.New())
		require.Error(t, err, "Expected error when deleting non-existent child")
		assert.Contains(t, err.Error(), "child not found")
	})
}
