package main

import (
	"context"
	"encoding/json"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/infrastructure/config"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/infrastructure/health"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/infrastructure/logging"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/infrastructure/server"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/infrastructure/shutdown"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/mocks"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockContainer is a mock implementation of the Container for testing
type MockContainer struct {
	mock.Mock
	repoFactory interface{}
	logger      *zap.Logger
}

func NewMockContainer(repoFactory interface{}, logger *zap.Logger) *MockContainer {
	return &MockContainer{
		repoFactory: repoFactory,
		logger:      logger,
	}
}

func (m *MockContainer) GetRepositoryFactory() any {
	return m.repoFactory
}

func (m *MockContainer) GetLogger() *zap.Logger {
	return m.logger
}

func (m *MockContainer) Close() error {
	args := m.Called()
	return args.Error(0)
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	// Set up koanf with test configuration
	k := koanf.New(".")
	k.Set("app.version", "1.0.0")

	// Create a mock config with the version from koanf
	mockConfig := &config.Config{
		App: config.AppConfig{
			Version: k.String("app.version"),
		},
	}

	// Verify constants are as expected
	assert.Equal(t, "Healthy", health.StatusHealthy)
	assert.Equal(t, "Up", health.ServiceUp)

	// Create a mock repository factory
	mockRepoFactory := mocks.NewMockRepositoryFactory()

	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a mock container
	mockContainer := NewMockContainer(mockRepoFactory, logger)
	mockContainer.On("Close").Return(nil)

	// Create a request to the health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Get the handler function from health.NewHandler
	handler := health.NewHandler(mockContainer, logger, mockConfig)

	// Call the handler function
	handler(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify the response fields
	assert.Equal(t, "Healthy", response["status"])
	assert.NotEmpty(t, response["timestamp"])
	assert.Equal(t, "1.0.0", response["version"])

	// Verify services
	services, ok := response["services"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Up", services["database"])
}

// TestHealthCheckDegraded tests the health check endpoint when database is down
func TestHealthCheckDegraded(t *testing.T) {
	// Set up koanf with test configuration
	k := koanf.New(".")
	k.Set("app.version", "1.0.0")

	// Create a mock config with the version from koanf
	mockConfig := &config.Config{
		App: config.AppConfig{
			Version: k.String("app.version"),
		},
	}

	// Verify constants are as expected
	assert.Equal(t, "Degraded", health.StatusDegraded)
	assert.Equal(t, "Down", health.ServiceDown)

	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a mock container with nil repository factory
	mockContainer := NewMockContainer(nil, logger)
	mockContainer.On("Close").Return(nil)

	// Create a request to the health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Get the handler function from health.NewHandler
	handler := health.NewHandler(mockContainer, logger, mockConfig)

	// Call the handler function
	handler(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

	// Check the response body
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify the response fields
	assert.Equal(t, "Degraded", response["status"])
	assert.NotEmpty(t, response["timestamp"])
	assert.Equal(t, "1.0.0", response["version"])

	// Verify services
	services, ok := response["services"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Down", services["database"])
}

// TestServerConfig tests the server configuration
func TestServerConfig(t *testing.T) {
	// Set up environment variables for configuration
	// Set APP_ENV to empty to avoid loading environment files
	os.Setenv("APP_ENV", "")
	// Set required environment variables for database connection strings
	os.Setenv("MONGODB_ROOT_PASSWORD", "testpassword")
	os.Setenv("POSTGRESQL_POSTGRES_PASSWORD", "testpassword")
	// Don't set server configuration via environment variables
	// Instead, rely on the default values defined in getDefaultsMap()

	defer func() {
		// No environment variables to unset
	}()

	// Initialize configuration
	cfg, err := config.LoadConfig()
	assert.Nil(t, err, "Failed initialize application configuration: %v", err)

	// Print the actual values for debugging
	t.Logf("Original ReadTimeout: %v", cfg.Server.ReadTimeout)
	t.Logf("Original WriteTimeout: %v", cfg.Server.WriteTimeout)
	t.Logf("Original IdleTimeout: %v", cfg.Server.IdleTimeout)
	t.Logf("Original ShutdownTimeout: %v", cfg.Server.ShutdownTimeout)

	// No need to convert, the values are already in the correct unit (time.Duration)

	// Print the converted values for debugging
	t.Logf("Converted ReadTimeout: %v", cfg.Server.ReadTimeout)
	t.Logf("Converted WriteTimeout: %v", cfg.Server.WriteTimeout)
	t.Logf("Converted IdleTimeout: %v", cfg.Server.IdleTimeout)
	t.Logf("Converted ShutdownTimeout: %v", cfg.Server.ShutdownTimeout)

	// Verify the configuration
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, 10*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 120*time.Second, cfg.Server.IdleTimeout)
	assert.Equal(t, 10*time.Second, cfg.Server.ShutdownTimeout)
}

// TestServerStartAndShutdown tests starting and shutting down the server
func TestServerStartAndShutdown(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a server configuration
	cfg := server.Config{
		Port:            "0", // Use any available port
		ReadTimeout:     1 * time.Second,
		WriteTimeout:    1 * time.Second,
		IdleTimeout:     1 * time.Second,
		ShutdownTimeout: 1 * time.Second,
	}

	// Create a context logger
	contextLogger := logging.NewContextLogger(logger)

	// Create a server
	srv := server.New(cfg, http.NewServeMux(), logger, contextLogger)

	// Start the server
	srv.Start()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown the server
	err := srv.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestGracefulShutdown tests the graceful shutdown functionality
func TestGracefulShutdown(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to track if the shutdown function was called
	shutdownCalled := make(chan struct{})

	// Create a shutdown function
	shutdownFunc := func() error {
		close(shutdownCalled)
		return nil
	}

	// Start the graceful shutdown in a goroutine
	go func() {
		// Wait a moment before cancelling the context
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Call GracefulShutdown
	shutdown.GracefulShutdown(ctx, logger, shutdownFunc)

	// Verify that the shutdown function was called
	select {
	case <-shutdownCalled:
		// Shutdown function was called
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for shutdown function to be called")
	}
}

// Note: The containerAdapter is a simple wrapper around the di.Container
// and doesn't need extensive testing. Its functionality is indirectly
// tested through the health check tests.
