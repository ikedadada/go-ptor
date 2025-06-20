package value_object_test

import (
	"ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

func TestAESKey_Table(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		expectsErr bool
	}{
		{"valid 32 bytes", make([]byte, 32), false},
		{"too short", make([]byte, 16), true},
		{"too long", make([]byte, 33), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value_object.AESKeyFrom(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error for input len %d", len(tt.input))
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error for input len %d: %v", len(tt.input), err)
			}
		})
	}
}
