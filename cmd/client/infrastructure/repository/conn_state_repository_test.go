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
	// Use shorter TTL to ensure GC runs within test timeframe
	repo := repoimpl.NewConnStateRepository(100 * time.Millisecond)
	id := vo.NewCircuitID()
	up, down := net.Pipe()
	defer up.Close()
	defer down.Close()
	st := entity.NewConnState(vo.AESKey{}, vo.Nonce{}, up, down)
	if err := repo.Add(id, st); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Verify entry exists initially
	if _, err := repo.Find(id); err != nil {
		t.Fatalf("initial find failed: %v", err)
	}

	// Wait for TTL to expire and GC to run
	// Don't call Find() repeatedly as it updates last-used time
	time.Sleep(1200 * time.Millisecond) // TTL (100ms) + GC interval (1000ms) + margin (100ms)

	// Now check if entry was garbage collected
	if _, err := repo.Find(id); !errors.Is(err, repository.ErrNotFound) {
		t.Fatal("entry was not garbage collected")
	}
}
