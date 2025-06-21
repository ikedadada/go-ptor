package repository_test

import (
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/repository"
)

func makeTestCircuit(id value_object.CircuitID) *entity.Circuit {
	relayID, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	c, _ := entity.NewCircuit(id, []value_object.RelayID{relayID}, []value_object.AESKey{key}, []value_object.Nonce{nonce})
	return c
}

func TestCircuitRepo_Save_Find_Delete(t *testing.T) {
	repo := repository.NewCircuitRepo()
	id := value_object.NewCircuitID()
	c := makeTestCircuit(id)
	// Save
	err := repo.Save(c)
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}
	// Find
	got, err := repo.Find(id)
	if err != nil {
		t.Fatalf("Find error: %v", err)
	}
	if got != c {
		t.Errorf("Find returned wrong circuit")
	}
	// Delete
	err = repo.Delete(id)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	_, err = repo.Find(id)
	if !errors.Is(err, repoif.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestCircuitRepo_ListActive(t *testing.T) {
	repo := repository.NewCircuitRepo()
	id1 := value_object.NewCircuitID()
	id2 := value_object.NewCircuitID()
	c1 := makeTestCircuit(id1)
	c2 := makeTestCircuit(id2)
	repo.Save(c1)
	repo.Save(c2)
	list, err := repo.ListActive()
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 circuits, got %d", len(list))
	}
	found1, found2 := false, false
	for _, c := range list {
		if c == c1 {
			found1 = true
		}
		if c == c2 {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("not all circuits found in ListActive")
	}
}
