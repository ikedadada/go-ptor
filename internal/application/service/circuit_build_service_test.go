package service_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"ikedadada/go-ptor/internal/application/service"
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

// --- Mock RelayRepository ---
type mockRelayRepo struct {
	online []*entity.Relay
	err    error
}

func (m *mockRelayRepo) AllOnline() ([]*entity.Relay, error) {
	return m.online, m.err
}
func (m *mockRelayRepo) FindByID(_ value_object.RelayID) (*entity.Relay, error) { return nil, nil }
func (m *mockRelayRepo) Save(_ *entity.Relay) error                             { return nil }

// --- Mock CircuitRepository ---
type mockCircuitRepo struct {
	saved *entity.Circuit
	err   error
}

func (m *mockCircuitRepo) Save(c *entity.Circuit) error {
	m.saved = c
	return m.err
}
func (m *mockCircuitRepo) Find(_ value_object.CircuitID) (*entity.Circuit, error) { return nil, nil }
func (m *mockCircuitRepo) Delete(_ value_object.CircuitID) error                  { return nil }
func (m *mockCircuitRepo) ListActive() ([]*entity.Circuit, error)                 { return nil, nil }

func makeTestRelay() *entity.Relay {
	relayID, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	pkix, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &pkix.PublicKey
	pkixBytes, _ := x509.MarshalPKIXPublicKey(pub)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkixBytes})
	rsaPub, _ := value_object.RSAPubKeyFromPEM(pemBytes)
	end, _ := value_object.NewEndpoint("127.0.0.1", 5000)
	return entity.NewRelay(relayID, end, rsaPub)
}

func TestCircuitBuildService_Build_Table(t *testing.T) {
	relay := makeTestRelay()
	tests := []struct {
		name       string
		online     []*entity.Relay
		relayErr   error
		saveErr    error
		hops       int
		expectsErr bool
	}{
		{"ok", []*entity.Relay{relay, relay, relay}, nil, nil, 3, false},
		{"not enough relays", []*entity.Relay{relay}, nil, nil, 3, true},
		{"repo error", nil, errors.New("repo error"), nil, 3, true},
		{"save error", []*entity.Relay{relay, relay, relay}, nil, errors.New("save error"), 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := &mockRelayRepo{online: tt.online, err: tt.relayErr}
			cr := &mockCircuitRepo{err: tt.saveErr}
			builder := service.NewCircuitBuildService(rr, cr)
			circuit, err := builder.Build(tt.hops)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectsErr && circuit == nil {
				t.Errorf("expected circuit instance")
			}
		})
	}
}
