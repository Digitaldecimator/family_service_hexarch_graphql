package postgres_test

import (
	"testing"
	"time"

	"github.com/abitofhelp/family-service2/internal/adapters/postgres"
	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/abitofhelp/family-service2/internal/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParentRepositoryIntegration tests the PostgreSQL parent repository with a real PostgreSQL database
func TestParentRepositoryIntegration(t *testing.T) {
	// Skip if short flag is set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test repositories using the helper
	factory, ctx, cleanup := postgres.SetupTestRepositories(t)
	defer cleanup()

	// Get parent repository from the factory
	repo := factory.NewParentRepository()

	// Test creating a parent
	t.Run("Create", func(t *testing.T) {
		// Create a parent
		parent := domain.NewParent("John", "Doe", "john.doe@example.com", time.Now().AddDate(-30, 0, 0))

		// Save parent to repository
		err := repo.Create(ctx, parent)
		require.NoError(t, err, "Failed to create parent")

		// Retrieve parent from repository
		retrievedParent, err := repo.GetByID(ctx, parent.ID)
		require.NoError(t, err, "Failed to retrieve parent")
		assert.Equal(t, parent.ID, retrievedParent.ID)
		assert.Equal(t, parent.FirstName, retrievedParent.FirstName)
		assert.Equal(t, parent.LastName, retrievedParent.LastName)
		assert.Equal(t, parent.Email, retrievedParent.Email)
	})

	// Test updating a parent
	t.Run("Update", func(t *testing.T) {
		// Create a parent
		parent := domain.NewParent("Jane", "Smith", "jane.smith@example.com", time.Now().AddDate(-25, 0, 0))

		// Save parent to repository
		err := repo.Create(ctx, parent)
		require.NoError(t, err, "Failed to create parent")

		// Update parent
		parent.FirstName = "Janet"
		parent.LastName = "Johnson"
		parent.Email = "janet.johnson@example.com"

		// Save updated parent to repository
		err = repo.Update(ctx, parent)
		require.NoError(t, err, "Failed to update parent")

		// Retrieve updated parent from repository
		retrievedParent, err := repo.GetByID(ctx, parent.ID)
		require.NoError(t, err, "Failed to retrieve parent")
		assert.Equal(t, parent.ID, retrievedParent.ID)
		assert.Equal(t, "Janet", retrievedParent.FirstName)
		assert.Equal(t, "Johnson", retrievedParent.LastName)
		assert.Equal(t, "janet.johnson@example.com", retrievedParent.Email)
	})

	// Test deleting a parent
	t.Run("Delete", func(t *testing.T) {
		// Create a parent
		parent := domain.NewParent("Bob", "Brown", "bob.brown@example.com", time.Now().AddDate(-40, 0, 0))

		// Save parent to repository
		err := repo.Create(ctx, parent)
		require.NoError(t, err, "Failed to create parent")

		// Delete parent
		err = repo.Delete(ctx, parent.ID)
		require.NoError(t, err, "Failed to delete parent")

		// Try to retrieve deleted parent
		_, err = repo.GetByID(ctx, parent.ID)
		require.Error(t, err, "Expected error when retrieving deleted parent")
		assert.Contains(t, err.Error(), "parent not found")
	})

	// Test listing parents
	t.Run("List", func(t *testing.T) {
		// Create multiple parents
		parent1 := domain.NewParent("Alice", "Anderson", "alice.anderson@example.com", time.Now().AddDate(-20, 0, 0))
		parent2 := domain.NewParent("Bob", "Baker", "bob.baker@example.com", time.Now().AddDate(-30, 0, 0))
		parent3 := domain.NewParent("Charlie", "Clark", "charlie.clark@example.com", time.Now().AddDate(-40, 0, 0))

		// Save parents to repository
		err := repo.Create(ctx, parent1)
		require.NoError(t, err, "Failed to create parent1")
		err = repo.Create(ctx, parent2)
		require.NoError(t, err, "Failed to create parent2")
		err = repo.Create(ctx, parent3)
		require.NoError(t, err, "Failed to create parent3")

		// List all parents
		options := ports.QueryOptions{
			Pagination: ports.PaginationOptions{
				Page:     0,
				PageSize: 10,
			},
		}

		parents, pagedResult, err := repo.List(ctx, options)
		require.NoError(t, err, "Failed to list parents")
		assert.GreaterOrEqual(t, len(parents), 3, "Expected at least 3 parents")
		assert.GreaterOrEqual(t, pagedResult.TotalCount, int64(3), "Expected total count of at least 3")
	})

	// Test filtering parents
	t.Run("Filter", func(t *testing.T) {
		// Create a unique parent for this test
		parent := domain.NewParent("Unique", "Name", "unique.name@example.com", time.Now().AddDate(-35, 0, 0))

		// Save parent to repository
		err := repo.Create(ctx, parent)
		require.NoError(t, err, "Failed to create parent")

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

		parents, pagedResult, err := repo.List(ctx, options)
		require.NoError(t, err, "Failed to filter parents")
		assert.Equal(t, 1, len(parents), "Expected 1 parent")
		assert.Equal(t, int64(1), pagedResult.TotalCount, "Expected total count of 1")
		assert.Equal(t, "Unique", parents[0].FirstName, "Expected first name to be 'Unique'")
	})

	// Test counting parents
	t.Run("Count", func(t *testing.T) {
		// Count all parents
		count, err := repo.Count(ctx, ports.FilterOptions{})
		require.NoError(t, err, "Failed to count parents")
		assert.GreaterOrEqual(t, count, int64(5), "Expected at least 5 parents")

		// Count parents with specific first name
		count, err = repo.Count(ctx, ports.FilterOptions{
			FirstName: "Unique",
		})
		require.NoError(t, err, "Failed to count parents with filter")
		assert.Equal(t, int64(1), count, "Expected 1 parent with first name 'Unique'")
	})

	// Test getting a non-existent parent
	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		require.Error(t, err, "Expected error when retrieving non-existent parent")
		assert.Contains(t, err.Error(), "parent not found")
	})

	// Test updating a non-existent parent
	t.Run("UpdateNonExistent", func(t *testing.T) {
		parent := domain.NewParent("NonExistent", "Parent", "nonexistent.parent@example.com", time.Now().AddDate(-30, 0, 0))
		err := repo.Update(ctx, parent)
		require.Error(t, err, "Expected error when updating non-existent parent")
		assert.Contains(t, err.Error(), "parent not found")
	})

	// Test deleting a non-existent parent
	t.Run("DeleteNonExistent", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		require.Error(t, err, "Expected error when deleting non-existent parent")
		assert.Contains(t, err.Error(), "parent not found")
	})
}
