package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"testing"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

type mockRepoDestroy struct {
	err    error
	del    vo.CircuitID
	circ   *entity.Circuit
	events *[]string
}

func (m *mockRepoDestroy) Find(id vo.CircuitID) (*entity.Circuit, error) {
	return m.circ, nil
}
func (m *mockRepoDestroy) Save(*entity.Circuit) error { return nil }
func (m *mockRepoDestroy) Delete(id vo.CircuitID) error {
	m.del = id
	if m.events != nil {
		*m.events = append(*m.events, "delete")
	}
	return m.err
}
func (m *mockRepoDestroy) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockTxDestroy struct {
	err     error
	destroy vo.CircuitID
	ends    []vo.StreamID
	events  *[]string
}

func (m *mockTxDestroy) TransmitData(vo.CircuitID, vo.StreamID, []byte) error {
	return nil
}
func (m *mockTxDestroy) InitiateStream(vo.CircuitID, vo.StreamID, []byte) error {
	return nil
}
func (m *mockTxDestroy) TerminateStream(_ vo.CircuitID, s vo.StreamID) error {
	m.ends = append(m.ends, s)
	return nil
}
func (m *mockTxDestroy) DestroyCircuit(c vo.CircuitID) error {
	if m.events != nil {
		*m.events = append(*m.events, "destroy")
	}
	m.destroy = c
	return m.err
}
func (m *mockTxDestroy) EstablishConnection(vo.CircuitID, []byte) error { return nil }

type destroyFactory struct {
	tx service.CircuitMessagingService
}

func (m destroyFactory) New(net.Conn) service.CircuitMessagingService { return m.tx }

func makeCircuitForDestroy() (*entity.Circuit, error) {
	id := vo.NewCircuitID()
	rid, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	c, err := entity.NewCircuit(id, []vo.RelayID{rid}, []vo.AESKey{key}, []vo.Nonce{nonce}, priv)
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
		uc := usecase.NewDestroyCircuitUsecase(repo, destroyFactory{tx})
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
			t.Errorf("expected 1 TerminateStream call, got %d", len(tx.ends))
		}
	})

	t.Run("bad id", func(t *testing.T) {
		uc := usecase.NewDestroyCircuitUsecase(&mockRepoDestroy{}, destroyFactory{&mockTxDestroy{}})
		_, err := uc.Handle(usecase.DestroyCircuitInput{CircuitID: "bad"})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("tx error", func(t *testing.T) {
		repo := &mockRepoDestroy{}
		tx := &mockTxDestroy{err: errors.New("fail")}
		uc := usecase.NewDestroyCircuitUsecase(repo, destroyFactory{tx})
		_, err := uc.Handle(usecase.DestroyCircuitInput{CircuitID: cid})
		if err == nil {
			t.Errorf("expected error")
		}
		if repo.del != (vo.CircuitID{}) {
			t.Errorf("delete should not be called on tx error")
		}
	})

	t.Run("delete error", func(t *testing.T) {
		repo := &mockRepoDestroy{err: errors.New("fail")}
		tx := &mockTxDestroy{}
		uc := usecase.NewDestroyCircuitUsecase(repo, destroyFactory{tx})
		_, err := uc.Handle(usecase.DestroyCircuitInput{CircuitID: cid})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("order", func(t *testing.T) {
		events := []string{}
		repo := &mockRepoDestroy{circ: cir, events: &events}
		tx := &mockTxDestroy{events: &events}
		uc := usecase.NewDestroyCircuitUsecase(repo, destroyFactory{tx})
		if _, err := uc.Handle(usecase.DestroyCircuitInput{CircuitID: cid}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(events) != 2 || events[0] != "destroy" || events[1] != "delete" {
			t.Errorf("wrong call order: %v", events)
		}
	})
}
