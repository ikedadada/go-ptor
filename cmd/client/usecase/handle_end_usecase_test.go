package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

type mockRepoEnd struct {
	cir   *entity.Circuit
	find  error
	delID vo.CircuitID
}

func (m *mockRepoEnd) Find(id vo.CircuitID) (*entity.Circuit, error) {
	return m.cir, m.find
}
func (m *mockRepoEnd) Save(*entity.Circuit) error             { return nil }
func (m *mockRepoEnd) Delete(id vo.CircuitID) error           { m.delID = id; return nil }
func (m *mockRepoEnd) ListActive() ([]*entity.Circuit, error) { return nil, nil }

func makeCircuitForEnd() (*entity.Circuit, vo.StreamID, error) {
	id := vo.NewCircuitID()
	rid, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	cir, err := entity.NewCircuit(id, []vo.RelayID{rid}, []vo.AESKey{key}, []vo.Nonce{nonce}, priv)
	if err != nil {
		return nil, 0, err
	}
	st, err := cir.OpenStream()
	if err != nil {
		return nil, 0, err
	}
	return cir, st.ID, nil
}

func TestHandleEndUseCase(t *testing.T) {
	cir, sid, err := makeCircuitForEnd()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	cid := cir.ID().String()

	t.Run("stream", func(t *testing.T) {
		cRepo := &mockRepoEnd{cir: cir}
		uc := usecase.NewHandleEndUseCase(cRepo)
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
		uc := usecase.NewHandleEndUseCase(repo)
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
		uc := usecase.NewHandleEndUseCase(repo)
		_, err := uc.Handle(usecase.HandleEndInput{CircuitID: cid, StreamID: sid.UInt16()})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("bad id", func(t *testing.T) {
		repo := &mockRepoEnd{}
		uc := usecase.NewHandleEndUseCase(repo)
		_, err := uc.Handle(usecase.HandleEndInput{CircuitID: "bad", StreamID: 1})
		if err == nil {
			t.Errorf("expected error")
		}
	})

}
