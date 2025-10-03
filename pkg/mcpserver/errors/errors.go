package errors

import (
	"errors"
	"fmt"
)

// Error types for MCP server
// These are standard Go errors that will be returned from tool handlers

var (
	// ErrValidation is returned for validation errors
	ErrValidation = errors.New("validation error")
	// ErrNotFound is returned when a resource is not found
	ErrNotFound = errors.New("not found")
	// ErrInternal is returned for internal server errors
	ErrInternal = errors.New("internal error")
	// ErrStorage is returned for storage-related errors
	ErrStorage = errors.New("storage error")
	// ErrUnauthorized is returned for unauthorized access
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden is returned for forbidden access
	ErrForbidden = errors.New("forbidden")
)

// NewValidationError creates an error for validation failures
func NewValidationError(field string, err error) error {
	return fmt.Errorf("invalid parameter '%s': %w: %v", field, ErrValidation, err)
}

// NewNotFoundError creates an error for not found resources
func NewNotFoundError(resource string, id string) error {
	return fmt.Errorf("%s not found: %s: %w", resource, id, ErrNotFound)
}

// NewInternalError creates an error for internal errors
func NewInternalError(err error) error {
	return fmt.Errorf("internal error: %w: %v", ErrInternal, err)
}

// NewStorageError creates an error for storage-related errors
func NewStorageError(err error) error {
	return fmt.Errorf("storage error: %w: %v", ErrStorage, err)
}

// NewUnauthorizedError creates an error for unauthorized access
func NewUnauthorizedError(message string) error {
	return fmt.Errorf("%s: %w", message, ErrUnauthorized)
}

// NewForbiddenError creates an error for forbidden access
func NewForbiddenError(message string) error {
	return fmt.Errorf("%s: %w", message, ErrForbidden)
}

// MapError maps simple-content errors to meaningful MCP errors
// This function attempts to classify errors based on their string content
// For more precise mapping, we'd need typed errors from simple-content
func MapError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Try to classify based on error message
	// These are heuristics - ideally simple-content would export typed errors
	switch {
	case contains(errStr, "not found"):
		return fmt.Errorf("%v: %w", err, ErrNotFound)
	case contains(errStr, "validation"), contains(errStr, "invalid"):
		return fmt.Errorf("%v: %w", err, ErrValidation)
	case contains(errStr, "storage"), contains(errStr, "blob"), contains(errStr, "s3"):
		return fmt.Errorf("%v: %w", err, ErrStorage)
	case contains(errStr, "unauthorized"), contains(errStr, "authentication"):
		return fmt.Errorf("%v: %w", err, ErrUnauthorized)
	case contains(errStr, "forbidden"), contains(errStr, "permission"):
		return fmt.Errorf("%v: %w", err, ErrForbidden)
	default:
		// Return the original error
		return err
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if matchAt(s, substr, i) {
			return true
		}
	}
	return false
}

func matchAt(s, substr string, pos int) bool {
	for i := 0; i < len(substr); i++ {
		c1 := s[pos+i]
		c2 := substr[i]
		if c1 != c2 && toLower(c1) != toLower(c2) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
