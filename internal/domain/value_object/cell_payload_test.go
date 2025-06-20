package value_object_test

import (
	"ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

func TestCellPayload_Table(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		expectsErr bool
	}{
		{"valid payload", make([]byte, value_object.MaxDataLen), false},
		{"too large", make([]byte, value_object.MaxDataLen+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value_object.NewCellPayload(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error for input len %d", len(tt.input))
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error for input len %d: %v", len(tt.input), err)
			}
		})
	}
}
