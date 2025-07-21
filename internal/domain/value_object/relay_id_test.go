package value_object_test

import (
	vo "ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

func TestRelayID_Table(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectsErr bool
	}{
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"invalid uuid", "not-a-uuid", true},
		{"invalid version", "00000000-0000-0000-0000-000000000000", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vo.NewRelayID(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error for input %s", tt.input)
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.input, err)
			}
		})
	}
}
