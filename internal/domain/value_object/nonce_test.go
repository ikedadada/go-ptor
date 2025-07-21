package value_object_test

import (
	vo "ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

func TestNonce_Table(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		expectsErr bool
	}{
		{"valid nonce", make([]byte, 12), false},
		{"too short", make([]byte, 8), true},
		{"too long", make([]byte, 13), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vo.NonceFrom(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error for input len %d", len(tt.input))
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error for input len %d: %v", len(tt.input), err)
			}
		})
	}
}
