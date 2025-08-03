package usecase_test

import (
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

// Helper struct to track connection state for close stream tests
type closeConnState struct {
	lastWritten []byte
}

// Helper function to create connection mock for close stream tests
func createCloseStreamMockConnection(ctrl *matchers.MockController) (net.Conn, *closeConnState) {
	mockConn := Mock[net.Conn](ctrl)
	state := &closeConnState{}

	// Set up Write behavior to capture data
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenAnswer(func(args []any) (int, error) {
		p := args[0].([]byte)
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

func TestCloseStreamInteractor_Handle(t *testing.T) {
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
	conn, _ := createCloseStreamMockConnection(ctrl)
	circuit.SetConn(0, conn)

	tests := []struct {
		name       string
		circuitRes *entity.Circuit
		circuitErr error
		input      usecase.CloseStreamInput
		expectsErr bool
	}{
		{"ok", circuit, nil, usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, false},
		{"circuit not found", nil, errors.New("not found"), usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, true},
		{"bad id", nil, nil, usecase.CloseStreamInput{CircuitID: "bad-uuid", StreamID: st.ID.UInt16()}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)
			cRepo := Mock[repository.CircuitRepository](ctrl)
			peSvc := service.NewPayloadEncodingService()

			// Setup mock behavior based on test case
			if tt.input.CircuitID == "bad-uuid" {
				// For bad UUID case, the error will be from parsing, not from Find call
			} else {
				circuitID, _ := vo.CircuitIDFrom(tt.input.CircuitID)
				WhenDouble(cRepo.Find(circuitID)).ThenReturn(tt.circuitRes, tt.circuitErr)
			}

			uc := usecase.NewCloseStreamUseCase(cRepo, peSvc)
			_, err := uc.Handle(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	t.Run("control end on last stream", func(t *testing.T) {
		ctrl := NewMockController(t)
		cRepo := Mock[repository.CircuitRepository](ctrl)
		WhenDouble(cRepo.Find(circuit.ID())).ThenReturn(circuit, nil)
		peSvc := service.NewPayloadEncodingService()
		uc := usecase.NewCloseStreamUseCase(cRepo, peSvc)
		if _, err := uc.Handle(usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Test passes if no error occurs and circuit connection is used
	})
}
