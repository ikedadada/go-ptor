package usecase_test

import (
	"errors"
	"net"
	"testing"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

type mockCircuitRepoClose struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockCircuitRepoClose) Find(id vo.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.err
}
func (m *mockCircuitRepoClose) Save(*entity.Circuit) error             { return nil }
func (m *mockCircuitRepoClose) Delete(vo.CircuitID) error              { return nil }
func (m *mockCircuitRepoClose) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockTransmitterClose struct {
	err  error
	ends []struct {
		cid vo.CircuitID
		sid vo.StreamID
	}
}

func (m *mockTransmitterClose) TerminateStream(c vo.CircuitID, s vo.StreamID) error {
	m.ends = append(m.ends, struct {
		cid vo.CircuitID
		sid vo.StreamID
	}{c, s})
	return m.err
}
func (m *mockTransmitterClose) InitiateStream(vo.CircuitID, vo.StreamID, []byte) error {
	return nil
}
func (m *mockTransmitterClose) TransmitData(c vo.CircuitID, s vo.StreamID, data []byte) error {
	return nil
}
func (m *mockTransmitterClose) DestroyCircuit(vo.CircuitID) error              { return nil }
func (m *mockTransmitterClose) EstablishConnection(vo.CircuitID, []byte) error { return nil }

type closeFactory struct {
	tx service.CircuitMessagingService
}

func (m closeFactory) New(net.Conn) service.CircuitMessagingService { return m.tx }

func TestCloseStreamInteractor_Handle(t *testing.T) {
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
		fac        service.MessagingServiceFactory
		input      usecase.CloseStreamInput
		expectsErr bool
	}{
		{"ok", &mockCircuitRepoClose{circuit: circuit}, closeFactory{&mockTransmitterClose{}}, usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, false},
		{"circuit not found", &mockCircuitRepoClose{circuit: nil, err: errors.New("not found")}, closeFactory{&mockTransmitterClose{}}, usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, true},
		{"bad id", &mockCircuitRepoClose{circuit: nil}, closeFactory{&mockTransmitterClose{}}, usecase.CloseStreamInput{CircuitID: "bad-uuid", StreamID: st.ID.UInt16()}, true},
		{"tx error", &mockCircuitRepoClose{circuit: circuit}, closeFactory{&mockTransmitterClose{err: errors.New("fail")}}, usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewCloseStreamUsecase(tt.repo, tt.fac)
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
		tx := &mockTransmitterClose{}
		repo := &mockCircuitRepoClose{circuit: circuit}
		uc := usecase.NewCloseStreamUsecase(repo, closeFactory{tx})
		if _, err := uc.Handle(usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tx.ends) != 2 {
			t.Fatalf("expected 2 TerminateStream calls, got %d", len(tx.ends))
		}
		if tx.ends[1].sid.UInt16() != 0 {
			t.Errorf("second TerminateStream should be control END")
		}
	})
}
