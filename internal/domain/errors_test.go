package domain_test

import (
	"errors"
	"testing"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNotFoundError(t *testing.T) {
	// Test constructor
	err := domain.NewNotFoundError("Parent", "123")
	assert.NotNil(t, err)
	assert.Equal(t, "Parent", err.EntityType)
	assert.Equal(t, "123", err.ID)
	assert.Equal(t, domain.ErrNotFound, err.Err)

	// Test Error method
	assert.Equal(t, "Parent with ID 123 not found", err.Error())

	// Test Unwrap method
	assert.Equal(t, domain.ErrNotFound, errors.Unwrap(err))

	// Test Is method
	assert.True(t, errors.Is(err, domain.ErrNotFound))
	assert.False(t, errors.Is(err, domain.ErrValidation))
}

func TestValidationError(t *testing.T) {
	// Test constructor with field
	err := domain.NewValidationError("Parent", "firstName", "is required")
	assert.NotNil(t, err)
	assert.Equal(t, "Parent", err.EntityType)
	assert.Equal(t, "firstName", err.Field)
	assert.Equal(t, "is required", err.Reason)
	assert.Equal(t, domain.ErrValidation, err.Err)

	// Test Error method with field
	assert.Equal(t, "validation failed for Parent: field first name is required", err.Error())

	// Test constructor without field
	err = domain.NewValidationError("Parent", "", "invalid data")
	assert.NotNil(t, err)
	assert.Equal(t, "Parent", err.EntityType)
	assert.Equal(t, "", err.Field)
	assert.Equal(t, "invalid data", err.Reason)

	// Test Error method without field
	assert.Equal(t, "validation failed for Parent: invalid data", err.Error())

	// Test Unwrap method
	assert.Equal(t, domain.ErrValidation, errors.Unwrap(err))

	// Test Is method
	assert.True(t, errors.Is(err, domain.ErrValidation))
	assert.False(t, errors.Is(err, domain.ErrNotFound))

	// Test camelCase field name formatting
	err = domain.NewValidationError("Child", "birthDate", "is invalid")
	assert.Equal(t, "validation failed for Child: field birth date is invalid", err.Error())
}

func TestTransactionError(t *testing.T) {
	// Create a test error
	testErr := errors.New("test error")

	// Test constructor
	err := domain.NewTransactionError("begin", testErr)
	assert.NotNil(t, err)
	assert.Equal(t, "begin", err.Operation)
	assert.Equal(t, testErr, err.Err)

	// Test Error method
	assert.Equal(t, "transaction begin failed: test error", err.Error())

	// Test Unwrap method
	assert.Equal(t, testErr, errors.Unwrap(err))
}

func TestDatabaseError(t *testing.T) {
	// Create a test error
	testErr := errors.New("test error")

	// Test constructor
	err := domain.NewDatabaseError("create", "Parent", testErr)
	assert.NotNil(t, err)
	assert.Equal(t, "create", err.Operation)
	assert.Equal(t, "Parent", err.EntityType)
	assert.Equal(t, testErr, err.Err)

	// Test Error method
	assert.Equal(t, "database operation create on Parent failed: test error", err.Error())

	// Test Unwrap method
	assert.Equal(t, testErr, errors.Unwrap(err))
}

func TestCommonErrorTypes(t *testing.T) {
	// Test that all common error types are defined
	assert.NotNil(t, domain.ErrNotFound)
	assert.NotNil(t, domain.ErrValidation)
	assert.NotNil(t, domain.ErrDuplicate)
	assert.NotNil(t, domain.ErrInvalidInput)
	assert.NotNil(t, domain.ErrUnauthorized)
	assert.NotNil(t, domain.ErrForbidden)
	assert.NotNil(t, domain.ErrInternal)

	// Test error messages
	assert.Equal(t, "entity not found", domain.ErrNotFound.Error())
	assert.Equal(t, "validation failed", domain.ErrValidation.Error())
	assert.Equal(t, "entity already exists", domain.ErrDuplicate.Error())
	assert.Equal(t, "invalid input", domain.ErrInvalidInput.Error())
	assert.Equal(t, "unauthorized", domain.ErrUnauthorized.Error())
	assert.Equal(t, "forbidden", domain.ErrForbidden.Error())
	assert.Equal(t, "internal error", domain.ErrInternal.Error())
}