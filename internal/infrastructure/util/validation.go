package util

import (
	"fmt"
	"net"
)

// ValidationError represents a validation error with context
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// ValidateRequired checks if a string value is not empty
func ValidateRequired(value, fieldName string) error {
	if value == "" {
		return ValidationError{Field: fieldName, Message: "cannot be empty"}
	}
	return nil
}

// ValidatePositive checks if a numeric value is positive
func ValidatePositive(value int, fieldName string) error {
	if value <= 0 {
		return ValidationError{Field: fieldName, Message: "must be positive"}
	}
	return nil
}

// ValidateNotNil checks if a pointer is not nil
func ValidateNotNil(value interface{}, fieldName string) error {
	if value == nil {
		return ValidationError{Field: fieldName, Message: "cannot be nil"}
	}
	return nil
}

// ValidateMaxLength checks if a string doesn't exceed maximum length
func ValidateMaxLength(value string, maxLen int, fieldName string) error {
	if len(value) > maxLen {
		return ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("length %d exceeds maximum %d", len(value), maxLen),
		}
	}
	return nil
}

// ValidateEndpoint checks if an endpoint string is valid
func ValidateEndpoint(endpoint, fieldName string) error {
	if err := ValidateRequired(endpoint, fieldName); err != nil {
		return err
	}
	
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return ValidationError{Field: fieldName, Message: "invalid host:port format"}
	}
	
	if host == "" {
		return ValidationError{Field: fieldName, Message: "host cannot be empty"}
	}
	
	if port == "" {
		return ValidationError{Field: fieldName, Message: "port cannot be empty"}
	}
	
	return nil
}

// ValidateSliceNotEmpty checks if a slice is not empty
func ValidateSliceNotEmpty[T any](slice []T, fieldName string) error {
	if len(slice) == 0 {
		return ValidationError{Field: fieldName, Message: "cannot be empty"}
	}
	return nil
}

// ValidateIDNotZero checks if an ID value is not zero
func ValidateIDNotZero[T comparable](id T, fieldName string) error {
	var zero T
	if id == zero {
		return ValidationError{Field: fieldName, Message: "cannot be zero"}
	}
	return nil
}