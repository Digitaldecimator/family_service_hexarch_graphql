package application_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/abitofhelp/family-service2/internal/adapters/mongodb"
	"github.com/abitofhelp/family-service2/internal/application"
	"github.com/abitofhelp/family-service2/internal/ports"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap/zaptest"
)

// setupMongoDBForIntegration sets up a MongoDB connection for integration testing
func setupMongoDBForIntegration(t *testing.T) (*mongo.Database, context.Context, func()) {
	// Get MongoDB URI from environment variable or use default
	mongoURI := os.Getenv("TEST_MONGODB_URI")
	if mongoURI == "" {
		mongoPassword := os.Getenv("MONGODB_ROOT_PASSWORD")
		if mongoPassword == "" {
			mongoPassword = os.Getenv("MONGO_INITDB_ROOT_PASSWORD") // Fallback to the old environment variable
		}
		if mongoPassword == "" {
			// Use a hardcoded password for testing purposes if environment variables are not set
			mongoPassword = "NVsHFXcxqUsMoEgiUnE7jvzXxhp3gn9nsgkXCsetAHLhcpyLRmWhKixUpfr3J7tE"
		}
		mongoURI = "mongodb://root:" + mongoPassword + "@localhost:27017/?authSource=admin"
	}

	// Get MongoDB database name from environment variable or use default
	dbName := os.Getenv("TEST_MONGODB_DATABASE")
	if dbName == "" {
		dbName = "family_service_integration_test"
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err, "Failed to connect to MongoDB")

	// Ping the database to verify connection
	err = client.Ping(ctx, nil)
	require.NoError(t, err, "Failed to ping MongoDB")

	// Get database
	db := client.Database(dbName)

	// Return cleanup function
	cleanup := func() {
		// Drop the database to clean up
		err := db.Drop(ctx)
		if err != nil {
			t.Logf("Failed to drop test database: %v", err)
		}

		// Disconnect from MongoDB
		err = client.Disconnect(ctx)
		if err != nil {
			t.Logf("Failed to disconnect from MongoDB: %v", err)
		}

		// Cancel context
		cancel()
	}

	return db, ctx, cleanup
}

// setupFamilyServiceIntegration sets up a family service with real MongoDB repositories for integration testing
func setupFamilyServiceIntegration(t *testing.T) (*application.FamilyService, context.Context, func()) {
	// Set up MongoDB connection
	db, ctx, cleanup := setupMongoDBForIntegration(t)

	// Create logger
	logger := zaptest.NewLogger(t)

	// Create a koanf instance with default values
	k := koanf.New(".")
	// Set default values for MongoDB timeouts
	k.Set("database.mongodb.index_timeout", 10000)      // 10 seconds
	k.Set("database.mongodb.connection_timeout", 10000) // 10 seconds
	k.Set("database.mongodb.ping_timeout", 5000)        // 5 seconds
	k.Set("database.mongodb.disconnect_timeout", 5000)  // 5 seconds

	// Create MongoDB config adapter
	mongoConfig := &KoanfMongoDBConfig{k: k}

	// Set MongoDB URI
	mongoURI := os.Getenv("TEST_MONGODB_URI")
	if mongoURI == "" {
		mongoPassword := os.Getenv("MONGODB_ROOT_PASSWORD")
		if mongoPassword == "" {
			mongoPassword = os.Getenv("MONGO_INITDB_ROOT_PASSWORD") // Fallback to the old environment variable
		}
		if mongoPassword == "" {
			// Use a hardcoded password for testing purposes if environment variables are not set
			mongoPassword = "NVsHFXcxqUsMoEgiUnE7jvzXxhp3gn9nsgkXCsetAHLhcpyLRmWhKixUpfr3J7tE"
		}
		mongoURI = "mongodb://root:" + mongoPassword + "@localhost:27017/?authSource=admin"
	}
	k.Set("database.mongodb.uri", mongoURI)

	// Create repositories
	parentRepo := mongodb.NewParentRepository(ctx, db, logger, mongoConfig)
	childRepo := mongodb.NewChildRepository(ctx, db, logger, mongoConfig)

	// Create transaction manager
	txManager := mongodb.NewTransactionManager(db.Client(), logger)

	// Create repository factory
	repoFactory := &mongoRepositoryFactory{
		parentRepo: parentRepo,
		childRepo:  childRepo,
		txManager:  txManager,
	}

	// Create validator
	validate := validator.New()

	// Create family service
	service := application.NewFamilyService(
		repoFactory,
		validate,
		logger,
	)

	return service, ctx, cleanup
}

// KoanfMongoDBConfig adapts koanf to implement the ports.MongoDBConfig interface
type KoanfMongoDBConfig struct {
	k *koanf.Koanf
}

// GetConnectionTimeout returns the connection timeout
func (c *KoanfMongoDBConfig) GetConnectionTimeout() time.Duration {
	return time.Duration(c.k.Int("database.mongodb.connection_timeout")) * time.Millisecond
}

// GetPingTimeout returns the ping timeout
func (c *KoanfMongoDBConfig) GetPingTimeout() time.Duration {
	return time.Duration(c.k.Int("database.mongodb.ping_timeout")) * time.Millisecond
}

// GetDisconnectTimeout returns the disconnect timeout
func (c *KoanfMongoDBConfig) GetDisconnectTimeout() time.Duration {
	return time.Duration(c.k.Int("database.mongodb.disconnect_timeout")) * time.Millisecond
}

// GetIndexTimeout returns the index timeout
func (c *KoanfMongoDBConfig) GetIndexTimeout() time.Duration {
	return time.Duration(c.k.Int("database.mongodb.index_timeout")) * time.Millisecond
}

// GetURI returns the MongoDB URI
func (c *KoanfMongoDBConfig) GetURI() string {
	return c.k.String("database.mongodb.uri")
}

// mongoRepositoryFactory is a simple implementation of the RepositoryFactory interface for MongoDB
type mongoRepositoryFactory struct {
	parentRepo ports.ParentRepository
	childRepo  ports.ChildRepository
	txManager  ports.TransactionManager
}

func (f *mongoRepositoryFactory) NewParentRepository() ports.ParentRepository {
	return f.parentRepo
}

func (f *mongoRepositoryFactory) NewChildRepository() ports.ChildRepository {
	return f.childRepo
}

func (f *mongoRepositoryFactory) GetTransactionManager() ports.TransactionManager {
	return f.txManager
}

// mockTransactionManager is a simple implementation of the TransactionManager interface for testing
type mockTransactionManager struct{}

func (m *mockTransactionManager) BeginTx(ctx context.Context) (context.Context, error) {
	// Just return the context as is for testing
	return ctx, nil
}

func (m *mockTransactionManager) CommitTx(ctx context.Context) error {
	// Do nothing for testing
	return nil
}

func (m *mockTransactionManager) RollbackTx(ctx context.Context) error {
	// Do nothing for testing
	return nil
}

// TestFamilyServiceIntegration tests the family service with real MongoDB repositories
func TestFamilyServiceIntegration(t *testing.T) {
	// Skip if short flag is set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up family service with real MongoDB repositories
	service, ctx, cleanup := setupFamilyServiceIntegration(t)
	defer cleanup()

	// Test creating a parent
	t.Run("CreateParent", func(t *testing.T) {
		firstName := "John"
		lastName := "Doe"
		email := "john.doe@example.com"
		birthDate := time.Now().AddDate(-30, 0, 0).Format(time.RFC3339)

		parent, err := service.CreateParent(ctx, firstName, lastName, email, birthDate)
		require.NoError(t, err, "Failed to create parent")
		assert.NotNil(t, parent, "Parent should not be nil")
		assert.Equal(t, firstName, parent.FirstName, "First name should match")
		assert.Equal(t, lastName, parent.LastName, "Last name should match")
		assert.Equal(t, email, parent.Email, "Email should match")

		// Verify parent was created by retrieving it
		retrievedParent, err := service.GetParentByID(ctx, parent.ID)
		require.NoError(t, err, "Failed to retrieve parent")
		assert.Equal(t, parent.ID, retrievedParent.ID, "Parent ID should match")
		assert.Equal(t, firstName, retrievedParent.FirstName, "First name should match")
		assert.Equal(t, lastName, retrievedParent.LastName, "Last name should match")
		assert.Equal(t, email, retrievedParent.Email, "Email should match")
	})

	// Test updating a parent
	t.Run("UpdateParent", func(t *testing.T) {
		// Create a parent first
		parent, err := service.CreateParent(
			ctx,
			"Jane",
			"Smith",
			"jane.smith@example.com",
			time.Now().AddDate(-25, 0, 0).Format(time.RFC3339),
		)
		require.NoError(t, err, "Failed to create parent")

		// Update the parent
		updatedParent, err := service.UpdateParent(
			ctx,
			parent.ID,
			"Janet",
			"Johnson",
			"janet.johnson@example.com",
			time.Now().AddDate(-26, 0, 0).Format(time.RFC3339),
		)
		require.NoError(t, err, "Failed to update parent")
		assert.Equal(t, parent.ID, updatedParent.ID, "Parent ID should match")
		assert.Equal(t, "Janet", updatedParent.FirstName, "First name should be updated")
		assert.Equal(t, "Johnson", updatedParent.LastName, "Last name should be updated")
		assert.Equal(t, "janet.johnson@example.com", updatedParent.Email, "Email should be updated")

		// Verify parent was updated by retrieving it
		retrievedParent, err := service.GetParentByID(ctx, parent.ID)
		require.NoError(t, err, "Failed to retrieve parent")
		assert.Equal(t, "Janet", retrievedParent.FirstName, "First name should be updated")
		assert.Equal(t, "Johnson", retrievedParent.LastName, "Last name should be updated")
		assert.Equal(t, "janet.johnson@example.com", retrievedParent.Email, "Email should be updated")
	})

	// Test creating a child
	t.Run("CreateChild", func(t *testing.T) {
		// Create a parent first
		parent, err := service.CreateParent(
			ctx,
			"Bob",
			"Brown",
			"bob.brown@example.com",
			time.Now().AddDate(-40, 0, 0).Format(time.RFC3339),
		)
		require.NoError(t, err, "Failed to create parent")

		// Create a child
		firstName := "Bobby"
		lastName := "Brown"
		birthDate := time.Now().AddDate(-10, 0, 0).Format(time.RFC3339)

		child, err := service.CreateChild(ctx, firstName, lastName, birthDate, parent.ID)
		require.NoError(t, err, "Failed to create child")
		assert.NotNil(t, child, "Child should not be nil")
		assert.Equal(t, firstName, child.FirstName, "First name should match")
		assert.Equal(t, lastName, child.LastName, "Last name should match")
		assert.Equal(t, parent.ID, child.ParentID, "Parent ID should match")

		// Verify child was created by retrieving it
		retrievedChild, err := service.GetChildByID(ctx, child.ID)
		require.NoError(t, err, "Failed to retrieve child")
		assert.Equal(t, child.ID, retrievedChild.ID, "Child ID should match")
		assert.Equal(t, firstName, retrievedChild.FirstName, "First name should match")
		assert.Equal(t, lastName, retrievedChild.LastName, "Last name should match")
		assert.Equal(t, parent.ID, retrievedChild.ParentID, "Parent ID should match")

		// Verify parent has the child
		retrievedParent, err := service.GetParentByID(ctx, parent.ID)
		require.NoError(t, err, "Failed to retrieve parent")
		assert.Len(t, retrievedParent.Children, 1, "Parent should have one child")
		assert.Equal(t, child.ID, retrievedParent.Children[0].ID, "Child ID should match")
	})

	// Test listing parents
	t.Run("ListParents", func(t *testing.T) {
		// Create multiple parents
		for i := 0; i < 5; i++ {
			_, err := service.CreateParent(
				ctx,
				"ListTest",
				"Parent"+string(rune('A'+i)),
				"list.test"+string(rune('a'+i))+"@example.com",
				time.Now().AddDate(-30-i, 0, 0).Format(time.RFC3339),
			)
			require.NoError(t, err, "Failed to create parent")
		}

		// List parents
		options := ports.QueryOptions{
			Filter: ports.FilterOptions{
				FirstName: "ListTest",
			},
			Pagination: ports.PaginationOptions{
				Page:     0,
				PageSize: 10,
			},
		}

		parents, pagedResult, err := service.ListParents(ctx, options)
		require.NoError(t, err, "Failed to list parents")
		assert.Len(t, parents, 5, "Should have 5 parents")
		assert.Equal(t, int64(5), pagedResult.TotalCount, "Total count should be 5")

		// Verify all parents have the correct first name
		for _, parent := range parents {
			assert.Equal(t, "ListTest", parent.FirstName, "First name should match")
		}
	})

	// Test listing children by parent ID
	t.Run("ListChildrenByParentID", func(t *testing.T) {
		// Create a parent
		parent, err := service.CreateParent(
			ctx,
			"ChildList",
			"Parent",
			"childlist.parent@example.com",
			time.Now().AddDate(-35, 0, 0).Format(time.RFC3339),
		)
		require.NoError(t, err, "Failed to create parent")

		// Create multiple children for the parent
		for i := 0; i < 3; i++ {
			_, err := service.CreateChild(
				ctx,
				"ChildList",
				"Child"+string(rune('A'+i)),
				time.Now().AddDate(-5-i, 0, 0).Format(time.RFC3339),
				parent.ID,
			)
			require.NoError(t, err, "Failed to create child")
		}

		// List children by parent ID
		options := ports.QueryOptions{
			Pagination: ports.PaginationOptions{
				Page:     0,
				PageSize: 10,
			},
		}

		children, pagedResult, err := service.ListChildrenByParentID(ctx, parent.ID, options)
		require.NoError(t, err, "Failed to list children by parent ID")
		assert.Len(t, children, 3, "Should have 3 children")
		assert.Equal(t, int64(3), pagedResult.TotalCount, "Total count should be 3")

		// Verify all children have the correct parent ID and first name
		for _, child := range children {
			assert.Equal(t, parent.ID, child.ParentID, "Parent ID should match")
			assert.Equal(t, "ChildList", child.FirstName, "First name should match")
		}
	})

	// Test deleting a parent
	t.Run("DeleteParent", func(t *testing.T) {
		// Create a parent
		parent, err := service.CreateParent(
			ctx,
			"Delete",
			"Parent",
			"delete.parent@example.com",
			time.Now().AddDate(-45, 0, 0).Format(time.RFC3339),
		)
		require.NoError(t, err, "Failed to create parent")

		// Create a child for the parent
		child, err := service.CreateChild(
			ctx,
			"Delete",
			"Child",
			time.Now().AddDate(-15, 0, 0).Format(time.RFC3339),
			parent.ID,
		)
		require.NoError(t, err, "Failed to create child")

		// Verify parent and child exist
		_, err = service.GetParentByID(ctx, parent.ID)
		require.NoError(t, err, "Parent should exist")
		_, err = service.GetChildByID(ctx, child.ID)
		require.NoError(t, err, "Child should exist")

		// Delete parent
		err = service.DeleteParent(ctx, parent.ID)
		require.NoError(t, err, "Failed to delete parent")

		// Verify parent and child are deleted
		_, err = service.GetParentByID(ctx, parent.ID)
		require.Error(t, err, "Parent should be deleted")
		assert.Contains(t, err.Error(), "not found", "Error should indicate parent not found")

		_, err = service.GetChildByID(ctx, child.ID)
		require.Error(t, err, "Child should be deleted")
		assert.Contains(t, err.Error(), "not found", "Error should indicate child not found")
	})

	// Test deleting a child
	t.Run("DeleteChild", func(t *testing.T) {
		// Create a parent
		parent, err := service.CreateParent(
			ctx,
			"DeleteChild",
			"Parent",
			"deletechild.parent@example.com",
			time.Now().AddDate(-50, 0, 0).Format(time.RFC3339),
		)
		require.NoError(t, err, "Failed to create parent")

		// Create a child for the parent
		child, err := service.CreateChild(
			ctx,
			"DeleteChild",
			"Child",
			time.Now().AddDate(-20, 0, 0).Format(time.RFC3339),
			parent.ID,
		)
		require.NoError(t, err, "Failed to create child")

		// Verify child exists
		_, err = service.GetChildByID(ctx, child.ID)
		require.NoError(t, err, "Child should exist")

		// Delete child
		err = service.DeleteChild(ctx, child.ID)
		require.NoError(t, err, "Failed to delete child")

		// Verify child is deleted
		_, err = service.GetChildByID(ctx, child.ID)
		require.Error(t, err, "Child should be deleted")
		assert.Contains(t, err.Error(), "not found", "Error should indicate child not found")

		// Verify parent still exists but doesn't have the child
		retrievedParent, err := service.GetParentByID(ctx, parent.ID)
		require.NoError(t, err, "Parent should still exist")
		assert.Len(t, retrievedParent.Children, 0, "Parent should have no children")
	})

	// Test validation errors
	t.Run("ValidationErrors", func(t *testing.T) {
		// Test creating a parent with invalid email
		_, err := service.CreateParent(
			ctx,
			"Invalid",
			"Email",
			"not-an-email",
			time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
		)
		require.Error(t, err, "Should fail with invalid email")
		assert.Contains(t, strings.ToLower(err.Error()), "email", "Error should mention email")

		// Test creating a parent with missing required fields
		_, err = service.CreateParent(
			ctx,
			"",
			"LastName",
			"email@example.com",
			time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
		)
		require.Error(t, err, "Should fail with missing first name")
		assert.Contains(t, strings.ToLower(err.Error()), "first name", "Error should mention first name")

		// Test creating a child with invalid birth date
		// Create a parent first
		parent, err := service.CreateParent(
			ctx,
			"Valid",
			"Parent",
			"valid.parent@example.com",
			time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
		)
		require.NoError(t, err, "Failed to create parent")

		_, err = service.CreateChild(
			ctx,
			"Invalid",
			"BirthDate",
			"not-a-date",
			parent.ID,
		)
		require.Error(t, err, "Should fail with invalid birth date")
		assert.Contains(t, strings.ToLower(err.Error()), "birth date", "Error should mention birth date")
	})

	// Test not found errors
	t.Run("NotFoundErrors", func(t *testing.T) {
		// Test getting a non-existent parent
		_, err := service.GetParentByID(ctx, uuid.New())
		require.Error(t, err, "Should fail with parent not found")
		assert.Contains(t, err.Error(), "not found", "Error should indicate parent not found")

		// Test getting a non-existent child
		_, err = service.GetChildByID(ctx, uuid.New())
		require.Error(t, err, "Should fail with child not found")
		assert.Contains(t, err.Error(), "not found", "Error should indicate child not found")

		// Test updating a non-existent parent
		_, err = service.UpdateParent(
			ctx,
			uuid.New(),
			"NonExistent",
			"Parent",
			"nonexistent.parent@example.com",
			time.Now().AddDate(-30, 0, 0).Format(time.RFC3339),
		)
		require.Error(t, err, "Should fail with parent not found")
		assert.Contains(t, err.Error(), "not found", "Error should indicate parent not found")

		// Test updating a non-existent child
		_, err = service.UpdateChild(
			ctx,
			uuid.New(),
			"NonExistent",
			"Child",
			time.Now().AddDate(-10, 0, 0).Format(time.RFC3339),
		)
		require.Error(t, err, "Should fail with child not found")
		assert.Contains(t, err.Error(), "not found", "Error should indicate child not found")

		// Test deleting a non-existent parent
		err = service.DeleteParent(ctx, uuid.New())
		require.Error(t, err, "Should fail with parent not found")
		assert.Contains(t, err.Error(), "not found", "Error should indicate parent not found")

		// Test deleting a non-existent child
		err = service.DeleteChild(ctx, uuid.New())
		require.Error(t, err, "Should fail with child not found")
		assert.Contains(t, err.Error(), "not found", "Error should indicate child not found")
	})
}
