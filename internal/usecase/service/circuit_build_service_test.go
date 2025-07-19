package service_test

import (
	"crypto/ecdh"
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
	repoif "ikedadada/go-ptor/internal/domain/repository"
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
func (m *mockRelayRepo) FindByID(id value_object.RelayID) (*entity.Relay, error) {
	for _, r := range m.online {
		if r.ID().Equal(id) {
			return r, nil
		}
	}
	return nil, repoif.ErrNotFound
}
func (m *mockRelayRepo) Save(_ *entity.Relay) error { return nil }

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
	createdCalled int
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
func (m *mockDialer) WaitCreated(net.Conn) ([]byte, error) {
	m.createdCalled++
	kp, _ := ecdh.X25519().GenerateKey(rand.Reader)
	var pub [32]byte
	copy(pub[:], kp.PublicKey().Bytes())
	b, _ := value_object.EncodeCreatedPayload(&value_object.CreatedPayload{RelayPub: pub})
	return b, nil
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
	r := entity.NewRelay(relayID, end, rsaPub)
	r.SetOnline()
	return r, nil
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
			circuit, err := builder.Build(tt.hops, value_object.RelayID{})
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
	if _, err := builder.Build(3, value_object.RelayID{}); err != nil {
		t.Fatalf("build: %v", err)
	}
	if d.dialCalled != 1 {
		t.Errorf("dial called %d", d.dialCalled)
	}
	if d.sendCalled != 3 {
		t.Errorf("send called %d", d.sendCalled)
	}
	if d.createdCalled != 3 {
		t.Errorf("created called %d", d.createdCalled)
	}
}

func TestCircuitBuildService_Build_KeyGeneration(t *testing.T) {
	relay, err := makeTestRelay()
	if err != nil {
		t.Fatalf("setup relay: %v", err)
	}
	rr := &mockRelayRepo{online: []*entity.Relay{relay, relay, relay}}
	cr := &mockCircuitRepo{}
	d := &mockDialer{}
	crypto := infraSvc.NewCryptoService()

	builder := service.NewCircuitBuildService(rr, cr, d, crypto)
	circuit, err := builder.Build(3, value_object.RelayID{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	for i := 0; i < 3; i++ {
		if circuit.HopKey(i) == (value_object.AESKey{}) {
			t.Errorf("key %d empty", i)
		}
		if circuit.HopBaseNonce(i) == (value_object.Nonce{}) {
			t.Errorf("nonce %d empty", i)
		}
	}
}

func TestCircuitBuildService_Build_WithExit(t *testing.T) {
	relay, err := makeTestRelay()
	if err != nil {
		t.Fatalf("setup relay: %v", err)
	}
	rr := &mockRelayRepo{online: []*entity.Relay{relay, relay, relay}}
	cr := &mockCircuitRepo{}
	d := &mockDialer{}
	crypto := infraSvc.NewCryptoService()

	builder := service.NewCircuitBuildService(rr, cr, d, crypto)
	circuit, err := builder.Build(3, relay.ID())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	hops := circuit.Hops()
	if !hops[len(hops)-1].Equal(relay.ID()) {
		t.Errorf("exit relay not last hop")
	}
}
