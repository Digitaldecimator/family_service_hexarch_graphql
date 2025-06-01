// Package domain defines the core business entities and rules for the family service.
// This file contains custom error types for domain-level errors.
package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Common error types that can be used for error checking
var (
	// ErrNotFound is returned when an entity is not found
	ErrNotFound = errors.New("entity not found")

	// ErrValidation is returned when entity validation fails
	ErrValidation = errors.New("validation failed")

	// ErrDuplicate is returned when an entity already exists
	ErrDuplicate = errors.New("entity already exists")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized is returned when a user is not authorized to perform an action
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when a user is forbidden from performing an action
	ErrForbidden = errors.New("forbidden")

	// ErrInternal is returned when an internal error occurs
	ErrInternal = errors.New("internal error")
)

// NotFoundError represents an error when an entity is not found
type NotFoundError struct {
	EntityType string
	ID         string
	Err        error
}

// Error returns the error message
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %s not found", e.EntityType, e.ID)
}

// Unwrap returns the underlying error
func (e *NotFoundError) Unwrap() error {
	return e.Err
}

// Is checks if the target error is of the same type
func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(entityType, id string) *NotFoundError {
	return &NotFoundError{
		EntityType: entityType,
		ID:         id,
		Err:        ErrNotFound,
	}
}

// ValidationError represents an error when entity validation fails
type ValidationError struct {
	EntityType string
	Field      string
	Reason     string
	Err        error
}

// Error returns the error message
func (e *ValidationError) Error() string {
	if e.Field != "" {
		// Format field name with spaces for better readability
		fieldName := e.Field
		// Convert camelCase to space-separated words (e.g., "firstName" to "first name")
		for i := 0; i < len(fieldName); i++ {
			if i > 0 && fieldName[i] >= 'A' && fieldName[i] <= 'Z' {
				fieldName = fieldName[:i] + " " + fieldName[i:]
				i++
			}
		}
		fieldName = strings.ToLower(fieldName)
		return fmt.Sprintf("validation failed for %s: field %s %s", e.EntityType, fieldName, e.Reason)
	}
	return fmt.Sprintf("validation failed for %s: %s", e.EntityType, e.Reason)
}

// Unwrap returns the underlying error
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// Is checks if the target error is of the same type
func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

// NewValidationError creates a new ValidationError
func NewValidationError(entityType, field, reason string) *ValidationError {
	return &ValidationError{
		EntityType: entityType,
		Field:      field,
		Reason:     reason,
		Err:        ErrValidation,
	}
}

// TransactionError represents an error during a database transaction
type TransactionError struct {
	Operation string
	Err       error
}

// Error returns the error message
func (e *TransactionError) Error() string {
	return fmt.Sprintf("transaction %s failed: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying error
func (e *TransactionError) Unwrap() error {
	return e.Err
}

// NewTransactionError creates a new TransactionError
func NewTransactionError(operation string, err error) *TransactionError {
	return &TransactionError{
		Operation: operation,
		Err:       err,
	}
}

// DatabaseError represents an error during a database operation
type DatabaseError struct {
	Operation  string
	EntityType string
	Err        error
}

// Error returns the error message
func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database operation %s on %s failed: %v", e.Operation, e.EntityType, e.Err)
}

// Unwrap returns the underlying error
func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// NewDatabaseError creates a new DatabaseError
func NewDatabaseError(operation, entityType string, err error) *DatabaseError {
	return &DatabaseError{
		Operation:  operation,
		EntityType: entityType,
		Err:        err,
	}
}
