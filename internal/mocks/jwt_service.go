package mocks

import (
	"github.com/abitofhelp/family-service2/internal/infrastructure/auth"
)

// MockJWTService is a mock implementation of the auth.JWTService
type MockJWTService struct {
	// Function mocks for testing specific scenarios
	GenerateTokenFunc func(userID string, roles []string) (string, error)
	ValidateTokenFunc func(tokenString string) (*auth.Claims, error)

	// Default return values
	DefaultToken  string
	DefaultClaims *auth.Claims

	// Call tracking for assertions
	GenerateTokenCalls []struct {
		UserID string
		Roles  []string
	}
	ValidateTokenCalls []string
}

// NewMockJWTService creates a new mock JWT service
func NewMockJWTService() *MockJWTService {
	return &MockJWTService{
		DefaultToken: "mock-token",
		DefaultClaims: &auth.Claims{
			UserID: "test-user-id",
			Roles:  []string{"user"},
		},
		GenerateTokenCalls: make([]struct {
			UserID string
			Roles  []string
		}, 0),
		ValidateTokenCalls: make([]string, 0),
	}
}

// GenerateToken generates a new JWT token for a user
func (s *MockJWTService) GenerateToken(userID string, roles []string) (string, error) {
	// Track the call
	s.GenerateTokenCalls = append(s.GenerateTokenCalls, struct {
		UserID string
		Roles  []string
	}{
		UserID: userID,
		Roles:  roles,
	})

	if s.GenerateTokenFunc != nil {
		return s.GenerateTokenFunc(userID, roles)
	}

	return s.DefaultToken, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *MockJWTService) ValidateToken(tokenString string) (*auth.Claims, error) {
	// Track the call
	s.ValidateTokenCalls = append(s.ValidateTokenCalls, tokenString)

	if s.ValidateTokenFunc != nil {
		return s.ValidateTokenFunc(tokenString)
	}

	return s.DefaultClaims, nil
}

// Reset resets the state of the mock JWT service
func (s *MockJWTService) Reset() {
	s.DefaultToken = "mock-token"
	s.DefaultClaims = &auth.Claims{
		UserID: "test-user-id",
		Roles:  []string{"user"},
	}
	s.GenerateTokenCalls = make([]struct {
		UserID string
		Roles  []string
	}, 0)
	s.ValidateTokenCalls = make([]string, 0)
}
