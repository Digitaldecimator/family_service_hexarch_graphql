package mocks

import (
	"errors"
	"reflect"
)

// MockValidator is a mock implementation of the validator.Validate interface
type MockValidator struct {
	// Function mocks for testing specific scenarios
	StructFunc func(s interface{}) error
	VarFunc    func(field interface{}, tag string) error

	// Call tracking for assertions
	StructCalls []interface{}
	VarCalls    []struct {
		Field interface{}
		Tag   string
	}

	// Error simulation
	ShouldFailValidation bool
	ValidationErrors     map[string]string
}

// NewMockValidator creates a new mock validator
func NewMockValidator() *MockValidator {
	return &MockValidator{
		StructCalls: make([]interface{}, 0),
		VarCalls: make([]struct {
			Field interface{}
			Tag   string
		}, 0),
		ValidationErrors: make(map[string]string),
	}
}

// Struct validates a struct
func (v *MockValidator) Struct(s interface{}) error {
	// Track the call
	v.StructCalls = append(v.StructCalls, s)

	if v.StructFunc != nil {
		return v.StructFunc(s)
	}

	if v.ShouldFailValidation {
		return errors.New("validation failed")
	}

	// Check if we have specific validation errors for this struct type
	structType := reflect.TypeOf(s).String()
	if errorMsg, exists := v.ValidationErrors[structType]; exists {
		return errors.New(errorMsg)
	}

	return nil
}

// Var validates a variable
func (v *MockValidator) Var(field interface{}, tag string) error {
	// Track the call
	v.VarCalls = append(v.VarCalls, struct {
		Field interface{}
		Tag   string
	}{field, tag})

	if v.VarFunc != nil {
		return v.VarFunc(field, tag)
	}

	if v.ShouldFailValidation {
		return errors.New("validation failed")
	}

	return nil
}

// Reset resets the state of the mock validator
func (v *MockValidator) Reset() {
	v.StructCalls = make([]interface{}, 0)
	v.VarCalls = make([]struct {
		Field interface{}
		Tag   string
	}, 0)
	v.ShouldFailValidation = false
	v.ValidationErrors = make(map[string]string)
}

// AddValidationError adds a validation error for a specific struct type
func (v *MockValidator) AddValidationError(structType, errorMsg string) {
	v.ValidationErrors[structType] = errorMsg
}

// SetShouldFailValidation sets whether validation should fail
func (v *MockValidator) SetShouldFailValidation(shouldFail bool) {
	v.ShouldFailValidation = shouldFail
}
