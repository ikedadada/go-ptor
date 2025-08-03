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

// Helper struct to track connection state for send connect tests
type sendConnectConnState struct {
	lastWritten []byte
	err         error
}

// Helper function to create connection mock for send connect tests
func createSendConnectMockConnection(ctrl *matchers.MockController, writeErr error) (net.Conn, *sendConnectConnState) {
	mockConn := Mock[net.Conn](ctrl)
	state := &sendConnectConnState{err: writeErr}

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

func makeTestCircuitConnect(ctrl *matchers.MockController) (*entity.Circuit, *sendConnectConnState, error) {
	id := vo.NewCircuitID()
	rid, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	conn, connState := createSendConnectMockConnection(ctrl, nil)
	cir, err := entity.NewCircuit(id, []vo.RelayID{rid}, []vo.AESKey{key}, []vo.Nonce{nonce}, priv)
	if err != nil {
		return nil, nil, err
	}
	cir.SetConn(0, conn)
	return cir, connState, nil
}

func TestSendConnectUseCase_Handle(t *testing.T) {
	ctrl := NewMockController(t)
	cir, connState, err := makeTestCircuitConnect(ctrl)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	cid := cir.ID().String()
	peSvc := service.NewPayloadEncodingService()
	payload, _ := peSvc.EncodeConnectPayload(&service.ConnectPayloadDTO{Target: "x"})

	tests := []struct {
		name       string
		circuitRes *entity.Circuit
		circuitErr error
		input      usecase.SendConnectInput
		expectsErr bool
	}{
		{"ok", cir, nil, usecase.SendConnectInput{CircuitID: cid, Target: "x"}, false},
		{"circuit not found", nil, errors.New("nf"), usecase.SendConnectInput{CircuitID: cid}, true},
		{"bad id", nil, nil, usecase.SendConnectInput{CircuitID: "bad"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)
			cRepo := Mock[repository.CircuitRepository](ctrl)
			cSvc := service.NewCryptoService()
			peSvc := service.NewPayloadEncodingService()

			// Setup mock behavior based on test case
			if tt.input.CircuitID == "bad" {
				// For bad UUID case, the error will be from parsing, not from Find call
			} else {
				circuitID, _ := vo.CircuitIDFrom(tt.input.CircuitID)
				WhenDouble(cRepo.Find(circuitID)).ThenReturn(tt.circuitRes, tt.circuitErr)
			}

			uc := usecase.NewSendConnectUseCase(cRepo, cSvc, peSvc)

			// Store expected nonces before use case execution
			k := make([][32]byte, len(cir.Hops()))
			n := make([][12]byte, len(cir.Hops()))
			for i := range cir.Hops() {
				k[i] = cir.HopKey(i)
				n[i] = cir.HopBeginNoncePeek(i)
			}

			_, err := uc.Handle(tt.input)
			if tt.expectsErr {
				if err == nil {
					t.Errorf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			// Verify that data was written to connection
			if len(connState.lastWritten) == 0 {
				t.Errorf("no data written to connection")
				return
			}

			// Extract circuit ID and cell data from written packet
			if len(connState.lastWritten) < 16 {
				t.Errorf("packet too short")
				return
			}
			cellData := connState.lastWritten[16:] // Skip circuit ID bytes

			// Decode and verify cell
			cell, err := entity.Decode(cellData)
			if err != nil {
				t.Fatalf("decode cell: %v", err)
			}
			if cell.Cmd != vo.CmdConnect {
				t.Errorf("expected CONNECT command, got %v", cell.Cmd)
			}

			// Decrypt payload and verify
			out, err := cSvc.AESMultiOpen(k, n, cell.Payload)
			if err != nil {
				t.Fatalf("decrypt: %v", err)
			}
			if tt.input.Target != "" && string(out) != string(payload) {
				t.Errorf("payload mismatch")
			}
			if tt.input.Target == "" && len(out) != 0 {
				t.Errorf("payload should be empty")
			}
		})
	}
}
