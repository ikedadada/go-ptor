package usecase_test

import (
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
)

type mockBuildService struct {
	circuit *entity.Circuit
	err     error
	exit    value_object.RelayID
}

func (m *mockBuildService) Build(hops int, exit value_object.RelayID) (*entity.Circuit, error) {
	if exit != (value_object.RelayID{}) {
		m.exit = exit
	}
	return m.circuit, m.err
}

func TestBuildCircuitUseCase_Handle_Table(t *testing.T) {
	circuit, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	tests := []struct {
		name       string
		circuit    *entity.Circuit
		err        error
		expectsErr bool
	}{
		{"ok", circuit, nil, false},
		{"error", nil, errors.New("fail"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockBuildService{circuit: tt.circuit, err: tt.err}
			uc := usecase.NewBuildCircuitUseCase(ms)
			out, err := uc.Handle(usecase.BuildCircuitInput{Hops: 3, ExitRelayID: "550e8400-e29b-41d4-a716-446655440000"})
			if tt.expectsErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectsErr && out.CircuitID == "" {
				t.Errorf("expected CircuitID")
			}
			if !tt.expectsErr && ms.exit == (value_object.RelayID{}) {
				t.Errorf("exit relay not passed to service")
			}
		})
	}
}
