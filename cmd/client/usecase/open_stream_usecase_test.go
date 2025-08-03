package usecase_test

import (
	"errors"
	"testing"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"

	. "github.com/ovechkin-dm/mockio/v2/mock"
)

func TestOpenStreamInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	diffCirID, err := vo.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("setup circuit ID: %v", err)
	}

	tests := []struct {
		name       string
		circuitRes *entity.Circuit
		circuitErr error
		input      usecase.OpenStreamInput
		expectsErr bool
	}{
		{"ok", circuit, nil, usecase.OpenStreamInput{CircuitID: circuit.ID()}, false},
		{"circuit not found", nil, errors.New("not found"), usecase.OpenStreamInput{CircuitID: diffCirID}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)
			cRepo := Mock[repository.CircuitRepository](ctrl)

			// Setup mock behavior based on test case
			WhenDouble(cRepo.Find(tt.input.CircuitID)).ThenReturn(tt.circuitRes, tt.circuitErr)

			uc := usecase.NewOpenStreamUseCase(cRepo)
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
