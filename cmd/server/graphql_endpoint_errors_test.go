package main

import (
	"context"
	"errors"
	"testing"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGraphQLParentQuery_AuthError tests the parent query with an authorization error
func TestGraphQLParentQuery_AuthError(t *testing.T) {
	// Set up the test server
	server, _, mockAuthService := setupGraphQLTest(t)
	defer server.Close()

	// Set up mock expectations
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, operation string) (bool, error) {
		assert.Equal(t, "parent:read", operation)
		return false, errors.New("authorization error")
	}

	// Create a GraphQL query
	query := `
		query GetParent($id: ID!) {
			parent(id: $id) {
				id
				firstName
				lastName
				email
			}
		}
	`

	// Execute the query
	response := executeGraphQLRequest(t, server, GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": uuid.New().String(),
		},
	})

	// Check for errors
	errors, hasErrors := response["errors"].([]interface{})
	require.True(t, hasErrors, "GraphQL query should return errors")
	require.NotEmpty(t, errors, "Errors should not be empty")

	// Verify the error message
	errorObj := errors[0].(map[string]interface{})
	errorMessage := errorObj["message"].(string)
	assert.Contains(t, errorMessage, "failed to check authorization")
}

// TestGraphQLParentQuery_Unauthorized tests the parent query with unauthorized access
func TestGraphQLParentQuery_Unauthorized(t *testing.T) {
	// Set up the test server
	server, _, mockAuthService := setupGraphQLTest(t)
	defer server.Close()

	// Set up mock expectations
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, operation string) (bool, error) {
		assert.Equal(t, "parent:read", operation)
		return false, nil
	}

	// Create a GraphQL query
	query := `
		query GetParent($id: ID!) {
			parent(id: $id) {
				id
				firstName
				lastName
				email
			}
		}
	`

	// Execute the query
	response := executeGraphQLRequest(t, server, GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": uuid.New().String(),
		},
	})

	// Check for errors
	errors, hasErrors := response["errors"].([]interface{})
	require.True(t, hasErrors, "GraphQL query should return errors")
	require.NotEmpty(t, errors, "Errors should not be empty")

	// Verify the error message
	errorObj := errors[0].(map[string]interface{})
	errorMessage := errorObj["message"].(string)
	assert.Contains(t, errorMessage, "not authorized to read parent")
}

// TestGraphQLParentQuery_InvalidID tests the parent query with an invalid ID
func TestGraphQLParentQuery_InvalidID(t *testing.T) {
	// Set up the test server
	server, _, mockAuthService := setupGraphQLTest(t)
	defer server.Close()

	// Set up mock expectations
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, operation string) (bool, error) {
		assert.Equal(t, "parent:read", operation)
		return true, nil
	}

	// Create a GraphQL query
	query := `
		query GetParent($id: ID!) {
			parent(id: $id) {
				id
				firstName
				lastName
				email
			}
		}
	`

	// Execute the query with an invalid ID
	response := executeGraphQLRequest(t, server, GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": "invalid-uuid",
		},
	})

	// Check for errors
	errors, hasErrors := response["errors"].([]interface{})
	require.True(t, hasErrors, "GraphQL query should return errors")
	require.NotEmpty(t, errors, "Errors should not be empty")

	// Verify the error message
	errorObj := errors[0].(map[string]interface{})
	errorMessage := errorObj["message"].(string)
	assert.Contains(t, errorMessage, "invalid parent ID")
}

// TestGraphQLParentQuery_NotFound tests the parent query with a non-existent parent
func TestGraphQLParentQuery_NotFound(t *testing.T) {
	// Set up the test server
	server, mockFamilyService, mockAuthService := setupGraphQLTest(t)
	defer server.Close()

	// Create a test parent ID
	parentID := uuid.New()

	// Set up mock expectations
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, operation string) (bool, error) {
		assert.Equal(t, "parent:read", operation)
		return true, nil
	}

	mockFamilyService.GetParentByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
		assert.Equal(t, parentID, id)
		return nil, errors.New("parent not found")
	}

	// Create a GraphQL query
	query := `
		query GetParent($id: ID!) {
			parent(id: $id) {
				id
				firstName
				lastName
				email
			}
		}
	`

	// Execute the query
	response := executeGraphQLRequest(t, server, GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": parentID.String(),
		},
	})

	// Check for errors
	errors, hasErrors := response["errors"].([]interface{})
	require.True(t, hasErrors, "GraphQL query should return errors")
	require.NotEmpty(t, errors, "Errors should not be empty")

	// Verify the error message
	errorObj := errors[0].(map[string]interface{})
	errorMessage := errorObj["message"].(string)
	assert.Contains(t, errorMessage, "failed to get parent")
}

// TestGraphQLCreateParentMutation_ValidationError tests the createParent mutation with validation errors
func TestGraphQLCreateParentMutation_ValidationError(t *testing.T) {
	// Set up the test server
	server, mockFamilyService, mockAuthService := setupGraphQLTest(t)
	defer server.Close()

	// Set up mock expectations
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, operation string) (bool, error) {
		assert.Equal(t, "parent:create", operation)
		return true, nil
	}

	mockFamilyService.CreateParentFunc = func(ctx context.Context, firstName, lastName, email, birthDate string) (*domain.Parent, error) {
		return nil, errors.New("validation error: invalid email format")
	}

	// Create a GraphQL mutation
	mutation := `
		mutation CreateParent($input: CreateParentInput!) {
			createParent(input: $input) {
				id
				firstName
				lastName
				email
			}
		}
	`

	// Execute the mutation with invalid input
	response := executeGraphQLRequest(t, server, GraphQLRequest{
		Query: mutation,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
				"firstName": "Jane",
				"lastName":  "Smith",
				"email":     "invalid-email",
				"birthDate": "1980-01-01T00:00:00Z",
			},
		},
	})

	// Check for errors
	errors, hasErrors := response["errors"].([]interface{})
	require.True(t, hasErrors, "GraphQL mutation should return errors")
	require.NotEmpty(t, errors, "Errors should not be empty")

	// Verify the error message
	errorObj := errors[0].(map[string]interface{})
	errorMessage := errorObj["message"].(string)
	assert.Contains(t, errorMessage, "failed to create parent")
	assert.Contains(t, errorMessage, "validation error")
}
