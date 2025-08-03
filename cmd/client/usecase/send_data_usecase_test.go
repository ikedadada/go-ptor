package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/ovechkin-dm/mockio/v2/matchers"
	. "github.com/ovechkin-dm/mockio/v2/mock"
	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// Helper struct to track connection state for send data tests
type sendDataConnState struct {
	lastWritten []byte
	err         error
}

// Helper function to create connection mock for send data tests
func createSendDataMockConnection(ctrl *matchers.MockController, writeErr error) (net.Conn, *sendDataConnState) {
	mockConn := Mock[net.Conn](ctrl)
	state := &sendDataConnState{err: writeErr}

	// Set up Write behavior to capture data and optionally return error
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenAnswer(func(args []any) (int, error) {
		p := args[0].([]byte)
		if state.err != nil {
			return 0, state.err
		}
		state.lastWritten = make([]byte, len(p))
		copy(state.lastWritten, p)
		return len(p), nil
	})

	// Set up other methods with default behaviors
	WhenDouble(mockConn.Read(Any[[]byte]())).ThenReturn(0, nil)
	WhenSingle(mockConn.Close()).ThenReturn(nil)
	WhenSingle(mockConn.LocalAddr()).ThenReturn(nil)
	WhenSingle(mockConn.RemoteAddr()).ThenReturn(nil)
	WhenSingle(mockConn.SetDeadline(Any[time.Time]())).ThenReturn(nil)
	WhenSingle(mockConn.SetReadDeadline(Any[time.Time]())).ThenReturn(nil)
	WhenSingle(mockConn.SetWriteDeadline(Any[time.Time]())).ThenReturn(nil)

	return mockConn, state
}

func TestSendDataInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	st, err := circuit.OpenStream()
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Set up connection for the circuit
	ctrl := NewMockController(t)
	conn, _ := createSendDataMockConnection(ctrl, nil)
	circuit.SetConn(0, conn)

	tests := []struct {
		name       string
		circuitRes *entity.Circuit
		circuitErr error
		input      usecase.SendDataInput
		expectsErr bool
	}{
		{"ok", circuit, nil, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, false},
		{"begin", circuit, nil, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("target"), Cmd: vo.CmdBegin}, false},
		{"circuit not found", nil, errors.New("not found"), usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"bad id", nil, nil, usecase.SendDataInput{CircuitID: "bad-uuid", StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"stream not active", &entity.Circuit{}, nil, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)
			cRepo := Mock[repository.CircuitRepository](ctrl)
			cSvc := service.NewCryptoService()
			peSvc := service.NewPayloadEncodingService()

			// Setup mock behavior based on test case
			if tt.input.CircuitID == "bad-uuid" {
				// For bad UUID case, the error will be from parsing, not from Find call
			} else {
				circuitID, _ := vo.CircuitIDFrom(tt.input.CircuitID)
				WhenDouble(cRepo.Find(circuitID)).ThenReturn(tt.circuitRes, tt.circuitErr)
			}

			uc := usecase.NewSendDataUseCase(cRepo, cSvc, peSvc)
			_, err := uc.Handle(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Additional tests from send_data_roundtrip_test.go

// Helper struct to track connection state for record tests
type recordConnState struct {
	data []byte
}

// Helper function to create record connection mock
func createRecordMockConnection(ctrl *matchers.MockController) (net.Conn, *recordConnState) {
	mockConn := Mock[net.Conn](ctrl)
	state := &recordConnState{}

	// Set up Write behavior to record data (skip circuit ID)
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenAnswer(func(args []any) (int, error) {
		p := args[0].([]byte)
		if len(p) >= 16 { // Skip circuit ID
			state.data = p[16:]
		}
		return len(p), nil
	})

	// Set up other methods with default behaviors
	WhenDouble(mockConn.Read(Any[[]byte]())).ThenReturn(0, nil)
	WhenSingle(mockConn.Close()).ThenReturn(nil)
	WhenSingle(mockConn.LocalAddr()).ThenReturn(nil)
	WhenSingle(mockConn.RemoteAddr()).ThenReturn(nil)
	WhenSingle(mockConn.SetDeadline(Any[time.Time]())).ThenReturn(nil)
	WhenSingle(mockConn.SetReadDeadline(Any[time.Time]())).ThenReturn(nil)
	WhenSingle(mockConn.SetWriteDeadline(Any[time.Time]())).ThenReturn(nil)

	return mockConn, state
}

func TestSendData_OnionRoundTrip(t *testing.T) {
	hops := 3
	relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	ids := make([]vo.RelayID, hops)
	keys := make([]vo.AESKey, hops)
	nonces := make([]vo.Nonce, hops)
	for i := 0; i < hops; i++ {
		ids[i] = relayID
		k, _ := vo.NewAESKey()
		n, _ := vo.NewNonce()
		keys[i] = k
		nonces[i] = n
	}
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	cir, err := entity.NewCircuit(vo.NewCircuitID(), ids, keys, nonces, priv)
	if err != nil {
		t.Fatalf("circuit: %v", err)
	}
	st, _ := cir.OpenStream()

	ctrl := NewMockController(t)
	conn, connState := createRecordMockConnection(ctrl)
	cir.SetConn(0, conn)
	cRepo := Mock[repository.CircuitRepository](ctrl)
	WhenDouble(cRepo.Find(cir.ID())).ThenReturn(cir, nil)
	cSvc := service.NewCryptoService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewSendDataUseCase(cRepo, cSvc, peSvc)
	data := []byte("hello")
	if _, err := uc.Handle(usecase.SendDataInput{CircuitID: cir.ID().String(), StreamID: st.ID.UInt16(), Data: data}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	// Decode cell from written data
	cell, err := entity.Decode(connState.data)
	if err != nil {
		t.Fatalf("decode cell: %v", err)
	}

	// First decode the DataPayloadDTO from the cell payload
	dto, err := peSvc.DecodeDataPayload(cell.Payload)
	if err != nil {
		t.Fatalf("decode DataPayloadDTO: %v", err)
	}

	k2 := make([][32]byte, hops)
	n2 := make([][12]byte, hops)
	for i := 0; i < hops; i++ {
		k2[i] = keys[i]
		n2[i] = nonces[i]
	}
	out, err := cSvc.AESMultiOpen(k2, n2, dto.Data)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(out) != string(data) {
		t.Errorf("round-trip mismatch")
	}
}

func TestSendData_BeginRoundTrip(t *testing.T) {
	hops := 2
	relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	ids := make([]vo.RelayID, hops)
	keys := make([]vo.AESKey, hops)
	nonces := make([]vo.Nonce, hops)
	for i := 0; i < hops; i++ {
		ids[i] = relayID
		k, _ := vo.NewAESKey()
		n, _ := vo.NewNonce()
		keys[i] = k
		nonces[i] = n
	}
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	cir, err := entity.NewCircuit(vo.NewCircuitID(), ids, keys, nonces, priv)
	if err != nil {
		t.Fatalf("circuit: %v", err)
	}
	st, _ := cir.OpenStream()

	ctrl := NewMockController(t)
	conn, connState := createRecordMockConnection(ctrl)
	cir.SetConn(0, conn)
	cRepo := Mock[repository.CircuitRepository](ctrl)
	WhenDouble(cRepo.Find(cir.ID())).ThenReturn(cir, nil)
	cSvc := service.NewCryptoService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewSendDataUseCase(cRepo, cSvc, peSvc)
	payload, _ := peSvc.EncodeBeginPayload(&service.BeginPayloadDTO{StreamID: st.ID.UInt16(), Target: "example.com:80"})
	if _, err := uc.Handle(usecase.SendDataInput{CircuitID: cir.ID().String(), StreamID: st.ID.UInt16(), Data: payload, Cmd: vo.CmdBegin}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	// Decode cell from written data
	cell, err := entity.Decode(connState.data)
	if err != nil {
		t.Fatalf("decode cell: %v", err)
	}

	k2 := make([][32]byte, hops)
	n2 := make([][12]byte, hops)
	for i := 0; i < hops; i++ {
		k2[i] = keys[i]
		n2[i] = nonces[i]
	}
	out, err := cSvc.AESMultiOpen(k2, n2, cell.Payload)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(out) != string(payload) {
		t.Errorf("round-trip mismatch")
	}
}
