package repository_test

import (
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/repository"
)

func makeTestRelay(status entity.RelayStatus, idStr string) *entity.Relay {
	relayID, _ := value_object.NewRelayID(idStr)
	end, _ := value_object.NewEndpoint("127.0.0.1", 5000)
	pk, _ := value_object.RSAPubKeyFromPEM([]byte("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA7\n-----END PUBLIC KEY-----\n"))
	rel := entity.NewRelay(relayID, end, pk)
	switch status {
	case entity.Online:
		rel.SetOnline()
	case entity.Offline:
		rel.SetOffline()
	}
	return rel
}

func TestRelayRepo_Save_FindByID(t *testing.T) {
	repo := repository.NewRelayRepo()
	rel := makeTestRelay(entity.Online, "550e8400-e29b-41d4-a716-446655440000")
	err := repo.Save(rel)
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
	repo := repository.NewRelayRepo()
	relayID, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440001")
	_, err := repo.FindByID(relayID)
	if !errors.Is(err, repoif.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRelayRepo_AllOnline(t *testing.T) {
	repo := repository.NewRelayRepo()
	on := makeTestRelay(entity.Online, "550e8400-e29b-41d4-a716-446655440000")
	off := makeTestRelay(entity.Offline, "550e8400-e29b-41d4-a716-446655440001")
	repo.Save(on)
	repo.Save(off)
	list, err := repo.AllOnline()
	if err != nil {
		t.Fatalf("AllOnline error: %v", err)
	}
	if len(list) != 1 || list[0] != on {
		t.Errorf("expected only online relay, got %+v", list)
	}
}
