package repository_test

import (
	"errors"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	repoimpl "ikedadada/go-ptor/internal/infrastructure/repository"
)

func TestCircuitTableRepo_AddFindDelete(t *testing.T) {
	tbl := repoimpl.NewCircuitTableRepository(time.Second)
	id := value_object.NewCircuitID()
	st := entity.NewConnState(value_object.AESKey{}, nil, nil)
	if err := tbl.Add(id, st); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := tbl.Find(id); err != nil {
		t.Fatalf("find: %v", err)
	}
	if err := tbl.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := tbl.Find(id); !errors.Is(err, repoif.ErrNotFound) {
		t.Fatalf("expected ErrNotFound")
	}
}

func TestCircuitTableRepo_GC(t *testing.T) {
	tbl := repoimpl.NewCircuitTableRepository(500 * time.Millisecond)
	id := value_object.NewCircuitID()
	up, down := net.Pipe()
	st := entity.NewConnState(value_object.AESKey{}, up, down)
	if err := tbl.Add(id, st); err != nil {
		t.Fatalf("add: %v", err)
	}
	time.Sleep(1200 * time.Millisecond)
	if _, err := tbl.Find(id); !errors.Is(err, repoif.ErrNotFound) {
		t.Fatalf("entry not cleaned")
	}
}
