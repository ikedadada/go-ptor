package usecase

import (
	"testing"
)

func TestReceiveCellUseCase_Handle_InvalidCircuitID(t *testing.T) {
	// Create use case with nil dependencies (won't be called due to invalid circuit ID)
	uc := NewReceiveCellUseCase(nil, nil)

	// Test with invalid circuit ID
	_, err := uc.Handle(ReceiveCellInput{
		CircuitID: "", // Empty circuit ID should be invalid
	})

	// Assertions
	if err == nil {
		t.Error("Expected error for invalid circuit ID")
	}
}
