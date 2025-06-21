package usecase_test

import (
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
)

type mockCircuitRepoOpen struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockCircuitRepoOpen) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.err
}
func (m *mockCircuitRepoOpen) Save(*entity.Circuit) error             { return nil }
func (m *mockCircuitRepoOpen) Delete(value_object.CircuitID) error    { return nil }
func (m *mockCircuitRepoOpen) ListActive() ([]*entity.Circuit, error) { return nil, nil }

func makeTestCircuit() *entity.Circuit {
	id, _ := value_object.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	relayID, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	c, _ := entity.NewCircuit(id, []value_object.RelayID{relayID}, []value_object.AESKey{key}, []value_object.Nonce{nonce})
	return c
}

func TestOpenStreamInteractor_Handle(t *testing.T) {
	circuit := makeTestCircuit()

	tests := []struct {
		name       string
		repo       repository.CircuitRepository
		input      usecase.OpenStreamInput
		expectsErr bool
	}{
		{"ok", &mockCircuitRepoOpen{circuit: circuit}, usecase.OpenStreamInput{CircuitID: circuit.ID().String()}, false},
		{"circuit not found", &mockCircuitRepoOpen{circuit: nil, err: errors.New("not found")}, usecase.OpenStreamInput{CircuitID: "550e8400-e29b-41d4-a716-446655440000"}, true},
		{"bad id", &mockCircuitRepoOpen{circuit: nil}, usecase.OpenStreamInput{CircuitID: "bad-uuid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewOpenStreamInteractor(tt.repo)
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
