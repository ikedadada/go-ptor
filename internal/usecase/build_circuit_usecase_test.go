package usecase_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/domain/aggregate"
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
	"ikedadada/go-ptor/internal/usecase/service"
)

type mockRelayRepo struct {
	online        []*entity.Relay
	findByIDRelay *entity.Relay
	err           error
}

func (m *mockRelayRepo) AllOnline() ([]*entity.Relay, error) {
	return m.online, m.err
}
func (m *mockRelayRepo) FindByID(id value_object.RelayID) (*entity.Relay, error) {
	if m.findByIDRelay != nil && m.findByIDRelay.ID().Equal(id) {
		return m.findByIDRelay, m.err
	}
	for _, r := range m.online {
		if r.ID().Equal(id) {
			return r, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockRelayRepo) Save(_ *entity.Relay) error { return nil }

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

type mockDialer struct {
	dialCalled    int
	sendCalled    int
	createdCalled int
	destroyCalled int
}

func (m *mockDialer) ConnectToRelay(string) (net.Conn, error) {
	m.dialCalled++
	return dummyConn{}, nil
}
func (m *mockDialer) SendExtendCell(net.Conn, *aggregate.RelayCell) error {
	m.sendCalled++
	return nil
}
func (m *mockDialer) WaitForCreatedResponse(net.Conn) ([]byte, error) {
	m.createdCalled++
	kp, _ := ecdh.X25519().GenerateKey(rand.Reader)
	var pub [32]byte
	copy(pub[:], kp.PublicKey().Bytes())
	b, _ := value_object.EncodeCreatedPayload(&value_object.CreatedPayload{RelayPub: pub})
	return b, nil
}
func (m *mockDialer) TeardownCircuit(net.Conn, value_object.CircuitID) error {
	m.destroyCalled++
	return nil
}

type dummyConn struct{}

func (dummyConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (dummyConn) Write(b []byte) (int, error)      { return len(b), nil }
func (dummyConn) Close() error                     { return nil }
func (dummyConn) LocalAddr() net.Addr              { return nil }
func (dummyConn) RemoteAddr() net.Addr             { return nil }
func (dummyConn) SetDeadline(time.Time) error      { return nil }
func (dummyConn) SetReadDeadline(time.Time) error  { return nil }
func (dummyConn) SetWriteDeadline(time.Time) error { return nil }

func makeTestRelay(id string) (*entity.Relay, error) {
	relayID, err := value_object.NewRelayID(id)
	if err != nil {
		return nil, err
	}
	endpoint, _ := value_object.NewEndpoint("127.0.0.1", 9000)
	pubKey := value_object.RSAPubKey{} // Use a zero-value or mock key for testing
	relay := entity.NewRelay(relayID, endpoint, pubKey)
	relay.SetOnline() // Ensure the relay is marked as online
	return relay, nil
}

func TestBuildCircuitUseCase_Handle_Table(t *testing.T) {
	relay, err := makeTestRelay("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("setup relay: %v", err)
	}
	exitRelay, _ := makeTestRelay("550e8400-e29b-41d4-a716-446655440000")
	exitRelay.SetOnline()

	tests := []struct {
		name          string
		online        []*entity.Relay
		findByIDRelay *entity.Relay
		relayErr      error
		saveErr       error
		hops          int
		expectsErr    bool
	}{
		{"ok", []*entity.Relay{relay, relay, relay}, exitRelay, nil, nil, 3, false},
		{"not enough relays", []*entity.Relay{relay}, nil, nil, nil, 3, true},
		{"repo error", nil, nil, errors.New("repo error"), nil, 3, true},
		{"save error", []*entity.Relay{relay, relay, relay}, exitRelay, nil, errors.New("save error"), 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relayRepo := &mockRelayRepo{online: tt.online, findByIDRelay: tt.findByIDRelay, err: tt.relayErr}
			circuitRepo := &mockCircuitRepo{err: tt.saveErr}
			dialer := &mockDialer{}
			crypto := service.NewCryptoService()
			uc := usecase.NewBuildCircuitUseCase(relayRepo, circuitRepo, dialer, crypto)

			out, err := uc.Handle(usecase.BuildCircuitInput{Hops: tt.hops, ExitRelayID: "550e8400-e29b-41d4-a716-446655440000"})
			if tt.expectsErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectsErr && out.CircuitID == "" {
				t.Errorf("expected CircuitID")
			}
		})
	}
}
