package usecase_test

import (
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"
	"ikedadada/go-ptor/internal/usecase/service"
)

type mockCircuitRepoSend struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockCircuitRepoSend) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.err
}
func (m *mockCircuitRepoSend) Save(*entity.Circuit) error             { return nil }
func (m *mockCircuitRepoSend) Delete(value_object.CircuitID) error    { return nil }
func (m *mockCircuitRepoSend) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockTransmitterSend struct {
	err error
}

func (m *mockTransmitterSend) SendData(c value_object.CircuitID, s value_object.StreamID, data []byte) error {
	return m.err
}
func (m *mockTransmitterSend) SendEnd(c value_object.CircuitID, s value_object.StreamID) error {
	return nil
}
func (m *mockTransmitterSend) SendDestroy(value_object.CircuitID) error { return nil }

func TestSendDataInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	st, err := circuit.OpenStream()
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	tests := []struct {
		name       string
		repo       repository.CircuitRepository
		tx         service.CircuitTransmitter
		input      usecase.SendDataInput
		expectsErr bool
	}{
		{"ok", &mockCircuitRepoSend{circuit: circuit}, &mockTransmitterSend{}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, false},
		{"circuit not found", &mockCircuitRepoSend{circuit: nil, err: errors.New("not found")}, &mockTransmitterSend{}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"bad id", &mockCircuitRepoSend{circuit: nil}, &mockTransmitterSend{}, usecase.SendDataInput{CircuitID: "bad-uuid", StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"tx error", &mockCircuitRepoSend{circuit: circuit}, &mockTransmitterSend{err: errors.New("fail")}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"stream not active", &mockCircuitRepoSend{circuit: &entity.Circuit{}}, &mockTransmitterSend{}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewSendDataUsecase(tt.repo, tt.tx, infraSvc.NewCryptoService())
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
