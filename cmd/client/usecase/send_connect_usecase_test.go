package usecase_test

import (
	"errors"
	"net"
	"testing"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"

	. "github.com/ovechkin-dm/mockio/v2/mock"
)

// Helper struct to track connection state for send connect tests
type sendConnectConnState struct {
	lastWritten []byte
	err         error
}

func TestSendConnectUseCase_Handle(t *testing.T) {
	cir, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}

	ctrl := NewMockController(t)
	mockConn := Mock[net.Conn](ctrl)
	cir.SetConn(0, mockConn)

	connState := &sendConnectConnState{}
	// Set up Write behavior to capture data and optionally return error
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenAnswer(func(args []any) (int, error) {
		p := args[0].([]byte)
		if connState.err != nil {
			return 0, connState.err
		}
		connState.lastWritten = make([]byte, len(p))
		copy(connState.lastWritten, p)
		return len(p), nil
	})
	cid := cir.ID()
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)
			cRepo := Mock[repository.CircuitRepository](ctrl)
			cSvc := service.NewCryptoService()
			peSvc := service.NewPayloadEncodingService()

			// Setup mock behavior based on test case
			WhenDouble(cRepo.Find(tt.input.CircuitID)).ThenReturn(tt.circuitRes, tt.circuitErr)
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
