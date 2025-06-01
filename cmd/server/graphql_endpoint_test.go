package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/abitofhelp/family-service2/internal/adapters/graphql"
	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/abitofhelp/family-service2/internal/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupGraphQLTest sets up a test server with the GraphQL endpoint
func setupGraphQLTest(t *testing.T) (*httptest.Server, *mocks.MockFamilyService, *mocks.MockAuthorizationService) {
	// Create a logger
	logger := zaptest.NewLogger(t)

	// Create mock services
	mockFamilyService := mocks.NewMockFamilyService()
	mockAuthService := mocks.NewMockAuthorizationService()

	// Create a resolver
	resolver := graphql.NewResolver(mockFamilyService, mockAuthService, logger)

	// Create a GraphQL server
	gqlServer := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: resolver,
	}))

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gqlServer.ServeHTTP(w, r)
	}))

	return server, mockFamilyService, mockAuthService
}

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// executeGraphQLRequest executes a GraphQL request and returns the response
func executeGraphQLRequest(t *testing.T, server *httptest.Server, request GraphQLRequest) map[string]interface{} {
	// Marshal the request to JSON
	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	// Create an HTTP request
	req, err := http.NewRequest("POST", server.URL, bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check the status code
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse the response
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	return response
}

// TestGraphQLParentQuery tests the parent query
func TestGraphQLParentQuery(t *testing.T) {
	// Set up the test server
	server, mockFamilyService, mockAuthService := setupGraphQLTest(t)
	defer server.Close()

	// Create a test parent
	parentID := uuid.New()
	birthDate, _ := time.Parse(time.RFC3339, "1980-01-01T00:00:00Z")
	createdAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()

	parent := &domain.Parent{
		ID:        parentID,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		BirthDate: birthDate,
		Children:  []domain.Child{},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	// Set up mock expectations
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, operation string) (bool, error) {
		assert.Equal(t, "parent:read", operation)
		return true, nil
	}
	
	mockFamilyService.GetParentByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
		assert.Equal(t, parentID, id)
		return parent, nil
	}

	// Create a GraphQL query
	query := `
		query GetParent($id: ID!) {
			parent(id: $id) {
				id
				firstName
				lastName
				email
				birthDate
				createdAt
				updatedAt
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
	errors, hasErrors := response["errors"]
	if hasErrors {
		t.Fatalf("GraphQL query returned errors: %v", errors)
	}

	// Extract the data
	data, hasData := response["data"].(map[string]interface{})
	require.True(t, hasData, "Response should contain data")

	// Extract the parent
	parentData, hasParent := data["parent"].(map[string]interface{})
	require.True(t, hasParent, "Data should contain parent")

	// Verify the parent data
	assert.Equal(t, parentID.String(), parentData["id"])
	assert.Equal(t, "John", parentData["firstName"])
	assert.Equal(t, "Doe", parentData["lastName"])
	assert.Equal(t, "john.doe@example.com", parentData["email"])
	assert.Equal(t, birthDate.Format(time.RFC3339), parentData["birthDate"])
	assert.NotEmpty(t, parentData["createdAt"])
	assert.NotEmpty(t, parentData["updatedAt"])
}

// TestGraphQLCreateParentMutation tests the createParent mutation
func TestGraphQLCreateParentMutation(t *testing.T) {
	// Set up the test server
	server, mockFamilyService, mockAuthService := setupGraphQLTest(t)
	defer server.Close()

	// Create a test parent
	parentID := uuid.New()
	birthDate, _ := time.Parse(time.RFC3339, "1980-01-01T00:00:00Z")
	createdAt := time.Now()
	updatedAt := time.Now()

	parent := &domain.Parent{
		ID:        parentID,
		FirstName: "Jane",
		LastName:  "Smith",
		Email:     "jane.smith@example.com",
		BirthDate: birthDate,
		Children:  []domain.Child{},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	// Set up mock expectations
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, operation string) (bool, error) {
		assert.Equal(t, "parent:create", operation)
		return true, nil
	}
	
	mockFamilyService.CreateParentFunc = func(ctx context.Context, firstName, lastName, email, birthDate string) (*domain.Parent, error) {
		assert.Equal(t, "Jane", firstName)
		assert.Equal(t, "Smith", lastName)
		assert.Equal(t, "jane.smith@example.com", email)
		assert.Equal(t, "1980-01-01T00:00:00Z", birthDate)
		return parent, nil
	}

	// Create a GraphQL mutation
	mutation := `
		mutation CreateParent($input: CreateParentInput!) {
			createParent(input: $input) {
				id
				firstName
				lastName
				email
				birthDate
				createdAt
				updatedAt
			}
		}
	`

	// Execute the mutation
	response := executeGraphQLRequest(t, server, GraphQLRequest{
		Query: mutation,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
				"firstName": "Jane",
				"lastName":  "Smith",
				"email":     "jane.smith@example.com",
				"birthDate": "1980-01-01T00:00:00Z",
			},
		},
	})

	// Check for errors
	errors, hasErrors := response["errors"]
	if hasErrors {
		t.Fatalf("GraphQL mutation returned errors: %v", errors)
	}

	// Extract the data
	data, hasData := response["data"].(map[string]interface{})
	require.True(t, hasData, "Response should contain data")

	// Extract the parent
	parentData, hasParent := data["createParent"].(map[string]interface{})
	require.True(t, hasParent, "Data should contain createParent")

	// Verify the parent data
	assert.Equal(t, parentID.String(), parentData["id"])
	assert.Equal(t, "Jane", parentData["firstName"])
	assert.Equal(t, "Smith", parentData["lastName"])
	assert.Equal(t, "jane.smith@example.com", parentData["email"])
	assert.Equal(t, birthDate.Format(time.RFC3339), parentData["birthDate"])
	assert.NotEmpty(t, parentData["createdAt"])
	assert.NotEmpty(t, parentData["updatedAt"])
}

// TestGraphQLChildQuery tests the child query
func TestGraphQLChildQuery(t *testing.T) {
	// Set up the test server
	server, mockFamilyService, mockAuthService := setupGraphQLTest(t)
	defer server.Close()

	// Create a test child
	childID := uuid.New()
	parentID := uuid.New()
	birthDate, _ := time.Parse(time.RFC3339, "2010-01-01T00:00:00Z")
	createdAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()

	child := &domain.Child{
		ID:        childID,
		FirstName: "Alice",
		LastName:  "Doe",
		BirthDate: birthDate,
		ParentID:  parentID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	// Set up mock expectations
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, operation string) (bool, error) {
		assert.Equal(t, "child:read", operation)
		return true, nil
	}
	
	mockFamilyService.GetChildByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
		assert.Equal(t, childID, id)
		return child, nil
	}

	// Create a GraphQL query
	query := `
		query GetChild($id: ID!) {
			child(id: $id) {
				id
				firstName
				lastName
				birthDate
				parentId
				createdAt
				updatedAt
			}
		}
	`

	// Execute the query
	response := executeGraphQLRequest(t, server, GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": childID.String(),
		},
	})

	// Check for errors
	errors, hasErrors := response["errors"]
	if hasErrors {
		t.Fatalf("GraphQL query returned errors: %v", errors)
	}

	// Extract the data
	data, hasData := response["data"].(map[string]interface{})
	require.True(t, hasData, "Response should contain data")

	// Extract the child
	childData, hasChild := data["child"].(map[string]interface{})
	require.True(t, hasChild, "Data should contain child")

	// Verify the child data
	assert.Equal(t, childID.String(), childData["id"])
	assert.Equal(t, "Alice", childData["firstName"])
	assert.Equal(t, "Doe", childData["lastName"])
	assert.Equal(t, birthDate.Format(time.RFC3339), childData["birthDate"])
	assert.Equal(t, parentID.String(), childData["parentId"])
	assert.NotEmpty(t, childData["createdAt"])
	assert.NotEmpty(t, childData["updatedAt"])
}

// TestGraphQLCreateChildMutation tests the createChild mutation
func TestGraphQLCreateChildMutation(t *testing.T) {
	// Set up the test server
	server, mockFamilyService, mockAuthService := setupGraphQLTest(t)
	defer server.Close()

	// Create a test child
	childID := uuid.New()
	parentID := uuid.New()
	birthDate, _ := time.Parse(time.RFC3339, "2010-01-01T00:00:00Z")
	createdAt := time.Now()
	updatedAt := time.Now()

	child := &domain.Child{
		ID:        childID,
		FirstName: "Bob",
		LastName:  "Smith",
		BirthDate: birthDate,
		ParentID:  parentID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	// Set up mock expectations
	mockAuthService.IsAuthorizedFunc = func(ctx context.Context, operation string) (bool, error) {
		assert.Equal(t, "child:create", operation)
		return true, nil
	}
	
	mockFamilyService.CreateChildFunc = func(ctx context.Context, firstName, lastName string, birthDate string, parentID uuid.UUID) (*domain.Child, error) {
		assert.Equal(t, "Bob", firstName)
		assert.Equal(t, "Smith", lastName)
		assert.Equal(t, "2010-01-01T00:00:00Z", birthDate)
		return child, nil
	}

	// Create a GraphQL mutation
	mutation := `
		mutation CreateChild($input: CreateChildInput!) {
			createChild(input: $input) {
				id
				firstName
				lastName
				birthDate
				parentId
				createdAt
				updatedAt
			}
		}
	`

	// Execute the mutation
	response := executeGraphQLRequest(t, server, GraphQLRequest{
		Query: mutation,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
				"firstName": "Bob",
				"lastName":  "Smith",
				"birthDate": "2010-01-01T00:00:00Z",
				"parentId":  parentID.String(),
			},
		},
	})

	// Check for errors
	errors, hasErrors := response["errors"]
	if hasErrors {
		t.Fatalf("GraphQL mutation returned errors: %v", errors)
	}

	// Extract the data
	data, hasData := response["data"].(map[string]interface{})
	require.True(t, hasData, "Response should contain data")

	// Extract the child
	childData, hasChild := data["createChild"].(map[string]interface{})
	require.True(t, hasChild, "Data should contain createChild")

	// Verify the child data
	assert.Equal(t, childID.String(), childData["id"])
	assert.Equal(t, "Bob", childData["firstName"])
	assert.Equal(t, "Smith", childData["lastName"])
	assert.Equal(t, birthDate.Format(time.RFC3339), childData["birthDate"])
	assert.Equal(t, parentID.String(), childData["parentId"])
	assert.NotEmpty(t, childData["createdAt"])
	assert.NotEmpty(t, childData["updatedAt"])
}