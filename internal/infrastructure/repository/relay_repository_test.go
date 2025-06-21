package repository_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/repository"
)

func makeTestRelay(status entity.RelayStatus, idStr string) (*entity.Relay, error) {
	relayID, err := value_object.NewRelayID(idStr)
	if err != nil {
		return nil, err
	}
	end, err := value_object.NewEndpoint("127.0.0.1", 5000)
	if err != nil {
		return nil, err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	rel := entity.NewRelay(relayID, end, value_object.RSAPubKey{PublicKey: &key.PublicKey})
	switch status {
	case entity.Online:
		rel.SetOnline()
	case entity.Offline:
		rel.SetOffline()
	}
	return rel, nil
}

func TestRelayRepo_Save_FindByID(t *testing.T) {
	repo := repository.NewRelayRepository()
	rel, err := makeTestRelay(entity.Online, "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("setup relay: %v", err)
	}
	err = repo.Save(rel)
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}
	got, err := repo.FindByID(rel.ID())
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}
	if got != rel {
		t.Errorf("FindByID returned wrong relay")
	}
}

func TestRelayRepo_FindByID_NotFound(t *testing.T) {
	repo := repository.NewRelayRepository()
	relayID, err := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440001")
	if err != nil {
		t.Fatalf("NewRelayID: %v", err)
	}
	_, err = repo.FindByID(relayID)
	if !errors.Is(err, repoif.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRelayRepo_AllOnline(t *testing.T) {
	repo := repository.NewRelayRepository()
	on, err := makeTestRelay(entity.Online, "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("setup on relay: %v", err)
	}
	off, err := makeTestRelay(entity.Offline, "550e8400-e29b-41d4-a716-446655440001")
	if err != nil {
		t.Fatalf("setup off relay: %v", err)
	}
	if err := repo.Save(on); err != nil {
		t.Fatalf("Save on: %v", err)
	}
	if err := repo.Save(off); err != nil {
		t.Fatalf("Save off: %v", err)
	}
	list, err := repo.AllOnline()
	if err != nil {
		t.Fatalf("AllOnline error: %v", err)
	}
	if len(list) != 1 || list[0] != on {
		t.Errorf("expected only online relay, got %+v", list)
	}
}
