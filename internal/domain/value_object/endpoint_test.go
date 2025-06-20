package value_object_test

import (
	"ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

func TestEndpoint_Table(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		port       uint16
		expectsErr bool
	}{
		{"valid endpoint", "127.0.0.1", 5000, false},
		{"invalid port 0", "127.0.0.1", 0, true},
		{"invalid host", "", 5000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value_object.NewEndpoint(tt.host, tt.port)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error for host %s port %d", tt.host, tt.port)
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error for host %s port %d: %v", tt.host, tt.port, err)
			}
		})
	}
}
