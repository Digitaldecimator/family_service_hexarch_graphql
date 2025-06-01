package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/adapters/mongodb"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap/zaptest"
)

// TestMongoDBConfig implements the ports.MongoDBConfig interface for testing
type TestMongoDBConfig struct {
	uri               string
	connectionTimeout time.Duration
	pingTimeout       time.Duration
	disconnectTimeout time.Duration
	indexTimeout      time.Duration
}

// Ensure TestMongoDBConfig implements ports.MongoDBConfig
var _ ports.MongoDBConfig = (*TestMongoDBConfig)(nil)

// GetConnectionTimeout returns the connection timeout
func (c *TestMongoDBConfig) GetConnectionTimeout() time.Duration {
	return c.connectionTimeout
}

// GetPingTimeout returns the ping timeout
func (c *TestMongoDBConfig) GetPingTimeout() time.Duration {
	return c.pingTimeout
}

// GetDisconnectTimeout returns the disconnect timeout
func (c *TestMongoDBConfig) GetDisconnectTimeout() time.Duration {
	return c.disconnectTimeout
}

// GetIndexTimeout returns the index timeout
func (c *TestMongoDBConfig) GetIndexTimeout() time.Duration {
	return c.indexTimeout
}

// GetURI returns the MongoDB URI
func (c *TestMongoDBConfig) GetURI() string {
	return c.uri
}

// createTestMongoDBConfig creates a test MongoDB config
func createTestMongoDBConfig(uri string) *TestMongoDBConfig {
	return &TestMongoDBConfig{
		uri:               uri,
		connectionTimeout: 10 * time.Second,
		pingTimeout:       5 * time.Second,
		disconnectTimeout: 5 * time.Second,
		indexTimeout:      10 * time.Second,
	}
}

// No longer needed as we're using TestMongoDBConfig directly

// TestNewRepositoryFactory tests the creation of a new repository factory
func TestNewRepositoryFactory(t *testing.T) {
	// Skip if short flag is set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create logger
	logger := zaptest.NewLogger(t)

	// Test cases
	tests := []struct {
		name        string
		uri         string
		expectError bool
	}{
		{
			name:        "Valid connection string",
			uri:         getTestMongoURI(t),
			expectError: false,
		},
		{
			name:        "Invalid connection string",
			uri:         "mongodb://invalid:27017",
			expectError: true,
		},
		{
			name:        "Empty connection string",
			uri:         "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create test config
			testConfig := createTestMongoDBConfig(tc.uri)

			// Create repository factory
			factory, err := mongodb.NewRepositoryFactory(ctx, logger, testConfig)

			// Check error
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, factory)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, factory)

				// Test repository creation
				assert.NotNil(t, factory.NewParentRepository())
				assert.NotNil(t, factory.NewChildRepository())
				assert.NotNil(t, factory.GetTransactionManager())

				// Test close
				err = factory.Close(ctx, testConfig)
				assert.NoError(t, err)
			}
		})
	}
}

// TestRepositoryFactory_InitSchema tests the initialization of the database schema
func TestRepositoryFactory_InitSchema(t *testing.T) {
	// Skip if short flag is set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create logger
	logger := zaptest.NewLogger(t)

	// Create test config
	testConfig := createTestMongoDBConfig(getTestMongoURI(t))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create repository factory
	factory, err := mongodb.NewRepositoryFactory(ctx, logger, testConfig)
	require.NoError(t, err)
	require.NotNil(t, factory)

	// Test InitSchema
	err = factory.InitSchema(ctx)
	assert.NoError(t, err)

	// Clean up
	err = factory.Close(ctx, testConfig)
	assert.NoError(t, err)
}

// TestRepositoryFactory_Close tests closing the repository factory
func TestRepositoryFactory_Close(t *testing.T) {
	// Skip if short flag is set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create logger
	logger := zaptest.NewLogger(t)

	// Create test config
	testConfig := createTestMongoDBConfig(getTestMongoURI(t))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create repository factory
	factory, err := mongodb.NewRepositoryFactory(ctx, logger, testConfig)
	require.NoError(t, err)
	require.NotNil(t, factory)

	// Test Close
	err = factory.Close(ctx, testConfig)
	assert.NoError(t, err)

	// Note: We don't test Close with nil context here because the client is already disconnected
}

// TestRepositoryFactory_NilContext tests creating a repository factory with a nil context
func TestRepositoryFactory_NilContext(t *testing.T) {
	// Skip if short flag is set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create logger
	logger := zaptest.NewLogger(t)

	// Create test config
	testConfig := createTestMongoDBConfig(getTestMongoURI(t))

	// Create repository factory with nil context
	factory, err := mongodb.NewRepositoryFactory(nil, logger, testConfig)
	assert.NoError(t, err)
	assert.NotNil(t, factory)

	// Clean up
	err = factory.Close(context.Background(), testConfig)
	assert.NoError(t, err)
}

// Helper function to get a test MongoDB URI
func getTestMongoURI(t *testing.T) string {
	// Try to connect to the test MongoDB instance
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use environment variable or default to localhost
	mongoURI := "mongodb://localhost:27017/family_service_test"

	// Try to connect
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Logf("Could not connect to MongoDB at %s: %v", mongoURI, err)
		t.Logf("Falling back to mock behavior")
		return "mongodb://localhost:27017/family_service_test"
	}

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		t.Logf("Could not ping MongoDB at %s: %v", mongoURI, err)
		t.Logf("Falling back to mock behavior")
		return "mongodb://localhost:27017/family_service_test"
	}

	// Disconnect
	_ = client.Disconnect(ctx)

	return mongoURI
}
