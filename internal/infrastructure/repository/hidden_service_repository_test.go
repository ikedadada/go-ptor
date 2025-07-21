package repository_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	vo "ikedadada/go-ptor/internal/domain/value_object"
	repoImpl "ikedadada/go-ptor/internal/infrastructure/repository"
)

func TestHiddenServiceRepo_FindByAddressString(t *testing.T) {
	// Create mock HTTP client with test data in new array format
	type hiddenServiceDTO struct {
		Address string `json:"address"`
		Relay   string `json:"relay"`
		PubKey  string `json:"pubkey"`
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))

	mockClient := &mockHTTPClient{
		response: []hiddenServiceDTO{
			{
				Address: "TEST.PTOR",
				Relay:   "550e8400-e29b-41d4-a716-446655440000",
				PubKey:  pemStr,
			},
		},
	}

	repo, err := repoImpl.NewHiddenServiceRepository(mockClient, "http://test.com")
	if err != nil {
		t.Fatalf("NewHiddenServiceRepository: %v", err)
	}

	// Test case-insensitive lookup
	hs, err := repo.FindByAddressString("test.ptor")
	if err != nil {
		t.Fatalf("FindByAddressString: %v", err)
	}
	if hs.Address().String() != "TEST.PTOR" {
		t.Errorf("unexpected address: got %s, want TEST.PTOR", hs.Address().String())
	}

	// Test uppercase lookup
	hs2, err := repo.FindByAddressString("TEST.PTOR")
	if err != nil {
		t.Fatalf("FindByAddressString uppercase: %v", err)
	}
	if hs2.Address().String() != "TEST.PTOR" {
		t.Errorf("unexpected address: got %s, want TEST.PTOR", hs2.Address().String())
	}
}

func TestHiddenServiceRepo_FindByAddressString_NotFound(t *testing.T) {
	type hiddenServiceDTO struct {
		Address string `json:"address"`
		Relay   string `json:"relay"`
		PubKey  string `json:"pubkey"`
	}

	mockClient := &mockHTTPClient{
		response: []hiddenServiceDTO{},
	}

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
	type hiddenServiceDTO struct {
		Address string `json:"address"`
		Relay   string `json:"relay"`
		PubKey  string `json:"pubkey"`
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))

	mockClient := &mockHTTPClient{
		response: []hiddenServiceDTO{
			{
				Address: "test1.ptor",
				Relay:   "550e8400-e29b-41d4-a716-446655440000",
				PubKey:  pemStr,
			},
			{
				Address: "test2.ptor",
				Relay:   "550e8400-e29b-41d4-a716-446655440001",
				PubKey:  pemStr,
			},
		},
	}

	repo, err := repoImpl.NewHiddenServiceRepository(mockClient, "http://test.com")
	if err != nil {
		t.Fatalf("NewHiddenServiceRepository: %v", err)
	}

	all, err := repo.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("expected 2 hidden services, got %d", len(all))
	}
}

func TestHiddenServiceRepo_Save(t *testing.T) {
	type hiddenServiceDTO struct {
		Address string `json:"address"`
		Relay   string `json:"relay"`
		PubKey  string `json:"pubkey"`
	}

	mockClient := &mockHTTPClient{
		response: []hiddenServiceDTO{},
	}

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
