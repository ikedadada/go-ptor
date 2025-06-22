package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
)

type mockCircuitRepoShutdown struct {
	circuit   *entity.Circuit
	findErr   error
	deleteErr error
	deleted   value_object.CircuitID
}

func (m *mockCircuitRepoShutdown) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.findErr
}
func (m *mockCircuitRepoShutdown) Save(*entity.Circuit) error { return nil }
func (m *mockCircuitRepoShutdown) Delete(id value_object.CircuitID) error {
	m.deleted = id
	return m.deleteErr
}
func (m *mockCircuitRepoShutdown) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockTransmitterShutdown struct {
	endCalls []struct {
		cid value_object.CircuitID
		sid value_object.StreamID
	}
}

func (m *mockTransmitterShutdown) SendEnd(c value_object.CircuitID, s value_object.StreamID) error {
	m.endCalls = append(m.endCalls, struct {
		cid value_object.CircuitID
		sid value_object.StreamID
	}{c, s})
	return nil
}
func (m *mockTransmitterShutdown) SendData(c value_object.CircuitID, s value_object.StreamID, data []byte) error {
	return nil
}
func (m *mockTransmitterShutdown) SendDestroy(value_object.CircuitID) error { return nil }

func makeTestCircuitShutdown() (*entity.Circuit, error) {
	id, err := value_object.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	relayID, err := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	key, err := value_object.NewAESKey()
	if err != nil {
		return nil, err
	}
	nonce, err := value_object.NewNonce()
	if err != nil {
		return nil, err
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	c, err := entity.NewCircuit(id, []value_object.RelayID{relayID}, []value_object.AESKey{key}, []value_object.Nonce{nonce}, priv)
	if err != nil {
		return nil, err
	}
	if _, err := c.OpenStream(); err != nil {
		return nil, err
	}
	if _, err := c.OpenStream(); err != nil {
		return nil, err
	}
	return c, nil
}

func TestShutdownCircuitInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuitShutdown()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	cid := circuit.ID().String()

	t.Run("ok", func(t *testing.T) {
		repo := &mockCircuitRepoShutdown{circuit: circuit}
		tx := &mockTransmitterShutdown{}
		uc := usecase.NewShutdownCircuitUsecase(repo, tx)
		out, err := uc.Handle(usecase.ShutdownCircuitInput{CircuitID: cid})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !out.Success {
			t.Errorf("expected success true")
		}
		if repo.deleted.String() != cid {
			t.Errorf("expected repo.Delete called with %s", cid)
		}
		if len(tx.endCalls) < 3 { // 2 streams + control END
			t.Errorf("expected at least 3 SendEnd calls, got %d", len(tx.endCalls))
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockCircuitRepoShutdown{findErr: errors.New("not found")}
		uc := usecase.NewShutdownCircuitUsecase(repo, &mockTransmitterShutdown{})
		_, err := uc.Handle(usecase.ShutdownCircuitInput{CircuitID: cid})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("bad id", func(t *testing.T) {
		repo := &mockCircuitRepoShutdown{}
		uc := usecase.NewShutdownCircuitUsecase(repo, &mockTransmitterShutdown{})
		_, err := uc.Handle(usecase.ShutdownCircuitInput{CircuitID: "bad-uuid"})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("delete error", func(t *testing.T) {
		repo := &mockCircuitRepoShutdown{circuit: circuit, deleteErr: errors.New("fail")}
		uc := usecase.NewShutdownCircuitUsecase(repo, &mockTransmitterShutdown{})
		_, err := uc.Handle(usecase.ShutdownCircuitInput{CircuitID: cid})
		if err == nil {
			t.Errorf("expected error")
		}
	})
}
