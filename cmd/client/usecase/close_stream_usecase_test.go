package usecase_test

import (
	"errors"
	"net"
	"testing"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	"ikedadada/go-ptor/shared/service"

	. "github.com/ovechkin-dm/mockio/v2/mock"
)

func TestCloseStreamInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	st, err := circuit.OpenStream()
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	ctrl := NewMockController(t)
	conn := Mock[net.Conn](ctrl)
	circuit.SetConn(0, conn)

	tests := []struct {
		name       string
		circuitRes *entity.Circuit
		circuitErr error
		input      usecase.CloseStreamInput
		expectsErr bool
	}{
		{"ok", circuit, nil, usecase.CloseStreamInput{CircuitID: circuit.ID(), StreamID: st.ID}, false},
		{"circuit not found", nil, errors.New("not found"), usecase.CloseStreamInput{CircuitID: circuit.ID(), StreamID: st.ID}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)
			cRepo := Mock[repository.CircuitRepository](ctrl)
			peSvc := service.NewPayloadEncodingService()

			// Setup mock behavior based on test case
			WhenDouble(cRepo.Find(tt.input.CircuitID)).ThenReturn(tt.circuitRes, tt.circuitErr)

			uc := usecase.NewCloseStreamUseCase(cRepo, peSvc)
			err := uc.Handle(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
