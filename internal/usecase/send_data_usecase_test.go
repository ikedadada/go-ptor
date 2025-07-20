package usecase_test

import (
	"errors"
	"net"
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

type mockTransmitterSend struct{ err error }

func (m *mockTransmitterSend) SendData(c value_object.CircuitID, s value_object.StreamID, data []byte) error {
	return m.err
}
func (m *mockTransmitterSend) SendBegin(c value_object.CircuitID, s value_object.StreamID, data []byte) error {
	return m.err
}
func (m *mockTransmitterSend) SendEnd(c value_object.CircuitID, s value_object.StreamID) error {
	return nil
}
func (m *mockTransmitterSend) SendDestroy(value_object.CircuitID) error         { return nil }
func (m *mockTransmitterSend) SendConnect(value_object.CircuitID, []byte) error { return nil }

type sendFactory struct{ tx service.CircuitTransmitter }

func (m sendFactory) New(net.Conn) service.CircuitTransmitter { return m.tx }

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
		fac        infraSvc.TransmitterFactory
		input      usecase.SendDataInput
		expectsErr bool
	}{
		{"ok", &mockCircuitRepoSend{circuit: circuit}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, false},
		{"begin", &mockCircuitRepoSend{circuit: circuit}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("target"), Cmd: value_object.CmdBegin}, false},
		{"circuit not found", &mockCircuitRepoSend{circuit: nil, err: errors.New("not found")}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"bad id", &mockCircuitRepoSend{circuit: nil}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: "bad-uuid", StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"tx error", &mockCircuitRepoSend{circuit: circuit}, sendFactory{&mockTransmitterSend{err: errors.New("fail")}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"stream not active", &mockCircuitRepoSend{circuit: &entity.Circuit{}}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewSendDataUsecase(tt.repo, tt.fac, service.NewCryptoService())
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
