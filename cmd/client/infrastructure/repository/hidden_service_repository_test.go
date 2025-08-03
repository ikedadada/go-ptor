package repository_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"
	"ikedadada/go-ptor/cmd/client/infrastructure/http"
	repoImpl "ikedadada/go-ptor/cmd/client/infrastructure/repository"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestHiddenServiceRepo_FindByAddressString(t *testing.T) {
	ctrl := NewMockController(t)
	mockClient := Mock[http.HTTPClient](ctrl)
	WhenSingle(mockClient.FetchJSON(Any[string](), Any[interface{}]())).ThenReturn(nil)

	repo, err := repoImpl.NewHiddenServiceRepository(mockClient, "http://test.com")
	if err != nil {
		t.Fatalf("NewHiddenServiceRepository: %v", err)
	}

	// Since repository starts empty, first save a hidden service
	addr := vo.HiddenAddrFromString("TEST.PTOR")
	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("NewRelayID: %v", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubKey := vo.RSAPubKey{PublicKey: &key.PublicKey}

	hs := entity.NewHiddenService(addr, relayID, pubKey)
	err = repo.Save(hs)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Test case-insensitive lookup
	found, err := repo.FindByAddressString("test.ptor")
	if err != nil {
		t.Fatalf("FindByAddressString: %v", err)
	}
	if found.Address().String() != "TEST.PTOR" {
		t.Errorf("unexpected address: got %s, want TEST.PTOR", found.Address().String())
	}

	// Test uppercase lookup
	found2, err := repo.FindByAddressString("TEST.PTOR")
	if err != nil {
		t.Fatalf("FindByAddressString uppercase: %v", err)
	}
	if found2.Address().String() != "TEST.PTOR" {
		t.Errorf("unexpected address: got %s, want TEST.PTOR", found2.Address().String())
	}
}

func TestHiddenServiceRepo_FindByAddressString_NotFound(t *testing.T) {
	ctrl := NewMockController(t)
	mockClient := Mock[http.HTTPClient](ctrl)
	WhenSingle(mockClient.FetchJSON(Any[string](), Any[interface{}]())).ThenReturn(nil)

	repo, err := repoImpl.NewHiddenServiceRepository(mockClient, "http://test.com")
	if err != nil {
		t.Fatalf("NewHiddenServiceRepository: %v", err)
	}

	_, err = repo.FindByAddressString("nonexistent.ptor")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestHiddenServiceRepo_All(t *testing.T) {
	ctrl := NewMockController(t)
	mockClient := Mock[http.HTTPClient](ctrl)
	WhenSingle(mockClient.FetchJSON(Any[string](), Any[interface{}]())).ThenReturn(nil)

	repo, err := repoImpl.NewHiddenServiceRepository(mockClient, "http://test.com")
	if err != nil {
		t.Fatalf("NewHiddenServiceRepository: %v", err)
	}

	// Add test data
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubKey := vo.RSAPubKey{PublicKey: &key.PublicKey}

	// Save two hidden services
	relayID1, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	hs1 := entity.NewHiddenService(vo.HiddenAddrFromString("test1.ptor"), relayID1, pubKey)
	repo.Save(hs1)

	relayID2, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440001")
	hs2 := entity.NewHiddenService(vo.HiddenAddrFromString("test2.ptor"), relayID2, pubKey)
	repo.Save(hs2)

	all, err := repo.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("expected 2 hidden services, got %d", len(all))
	}
}

func TestHiddenServiceRepo_Save(t *testing.T) {
	ctrl := NewMockController(t)
	mockClient := Mock[http.HTTPClient](ctrl)
	WhenSingle(mockClient.FetchJSON(Any[string](), Any[interface{}]())).ThenReturn(nil)

	repo, err := repoImpl.NewHiddenServiceRepository(mockClient, "http://test.com")
	if err != nil {
		t.Fatalf("NewHiddenServiceRepository: %v", err)
	}

	addr := vo.HiddenAddrFromString("new.ptor")
	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("NewRelayID: %v", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubKey := vo.RSAPubKey{PublicKey: &key.PublicKey}

	hs := entity.NewHiddenService(addr, relayID, pubKey)

	err = repo.Save(hs)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify it was saved
	found, err := repo.FindByAddressString("new.ptor")
	if err != nil {
		t.Fatalf("FindByAddressString after save: %v", err)
	}
	if found.Address().String() != "new.ptor" {
		t.Errorf("unexpected address after save: got %s, want new.ptor", found.Address().String())
	}
}
