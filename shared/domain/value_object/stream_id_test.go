package value_object_test

import (
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"testing"
)

func TestStreamID_Table(t *testing.T) {
	tests := []struct {
		name       string
		input      uint16
		expectsErr bool
	}{
		{"valid id", 1, false},
		{"zero id", 0, true},
		{"max id", 65535, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := vo.StreamIDFrom(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error for input %d", tt.input)
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error for input %d: %v", tt.input, err)
			}
			if !tt.expectsErr && id.UInt16() != tt.input {
				t.Errorf("expected %d, got %d", tt.input, id.UInt16())
			}
		})
	}
}
