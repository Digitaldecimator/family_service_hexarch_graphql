package mocks

import (
	"context"
	"fmt"
)

// MockLocalizer is a mock implementation of the application.Localizer interface
type MockLocalizer struct {
	// Function mocks for testing specific scenarios
	LocalizeErrorFunc func(ctx context.Context, messageID string, args ...interface{}) error
	LocalizeFunc      func(ctx context.Context, messageID string, args ...interface{}) string

	// Call tracking for assertions
	LocalizeErrorCalls map[string][]interface{}
	LocalizeCalls      map[string][]interface{}

	// Default error messages
	ErrorMessages map[string]string
}

// NewMockLocalizer creates a new mock localizer
func NewMockLocalizer() *MockLocalizer {
	return &MockLocalizer{
		LocalizeErrorCalls: make(map[string][]interface{}),
		LocalizeCalls:      make(map[string][]interface{}),
		ErrorMessages: map[string]string{
			"parent.firstName.required": "First name is required",
			"parent.lastName.required":  "Last name is required",
			"parent.email.required":     "Email is required",
			"parent.birthDate.required": "Birth date is required",
			"parent.birthDate.invalid":  "Invalid birth date format",
			"parent.validation.failed":  "Parent validation failed",
			"parent.create.failed":      "Failed to create parent",
			"parent.notFound":           "Parent not found",
			"parent.update.failed":      "Failed to update parent",
			"parent.delete.failed":      "Failed to delete parent",
			"parent.list.failed":        "Failed to list parents",
			"parent.count.failed":       "Failed to count parents",

			"child.firstName.required": "First name is required",
			"child.lastName.required":  "Last name is required",
			"child.birthDate.required": "Birth date is required",
			"child.birthDate.invalid":  "Invalid birth date format",
			"child.validation.failed":  "Child validation failed",
			"child.create.failed":      "Failed to create child",
			"child.notFound":           "Child not found",
			"child.update.failed":      "Failed to update child",
			"child.delete.failed":      "Failed to delete child",
			"child.list.failed":        "Failed to list children",
			"child.count.failed":       "Failed to count children",
			"child.notFoundInParent":   "Child not found in parent",

			"transaction.begin.failed":    "Failed to begin transaction",
			"transaction.commit.failed":   "Failed to commit transaction",
			"transaction.rollback.failed": "Failed to rollback transaction",

			"error.notFound":              "not found",
			"log.child.notFoundInParent":  "Child not found in parent",
		},
	}
}

// LocalizeError localizes an error message
func (l *MockLocalizer) LocalizeError(ctx context.Context, messageID string, args ...interface{}) error {
	// Track the call
	l.LocalizeErrorCalls[messageID] = args

	if l.LocalizeErrorFunc != nil {
		return l.LocalizeErrorFunc(ctx, messageID, args...)
	}

	// Return a localized error message
	message, exists := l.ErrorMessages[messageID]
	if !exists {
		message = messageID
	}

	return fmt.Errorf("%s", message)
}

// Localize localizes a message
func (l *MockLocalizer) Localize(ctx context.Context, messageID string, args ...interface{}) string {
	// Track the call
	l.LocalizeCalls[messageID] = args

	if l.LocalizeFunc != nil {
		return l.LocalizeFunc(ctx, messageID, args...)
	}

	// Return a localized message
	message, exists := l.ErrorMessages[messageID]
	if !exists {
		message = messageID
	}

	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	return message
}

// Reset resets the state of the mock localizer
func (l *MockLocalizer) Reset() {
	l.LocalizeErrorCalls = make(map[string][]interface{})
	l.LocalizeCalls = make(map[string][]interface{})
}

// AddErrorMessage adds or updates an error message
func (l *MockLocalizer) AddErrorMessage(messageID, message string) {
	l.ErrorMessages[messageID] = message
}
