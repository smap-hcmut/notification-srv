package types

import "fmt"

// Custom error types for validation

// ErrMissingRequiredField creates an error for missing required field
func ErrMissingRequiredField(field string) error {
	return fmt.Errorf("missing required field: %s", field)
}

// ErrInvalidStatus creates an error for invalid status value
func ErrInvalidStatus(status string) error {
	return fmt.Errorf("invalid status: %s", status)
}

// ErrInvalidPlatform creates an error for invalid platform value
func ErrInvalidPlatform(platform string) error {
	return fmt.Errorf("invalid platform: %s", platform)
}

// ErrInvalidValue creates an error for invalid field value
func ErrInvalidValue(field, reason string) error {
	return fmt.Errorf("invalid value for %s: %s", field, reason)
}

// ErrInvalidField creates an error for invalid nested field
func ErrInvalidField(field string, cause error) error {
	return fmt.Errorf("invalid field %s: %w", field, cause)
}

// ErrInvalidArrayItem creates an error for invalid array item
func ErrInvalidArrayItem(field string, index int, cause error) error {
	return fmt.Errorf("invalid item at index %d in %s: %w", index, field, cause)
}