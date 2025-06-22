package usecase_test

import (
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
	"ikedadada/go-ptor/internal/usecase/service"
)

type mockCircuitRepoClose struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockCircuitRepoClose) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.err
}
func (m *mockCircuitRepoClose) Save(*entity.Circuit) error             { return nil }
func (m *mockCircuitRepoClose) Delete(value_object.CircuitID) error    { return nil }
func (m *mockCircuitRepoClose) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockTransmitterClose struct {
	err  error
	ends []struct {
		cid value_object.CircuitID
		sid value_object.StreamID
	}
}

func (m *mockTransmitterClose) SendEnd(c value_object.CircuitID, s value_object.StreamID) error {
	m.ends = append(m.ends, struct {
		cid value_object.CircuitID
		sid value_object.StreamID
	}{c, s})
	return m.err
}
func (m *mockTransmitterClose) SendBegin(value_object.CircuitID, value_object.StreamID, []byte) error {
	return nil
}
func (m *mockTransmitterClose) SendData(c value_object.CircuitID, s value_object.StreamID, data []byte) error {
	return nil
}
func (m *mockTransmitterClose) SendDestroy(value_object.CircuitID) error { return nil }

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
		tx         service.CircuitTransmitter
		input      usecase.CloseStreamInput
		expectsErr bool
	}{
		{"ok", &mockCircuitRepoClose{circuit: circuit}, &mockTransmitterClose{}, usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, false},
		{"circuit not found", &mockCircuitRepoClose{circuit: nil, err: errors.New("not found")}, &mockTransmitterClose{}, usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, true},
		{"bad id", &mockCircuitRepoClose{circuit: nil}, &mockTransmitterClose{}, usecase.CloseStreamInput{CircuitID: "bad-uuid", StreamID: st.ID.UInt16()}, true},
		{"tx error", &mockCircuitRepoClose{circuit: circuit}, &mockTransmitterClose{err: errors.New("fail")}, usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewCloseStreamUsecase(tt.repo, tt.tx)
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
		uc := usecase.NewCloseStreamUsecase(repo, tx)
		if _, err := uc.Handle(usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tx.ends) != 2 {
			t.Fatalf("expected 2 SendEnd calls, got %d", len(tx.ends))
		}
		if tx.ends[1].sid.UInt16() != 0 {
			t.Errorf("second SendEnd should be control END")
		}
	})
}
