package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
)

type mockRepoEnd struct {
	cir   *entity.Circuit
	find  error
	delID value_object.CircuitID
}

func (m *mockRepoEnd) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	return m.cir, m.find
}
func (m *mockRepoEnd) Save(*entity.Circuit) error             { return nil }
func (m *mockRepoEnd) Delete(id value_object.CircuitID) error { m.delID = id; return nil }
func (m *mockRepoEnd) ListActive() ([]*entity.Circuit, error) { return nil, nil }

func makeCircuitForEnd() (*entity.Circuit, value_object.StreamID, error) {
	id := value_object.NewCircuitID()
	rid, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cir, err := entity.NewCircuit(id, []value_object.RelayID{rid}, []value_object.AESKey{key}, []value_object.Nonce{nonce}, priv)
	if err != nil {
		return nil, 0, err
	}
	st, err := cir.OpenStream()
	if err != nil {
		return nil, 0, err
	}
	return cir, st.ID, nil
}

func TestHandleEndUsecase(t *testing.T) {
	cir, sid, err := makeCircuitForEnd()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	cid := cir.ID().String()

	t.Run("stream", func(t *testing.T) {
		repo := &mockRepoEnd{cir: cir}
		uc := usecase.NewHandleEndUsecase(repo)
		out, err := uc.Handle(usecase.HandleEndInput{CircuitID: cid, StreamID: sid.UInt16()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !out.Closed {
			t.Errorf("expected closed")
		}
	})

	t.Run("circuit", func(t *testing.T) {
		repo := &mockRepoEnd{cir: cir}
		uc := usecase.NewHandleEndUsecase(repo)
		out, err := uc.Handle(usecase.HandleEndInput{CircuitID: cid, StreamID: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.delID.String() != cid {
			t.Errorf("expected delete called")
		}
		if !out.Closed {
			t.Errorf("expected closed")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockRepoEnd{cir: nil, find: repository.ErrNotFound}
		uc := usecase.NewHandleEndUsecase(repo)
		_, err := uc.Handle(usecase.HandleEndInput{CircuitID: cid, StreamID: sid.UInt16()})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("bad id", func(t *testing.T) {
		repo := &mockRepoEnd{}
		uc := usecase.NewHandleEndUsecase(repo)
		_, err := uc.Handle(usecase.HandleEndInput{CircuitID: "bad", StreamID: 1})
		if err == nil {
			t.Errorf("expected error")
		}
	})

}
