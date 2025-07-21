package repository_test

import (
	"errors"
	"net"
	"testing"
	"time"

	repoimpl "ikedadada/go-ptor/cmd/client/infrastructure/repository"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestConnStateRepo_AddFindDelete(t *testing.T) {
	repo := repoimpl.NewConnStateRepository(time.Second)
	id := vo.NewCircuitID()
	st := entity.NewConnState(vo.AESKey{}, vo.Nonce{}, nil, nil)
	if err := repo.Add(id, st); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := repo.Find(id); err != nil {
		t.Fatalf("find: %v", err)
	}
	if err := repo.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Find(id); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound")
	}
}

func TestConnStateRepo_GC(t *testing.T) {
	repo := repoimpl.NewConnStateRepository(500 * time.Millisecond)
	id := vo.NewCircuitID()
	up, down := net.Pipe()
	st := entity.NewConnState(vo.AESKey{}, vo.Nonce{}, up, down)
	if err := repo.Add(id, st); err != nil {
		t.Fatalf("add: %v", err)
	}
	time.Sleep(1200 * time.Millisecond)
	if _, err := repo.Find(id); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("entry not cleaned")
	}
}
