package service_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase/service"
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

// --- Mock Dialer ---
type dummyConn struct{}

func (dummyConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (dummyConn) Write(b []byte) (int, error)      { return len(b), nil }
func (dummyConn) Close() error                     { return nil }
func (dummyConn) LocalAddr() net.Addr              { return nil }
func (dummyConn) RemoteAddr() net.Addr             { return nil }
func (dummyConn) SetDeadline(time.Time) error      { return nil }
func (dummyConn) SetReadDeadline(time.Time) error  { return nil }
func (dummyConn) SetWriteDeadline(time.Time) error { return nil }

type mockDialer struct {
	dialCalled    int
	sendCalled    int
	ackCalled     int
	destroyCalled int
}

func (m *mockDialer) Dial(string) (net.Conn, error) {
	m.dialCalled++
	return dummyConn{}, nil
}
func (m *mockDialer) SendCell(net.Conn, entity.Cell) error {
	m.sendCalled++
	return nil
}
func (m *mockDialer) WaitAck(net.Conn) error {
	m.ackCalled++
	return nil
}
func (m *mockDialer) SendDestroy(net.Conn, value_object.CircuitID) error {
	m.destroyCalled++
	return nil
}

func makeTestRelay() (*entity.Relay, error) {
	relayID, err := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	pkix, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	pub := &pkix.PublicKey
	pkixBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkixBytes})
	rsaPub, err := value_object.RSAPubKeyFromPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	end, err := value_object.NewEndpoint("127.0.0.1", 5000)
	if err != nil {
		return nil, err
	}
	return entity.NewRelay(relayID, end, rsaPub), nil
}

func TestCircuitBuildService_Build_Table(t *testing.T) {
	relay, err := makeTestRelay()
	if err != nil {
		t.Fatalf("setup relay: %v", err)
	}
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
			dial := &mockDialer{}
			crypto := infraSvc.NewCryptoService()
			builder := service.NewCircuitBuildService(rr, cr, dial, crypto)
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
			if !tt.expectsErr && circuit.RSAPrivate() == nil {
				t.Errorf("expected rsa key")
			}
		})
	}
}

func TestCircuitBuildService_Build_DialerUsage(t *testing.T) {
	relay, err := makeTestRelay()
	if err != nil {
		t.Fatalf("setup relay: %v", err)
	}
	rr := &mockRelayRepo{online: []*entity.Relay{relay, relay, relay}}
	cr := &mockCircuitRepo{}
	d := &mockDialer{}
	crypto := infraSvc.NewCryptoService()

	builder := service.NewCircuitBuildService(rr, cr, d, crypto)
	if _, err := builder.Build(3); err != nil {
		t.Fatalf("build: %v", err)
	}
	if d.dialCalled != 1 {
		t.Errorf("dial called %d", d.dialCalled)
	}
	if d.sendCalled != 3 {
		t.Errorf("send called %d", d.sendCalled)
	}
	if d.ackCalled != 3 {
		t.Errorf("ack called %d", d.ackCalled)
	}
}
