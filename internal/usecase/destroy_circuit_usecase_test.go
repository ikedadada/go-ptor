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

type mockRepoDestroy struct {
	err    error
	del    value_object.CircuitID
	circ   *entity.Circuit
	events *[]string
}

func (m *mockRepoDestroy) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	return m.circ, nil
}
func (m *mockRepoDestroy) Save(*entity.Circuit) error { return nil }
func (m *mockRepoDestroy) Delete(id value_object.CircuitID) error {
	m.del = id
	if m.events != nil {
		*m.events = append(*m.events, "delete")
	}
	return m.err
}
func (m *mockRepoDestroy) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockTxDestroy struct {
	err     error
	destroy value_object.CircuitID
	ends    []value_object.StreamID
	events  *[]string
}

func (m *mockTxDestroy) SendData(value_object.CircuitID, value_object.StreamID, []byte) error {
	return nil
}
func (m *mockTxDestroy) SendEnd(_ value_object.CircuitID, s value_object.StreamID) error {
	m.ends = append(m.ends, s)
	return nil
}
func (m *mockTxDestroy) SendDestroy(c value_object.CircuitID) error {
	if m.events != nil {
		*m.events = append(*m.events, "destroy")
	}
	m.destroy = c
	return m.err
}

func makeCircuitForDestroy() (*entity.Circuit, error) {
	id := value_object.NewCircuitID()
	rid, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	c, err := entity.NewCircuit(id, []value_object.RelayID{rid}, []value_object.AESKey{key}, []value_object.Nonce{nonce}, priv)
	if err != nil {
		return nil, err
	}
	if _, err := c.OpenStream(); err != nil {
		return nil, err
	}
	return c, nil
}

func TestDestroyCircuitUsecase(t *testing.T) {
	cir, err := makeCircuitForDestroy()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	cid := cir.ID().String()

	t.Run("ok", func(t *testing.T) {
		repo := &mockRepoDestroy{circ: cir}
		tx := &mockTxDestroy{}
		uc := usecase.NewDestroyCircuitUsecase(repo, tx)
		out, err := uc.Handle(usecase.DestroyCircuitInput{CircuitID: cid})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !out.Aborted {
			t.Errorf("expected aborted true")
		}
		if repo.del.String() != cid || tx.destroy.String() != cid {
			t.Errorf("expected tx and repo with cid")
		}
		if len(tx.ends) != 1 {
			t.Errorf("expected 1 SendEnd call, got %d", len(tx.ends))
		}
	})

	t.Run("bad id", func(t *testing.T) {
		uc := usecase.NewDestroyCircuitUsecase(&mockRepoDestroy{}, &mockTxDestroy{})
		_, err := uc.Handle(usecase.DestroyCircuitInput{CircuitID: "bad"})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("tx error", func(t *testing.T) {
		repo := &mockRepoDestroy{}
		tx := &mockTxDestroy{err: errors.New("fail")}
		uc := usecase.NewDestroyCircuitUsecase(repo, tx)
		_, err := uc.Handle(usecase.DestroyCircuitInput{CircuitID: cid})
		if err == nil {
			t.Errorf("expected error")
		}
		if repo.del != (value_object.CircuitID{}) {
			t.Errorf("delete should not be called on tx error")
		}
	})

	t.Run("delete error", func(t *testing.T) {
		repo := &mockRepoDestroy{err: errors.New("fail")}
		tx := &mockTxDestroy{}
		uc := usecase.NewDestroyCircuitUsecase(repo, tx)
		_, err := uc.Handle(usecase.DestroyCircuitInput{CircuitID: cid})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("order", func(t *testing.T) {
		events := []string{}
		repo := &mockRepoDestroy{circ: cir, events: &events}
		tx := &mockTxDestroy{events: &events}
		uc := usecase.NewDestroyCircuitUsecase(repo, tx)
		if _, err := uc.Handle(usecase.DestroyCircuitInput{CircuitID: cid}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(events) != 2 || events[0] != "destroy" || events[1] != "delete" {
			t.Errorf("wrong call order: %v", events)
		}
	})
}
