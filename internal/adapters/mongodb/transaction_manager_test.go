package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/adapters/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap/zaptest"
)

// TestTransactionManager tests the MongoDB transaction manager
func TestTransactionManager(t *testing.T) {
	// Skip if short flag is set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create logger
	logger := zaptest.NewLogger(t)

	// Create MongoDB client
	client, err := createTestMongoClient(t)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer func() {
		err := client.Disconnect(context.Background())
		if err != nil {
			t.Logf("Failed to disconnect MongoDB client: %v", err)
		}
	}()

	// Create transaction manager
	txManager := mongodb.NewTransactionManager(client, logger)
	require.NotNil(t, txManager)

	// Test BeginTx
	t.Run("BeginTx", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txManager.BeginTx(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, txCtx)
		assert.NotEqual(t, ctx, txCtx)

		// Verify session is in context
		session := mongodb.GetSession(txCtx)
		assert.NotNil(t, session)

		// Clean up
		err = txManager.RollbackTx(txCtx)
		assert.NoError(t, err)
	})

	// Test CommitTx
	t.Run("CommitTx", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txManager.BeginTx(ctx)
		assert.NoError(t, err)

		// Commit transaction
		err = txManager.CommitTx(txCtx)
		assert.NoError(t, err)

		// Note: The session is still in the context, but it's been ended
		// We can't easily test this without exposing internal state
	})

	// Test RollbackTx
	t.Run("RollbackTx", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txManager.BeginTx(ctx)
		assert.NoError(t, err)

		// Rollback transaction
		err = txManager.RollbackTx(txCtx)
		assert.NoError(t, err)

		// Note: The session is still in the context, but it's been ended
		// We can't easily test this without exposing internal state
	})

	// Test CommitTx with no transaction
	t.Run("CommitTx_NoTransaction", func(t *testing.T) {
		ctx := context.Background()
		err := txManager.CommitTx(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no session found in context")
	})

	// Test RollbackTx with no transaction
	t.Run("RollbackTx_NoTransaction", func(t *testing.T) {
		ctx := context.Background()
		err := txManager.RollbackTx(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no session found in context")
	})

	// Test WithTx
	t.Run("WithTx", func(t *testing.T) {
		// Skip this test if we're not using a replica set
		// Transactions require a replica set in MongoDB
		t.Skip("Skipping WithTx test as it requires a MongoDB replica set")

		ctx := context.Background()

		// Test successful transaction
		err := txManager.WithTx(ctx, func(txCtx context.Context) error {
			// Verify session is in context
			session := mongodb.GetSession(txCtx)
			assert.NotNil(t, session)
			return nil
		})
		assert.NoError(t, err)

		// Test transaction with error
		err = txManager.WithTx(ctx, func(txCtx context.Context) error {
			return assert.AnError
		})
		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})

	// Test MustGetSession
	t.Run("MustGetSession", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txManager.BeginTx(ctx)
		assert.NoError(t, err)

		// Get session
		session, err := mongodb.MustGetSession(txCtx)
		assert.NoError(t, err)
		assert.NotNil(t, session)

		// Clean up
		err = txManager.RollbackTx(txCtx)
		assert.NoError(t, err)

		// Try to get session from context without session
		session, err = mongodb.MustGetSession(ctx)
		assert.Error(t, err)
		assert.Nil(t, session)
		assert.Contains(t, err.Error(), "no session found in context")
	})

	// Test BeginTx with existing session
	t.Run("BeginTx_ExistingSession", func(t *testing.T) {
		// Skip this test if we're not using a replica set
		// Transactions require a replica set in MongoDB
		t.Skip("Skipping BeginTx_ExistingSession test as it requires a MongoDB replica set")

		ctx := context.Background()
		txCtx, err := txManager.BeginTx(ctx)
		assert.NoError(t, err)

		// Try to begin another transaction with the same context
		txCtx2, err := txManager.BeginTx(txCtx)
		assert.NoError(t, err)
		assert.Equal(t, txCtx, txCtx2)

		// Clean up
		err = txManager.RollbackTx(txCtx)
		assert.NoError(t, err)
	})
}

// Helper function to create a test MongoDB client
func createTestMongoClient(t *testing.T) (*mongo.Client, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use environment variable or default to localhost
	mongoURI := "mongodb://localhost:27017/family_service_test"

	// Create client options
	clientOptions := options.Client().ApplyURI(mongoURI)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}
