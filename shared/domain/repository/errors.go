package repository

import "errors"

// Common domain errors used across different layers
var (
	// ErrNotFound indicates a requested resource was not found
	ErrNotFound = errors.New("not found")

	// ErrDuplicate indicates a resource already exists
	ErrDuplicate = errors.New("already exists")

	// ErrInvalidInput indicates invalid input parameters
	ErrInvalidInput = errors.New("invalid input")

	// ErrConnectionClosed indicates a network connection was closed
	ErrConnectionClosed = errors.New("connection closed")
)

// IsNotFound checks if an error is a "not found" error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsDuplicate checks if an error is a "duplicate" error
func IsDuplicate(err error) bool {
	return errors.Is(err, ErrDuplicate)
}

// IsInvalidInput checks if an error is an "invalid input" error
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}
