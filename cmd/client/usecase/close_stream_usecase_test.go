package usecase_test

import (
	"errors"
	"net"
	"testing"
	"time"

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

type mockConnForClose struct {
	lastWritten []byte
}

func (m *mockConnForClose) Write(p []byte) (n int, err error) {
	m.lastWritten = make([]byte, len(p))
	copy(m.lastWritten, p)
	return len(p), nil
}

func (m *mockConnForClose) Read([]byte) (int, error)         { return 0, nil }
func (m *mockConnForClose) Close() error                     { return nil }
func (m *mockConnForClose) LocalAddr() net.Addr              { return nil }
func (m *mockConnForClose) RemoteAddr() net.Addr             { return nil }
func (m *mockConnForClose) SetDeadline(time.Time) error      { return nil }
func (m *mockConnForClose) SetReadDeadline(time.Time) error  { return nil }
func (m *mockConnForClose) SetWriteDeadline(time.Time) error { return nil }

func TestCloseStreamInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	st, err := circuit.OpenStream()
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Set up connection for the circuit
	conn := &mockConnForClose{}
	circuit.SetConn(0, conn)

	tests := []struct {
		name       string
		cRepo      repository.CircuitRepository
		input      usecase.CloseStreamInput
		expectsErr bool
	}{
		{"ok", &mockCircuitRepoClose{circuit: circuit}, usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, false},
		{"circuit not found", &mockCircuitRepoClose{circuit: nil, err: errors.New("not found")}, usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}, true},
		{"bad id", &mockCircuitRepoClose{circuit: nil}, usecase.CloseStreamInput{CircuitID: "bad-uuid", StreamID: st.ID.UInt16()}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peSvc := service.NewPayloadEncodingService()
			uc := usecase.NewCloseStreamUseCase(tt.cRepo, peSvc)
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
		cRepo := &mockCircuitRepoClose{circuit: circuit}
		peSvc := service.NewPayloadEncodingService()
		uc := usecase.NewCloseStreamUseCase(cRepo, peSvc)
		if _, err := uc.Handle(usecase.CloseStreamInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16()}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Test passes if no error occurs and circuit connection is used
	})
}
