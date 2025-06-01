package mocks

import (
	"context"
)

// MockAuthorizationService is a mock implementation of the ports.AuthorizationService interface
type MockAuthorizationService struct {
	// Function mocks for testing specific scenarios
	IsAuthorizedFunc func(ctx context.Context, operation string) (bool, error)
	IsAdminFunc      func(ctx context.Context) (bool, error)
	GetUserIDFunc    func(ctx context.Context) (string, error)
	GetUserRolesFunc func(ctx context.Context) ([]string, error)

	// Default return values
	DefaultIsAuthorized bool
	DefaultIsAdmin      bool
	DefaultUserID       string
	DefaultUserRoles    []string

	// Call tracking for assertions
	IsAuthorizedCalls  []string
	IsAdminCalled      bool
	GetUserIDCalled    bool
	GetUserRolesCalled bool
}

// NewMockAuthorizationService creates a new mock authorization service
func NewMockAuthorizationService() *MockAuthorizationService {
	return &MockAuthorizationService{
		DefaultIsAuthorized: true,
		DefaultIsAdmin:      false,
		DefaultUserID:       "test-user-id",
		DefaultUserRoles:    []string{"user"},
		IsAuthorizedCalls:   make([]string, 0),
	}
}

// IsAuthorized checks if the user is authorized to perform the operation
func (s *MockAuthorizationService) IsAuthorized(ctx context.Context, operation string) (bool, error) {
	// Track the call
	s.IsAuthorizedCalls = append(s.IsAuthorizedCalls, operation)

	if s.IsAuthorizedFunc != nil {
		return s.IsAuthorizedFunc(ctx, operation)
	}

	return s.DefaultIsAuthorized, nil
}

// IsAdmin checks if the user has admin role
func (s *MockAuthorizationService) IsAdmin(ctx context.Context) (bool, error) {
	s.IsAdminCalled = true

	if s.IsAdminFunc != nil {
		return s.IsAdminFunc(ctx)
	}

	return s.DefaultIsAdmin, nil
}

// GetUserID retrieves the user ID from the context
func (s *MockAuthorizationService) GetUserID(ctx context.Context) (string, error) {
	s.GetUserIDCalled = true

	if s.GetUserIDFunc != nil {
		return s.GetUserIDFunc(ctx)
	}

	return s.DefaultUserID, nil
}

// GetUserRoles retrieves the user roles from the context
func (s *MockAuthorizationService) GetUserRoles(ctx context.Context) ([]string, error) {
	s.GetUserRolesCalled = true

	if s.GetUserRolesFunc != nil {
		return s.GetUserRolesFunc(ctx)
	}

	return s.DefaultUserRoles, nil
}

// Reset resets the state of the mock authorization service
func (s *MockAuthorizationService) Reset() {
	s.DefaultIsAuthorized = true
	s.DefaultIsAdmin = false
	s.DefaultUserID = "test-user-id"
	s.DefaultUserRoles = []string{"user"}
	s.IsAuthorizedCalls = make([]string, 0)
	s.IsAdminCalled = false
	s.GetUserIDCalled = false
	s.GetUserRolesCalled = false
}
