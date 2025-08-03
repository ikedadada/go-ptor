package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"
	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

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
	rawKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	priv := vo.NewRSAPrivKey(rawKey)
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
		circuitRes *entity.Circuit
		circuitErr error
		input      usecase.OpenStreamInput
		expectsErr bool
	}{
		{"ok", circuit, nil, usecase.OpenStreamInput{CircuitID: circuit.ID().String()}, false},
		{"circuit not found", nil, errors.New("not found"), usecase.OpenStreamInput{CircuitID: "550e8400-e29b-41d4-a716-446655440000"}, true},
		{"bad id", nil, nil, usecase.OpenStreamInput{CircuitID: "bad-uuid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)
			cRepo := Mock[repository.CircuitRepository](ctrl)

			// Setup mock behavior based on test case
			if tt.input.CircuitID == "bad-uuid" {
				// For bad UUID case, the error will be from parsing, not from Find call
			} else {
				circuitID, _ := vo.CircuitIDFrom(tt.input.CircuitID)
				WhenDouble(cRepo.Find(circuitID)).ThenReturn(tt.circuitRes, tt.circuitErr)
			}

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
