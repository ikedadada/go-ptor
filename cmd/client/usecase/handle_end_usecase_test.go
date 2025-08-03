package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"
	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

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
		ctrl := NewMockController(t)
		cRepo := Mock[repository.CircuitRepository](ctrl)
		WhenDouble(cRepo.Find(cir.ID())).ThenReturn(cir, nil)
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
		ctrl := NewMockController(t)
		repo := Mock[repository.CircuitRepository](ctrl)
		WhenDouble(repo.Find(cir.ID())).ThenReturn(cir, nil)
		WhenSingle(repo.Delete(cir.ID())).ThenReturn(nil)
		uc := usecase.NewHandleEndUseCase(repo)
		out, err := uc.Handle(usecase.HandleEndInput{CircuitID: cid, StreamID: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify that Delete was called with the correct circuit ID
		Verify(repo, Times(1)).Delete(cir.ID())
		if !out.Closed {
			t.Errorf("expected closed")
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := NewMockController(t)
		repo := Mock[repository.CircuitRepository](ctrl)
		WhenDouble(repo.Find(cir.ID())).ThenReturn(nil, repository.ErrNotFound)
		uc := usecase.NewHandleEndUseCase(repo)
		_, err := uc.Handle(usecase.HandleEndInput{CircuitID: cid, StreamID: sid.UInt16()})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("bad id", func(t *testing.T) {
		ctrl := NewMockController(t)
		repo := Mock[repository.CircuitRepository](ctrl)
		// No need to setup mock behavior as the error will come from parsing the bad ID
		uc := usecase.NewHandleEndUseCase(repo)
		_, err := uc.Handle(usecase.HandleEndInput{CircuitID: "bad", StreamID: 1})
		if err == nil {
			t.Errorf("expected error")
		}
	})

}
