package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	vo "ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
)

type mockCircuitRepoOpen struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockCircuitRepoOpen) Find(id vo.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.err
}
func (m *mockCircuitRepoOpen) Save(*entity.Circuit) error             { return nil }
func (m *mockCircuitRepoOpen) Delete(vo.CircuitID) error              { return nil }
func (m *mockCircuitRepoOpen) ListActive() ([]*entity.Circuit, error) { return nil, nil }

func makeTestCircuit() (*entity.Circuit, error) {
	id, err := vo.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	key, err := vo.NewAESKey()
	if err != nil {
		return nil, err
	}
	nonce, err := vo.NewNonce()
	if err != nil {
		return nil, err
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	c, err := entity.NewCircuit(id, []vo.RelayID{relayID}, []vo.AESKey{key}, []vo.Nonce{nonce}, priv)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func TestOpenStreamInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}

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
			uc := usecase.NewOpenStreamUsecase(tt.repo)
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
