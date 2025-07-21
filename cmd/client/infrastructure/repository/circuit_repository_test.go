package repository_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"

	repoimpl "ikedadada/go-ptor/cmd/client/infrastructure/repository"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func makeTestCircuit(id vo.CircuitID) (*entity.Circuit, error) {
	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	key, err := vo.NewAESKey()
	if err != nil {
		return nil, err
	}
	nonce, err := vo.NewNonce()
	if err != nil {
		return nil, err
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	c, err := entity.NewCircuit(id, []vo.RelayID{relayID}, []vo.AESKey{key}, []vo.Nonce{nonce}, priv)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func TestCircuitRepo_Save_Find_Delete(t *testing.T) {
	repo := repoimpl.NewCircuitRepository()
	id := vo.NewCircuitID()
	c, err := makeTestCircuit(id)
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}

	tests := []struct{ name string }{{"ok"}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err = repo.Save(c); err != nil {
				t.Fatalf("Save error: %v", err)
			}
			got, err := repo.Find(id)
			if err != nil {
				t.Fatalf("Find error: %v", err)
			}
			if got != c {
				t.Errorf("Find returned wrong circuit")
			}
			if err = repo.Delete(id); err != nil {
				t.Fatalf("Delete error: %v", err)
			}
			if _, err = repo.Find(id); !errors.Is(err, repository.ErrNotFound) {
				t.Errorf("expected ErrNotFound after delete, got %v", err)
			}
			if c.RSAPrivate() != nil {
				t.Errorf("rsa key not wiped")
			}
		})
	}
}

func TestCircuitRepo_ListActive(t *testing.T) {
	repo := repoimpl.NewCircuitRepository()
	id1 := vo.NewCircuitID()
	id2 := vo.NewCircuitID()
	c1, err := makeTestCircuit(id1)
	if err != nil {
		t.Fatalf("setup circuit1: %v", err)
	}
	c2, err := makeTestCircuit(id2)
	if err != nil {
		t.Fatalf("setup circuit2: %v", err)
	}

	tests := []struct{ name string }{{"ok"}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := repo.Save(c1); err != nil {
				t.Fatalf("save c1: %v", err)
			}
			if err := repo.Save(c2); err != nil {
				t.Fatalf("save c2: %v", err)
			}
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
		})
	}
}
