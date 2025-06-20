package usecase_test

import (
	"errors"
	"ikedadada/go-ptor/internal/application/usecase"
	"ikedadada/go-ptor/internal/domain/entity"
	"testing"
)

type mockBuildService struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockBuildService) Build(hops int) (*entity.Circuit, error) {
	return m.circuit, m.err
}

func TestBuildCircuitUseCase_Handle_Table(t *testing.T) {
	tests := []struct {
		name       string
		circuit    *entity.Circuit
		err        error
		expectsErr bool
	}{
		{"ok", makeTestCircuit(), nil, false},
		{"error", nil, errors.New("fail"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewBuildCircuitUseCase(&mockBuildService{circuit: tt.circuit, err: tt.err})
			out, err := uc.Handle(usecase.BuildCircuitInput{Hops: 3})
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
