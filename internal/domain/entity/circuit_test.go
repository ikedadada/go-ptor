package entity_test

import (
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

func TestNewCircuit_Table(t *testing.T) {
	id, err := value_object.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("CircuitIDFrom: %v", err)
	}
	relayID, err := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("NewRelayID: %v", err)
	}
	key, err := value_object.AESKeyFrom(make([]byte, 32))
	if err != nil {
		t.Fatalf("AESKeyFrom: %v", err)
	}
	nonce, err := value_object.NonceFrom(make([]byte, 12))
	if err != nil {
		t.Fatalf("NonceFrom: %v", err)
	}
	tests := []struct {
		name       string
		relays     []value_object.RelayID
		keys       []value_object.AESKey
		nonces     []value_object.Nonce
		expectsErr bool
	}{
		{"ok", []value_object.RelayID{relayID, relayID, relayID}, []value_object.AESKey{key, key, key}, []value_object.Nonce{nonce, nonce, nonce}, false},
		{"mismatch len", []value_object.RelayID{relayID}, []value_object.AESKey{key, key}, []value_object.Nonce{nonce, nonce}, true},
		{"empty", nil, nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := entity.NewCircuit(id, tt.relays, tt.keys, tt.nonces)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectsErr && c == nil {
				t.Errorf("expected circuit instance")
			}
		})
	}
}

func TestCircuit_StreamManagement(t *testing.T) {
	id, err := value_object.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("CircuitIDFrom: %v", err)
	}
	relayID, err := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("NewRelayID: %v", err)
	}
	key, err := value_object.AESKeyFrom(make([]byte, 32))
	if err != nil {
		t.Fatalf("AESKeyFrom: %v", err)
	}
	nonce, err := value_object.NonceFrom(make([]byte, 12))
	if err != nil {
		t.Fatalf("NonceFrom: %v", err)
	}

	tests := []struct {
		name string
	}{{"open close"}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := entity.NewCircuit(id, []value_object.RelayID{relayID, relayID, relayID}, []value_object.AESKey{key, key, key}, []value_object.Nonce{nonce, nonce, nonce})
			if err != nil {
				t.Fatalf("NewCircuit: %v", err)
			}
			st, err := c.OpenStream()
			if err != nil {
				t.Fatalf("OpenStream error: %v", err)
			}
			if st.Closed {
				t.Errorf("stream should be open")
			}
			c.CloseStream(st.ID)
			if !st.Closed {
				t.Errorf("stream should be closed")
			}
			if len(c.ActiveStreams()) != 0 {
				t.Errorf("no active streams expected")
			}
		})
	}
}
