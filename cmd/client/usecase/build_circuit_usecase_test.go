package usecase_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	. "github.com/ovechkin-dm/mockio/v2/mock"
	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/aggregate"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

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
	relayID, err := vo.NewRelayID(id)
	if err != nil {
		return nil, err
	}
	endpoint, _ := vo.NewEndpoint("127.0.0.1", 9000)
	pubKey := vo.RSAPubKey{} // Use a zero-value or mock key for testing
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

	// Generate realistic Created payload for WaitForCreatedResponse
	kp, _ := ecdh.X25519().GenerateKey(rand.Reader)
	var pub [32]byte
	copy(pub[:], kp.PublicKey().Bytes())
	payloadEncoder := service.NewPayloadEncodingService()
	createdPayload, _ := payloadEncoder.EncodeCreatedPayload(&service.CreatedPayloadDTO{RelayPub: pub})

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
			ctrl := NewMockController(t)
			rr := Mock[repository.RelayRepository](ctrl)
			cr := Mock[repository.CircuitRepository](ctrl)
			cbSvc := Mock[service.CircuitBuildService](ctrl)
			cSvc := service.NewCryptoService()
			peSvc := service.NewPayloadEncodingService()

			// Setup mock behaviors
			WhenDouble(rr.AllOnline()).ThenReturn(tt.online, tt.relayErr)
			if tt.findByIDRelay != nil {
				relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
				WhenDouble(rr.FindByID(relayID)).ThenReturn(tt.findByIDRelay, tt.relayErr)
			}
			WhenSingle(cr.Save(Any[*entity.Circuit]())).ThenReturn(tt.saveErr)
			WhenDouble(cbSvc.ConnectToRelay(Any[string]())).ThenReturn(dummyConn{}, nil)
			WhenSingle(cbSvc.SendExtendCell(Any[net.Conn](), Any[*aggregate.RelayCell]())).ThenReturn(nil)
			WhenDouble(cbSvc.WaitForCreatedResponse(Any[net.Conn]())).ThenReturn(createdPayload, nil)
			WhenSingle(cbSvc.TeardownCircuit(Any[net.Conn](), Any[vo.CircuitID]())).ThenReturn(nil)

			uc := usecase.NewBuildCircuitUseCase(rr, cr, cbSvc, cSvc, peSvc)

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
