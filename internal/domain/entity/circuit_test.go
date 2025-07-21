package entity_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	vo "ikedadada/go-ptor/internal/domain/value_object"
)

func TestNewCircuit_Table(t *testing.T) {
	id, err := vo.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("CircuitIDFrom: %v", err)
	}
	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("NewRelayID: %v", err)
	}
	key, err := vo.AESKeyFrom(make([]byte, 32))
	if err != nil {
		t.Fatalf("AESKeyFrom: %v", err)
	}
	nonce, err := vo.NonceFrom(make([]byte, 12))
	if err != nil {
		t.Fatalf("NonceFrom: %v", err)
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	tests := []struct {
		name       string
		relays     []vo.RelayID
		keys       []vo.AESKey
		nonces     []vo.Nonce
		expectsErr bool
	}{
		{"ok", []vo.RelayID{relayID, relayID, relayID}, []vo.AESKey{key, key, key}, []vo.Nonce{nonce, nonce, nonce}, false},
		{"mismatch len", []vo.RelayID{relayID}, []vo.AESKey{key, key}, []vo.Nonce{nonce, nonce}, true},
		{"empty", nil, nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := entity.NewCircuit(id, tt.relays, tt.keys, tt.nonces, priv)
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
	id, err := vo.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("CircuitIDFrom: %v", err)
	}
	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("NewRelayID: %v", err)
	}
	key, err := vo.AESKeyFrom(make([]byte, 32))
	if err != nil {
		t.Fatalf("AESKeyFrom: %v", err)
	}
	nonce, err := vo.NonceFrom(make([]byte, 12))
	if err != nil {
		t.Fatalf("NonceFrom: %v", err)
	}

	tests := []struct {
		name string
	}{{"open close"}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priv, _ := rsa.GenerateKey(rand.Reader, 2048)
			c, err := entity.NewCircuit(id, []vo.RelayID{relayID, relayID, relayID}, []vo.AESKey{key, key, key}, []vo.Nonce{nonce, nonce, nonce}, priv)
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
