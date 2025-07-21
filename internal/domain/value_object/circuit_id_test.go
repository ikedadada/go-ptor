package value_object_test

import (
	vo "ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

func TestCircuitID_Table(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectsErr bool
	}{
		{"valid uuid", "123e4567-e89b-12d3-a456-426614174000", false},
		{"invalid uuid", "not-a-uuid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vo.CircuitIDFrom(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error for input %s", tt.input)
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.input, err)
			}
		})
	}
}
