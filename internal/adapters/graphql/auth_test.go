package graphql_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abitofhelp/family-service2/internal/adapters/graphql"
	"github.com/abitofhelp/family-service2/internal/infrastructure/auth"
	"github.com/abitofhelp/family-service2/internal/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestNewAuthMiddleware(t *testing.T) {
	// Setup
	mockAuthService := mocks.NewMockAuthorizationService()
	logger := zaptest.NewLogger(t)
	jwtConfig := auth.JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: 1 * time.Hour,
		Issuer:        "test-issuer",
	}
	jwtService := auth.NewJWTService(jwtConfig, logger)

	// Execute
	middleware := graphql.NewAuthMiddleware(mockAuthService, jwtService, logger)

	// Assert
	assert.NotNil(t, middleware)
}

func TestAuthMiddleware_Middleware(t *testing.T) {
	// Setup
	mockAuthService := mocks.NewMockAuthorizationService()
	logger := zaptest.NewLogger(t)
	jwtConfig := auth.JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: 1 * time.Hour,
		Issuer:        "test-issuer",
	}
	jwtService := auth.NewJWTService(jwtConfig, logger)
	middleware := graphql.NewAuthMiddleware(mockAuthService, jwtService, logger)

	// Create a test handler that will be wrapped by the middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	})

	// Create a test request
	req := httptest.NewRequest("GET", "/graphql", nil)
	rec := httptest.NewRecorder()

	// Execute
	handler := middleware.Middleware(testHandler)
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "test response")
}
