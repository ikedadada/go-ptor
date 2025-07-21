package repository

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"ikedadada/go-ptor/cmd/client/infrastructure/http"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

type hiddenServiceRepositoryImpl struct {
	mu sync.RWMutex
	s  []*entity.HiddenService // key is lowercase address for case-insensitive lookup
}

// NewHiddenServiceRepository creates a HiddenServiceRepository pre-loaded with hidden service data from directory URL
func NewHiddenServiceRepository(httpClient http.HTTPClient, directoryURL string) (repository.HiddenServiceRepository, error) {
	url := strings.TrimRight(directoryURL, "/") + "/hidden_services"

	type hiddenServiceDTO struct {
		Address string `json:"address"`
		Relay   string `json:"relay"`
		PubKey  string `json:"pubkey"`
	}

	var hs []hiddenServiceDTO
	if err := httpClient.FetchJSON(url, &hs); err != nil {
		return nil, err
	}

	log.Printf("Loaded %d hidden services from %s", len(hs), url)

	s := make([]*entity.HiddenService, 0, len(hs))

	for _, h := range hs {
		// Parse hidden service address
		hiddenAddr := vo.HiddenAddrFromString(h.Address)

		// Parse relay ID
		relayID, err := vo.NewRelayID(h.Relay)
		if err != nil {
			return nil, fmt.Errorf("invalid relay id %q: %w", h.Relay, err)
		}

		// Parse public key (supports both RSA and Ed25519)
		pubKey, err := vo.ParsePublicKeyFromPEM([]byte(h.PubKey))
		if err != nil {
			return nil, fmt.Errorf("parse pubkey %q: %w", h.PubKey, err)
		}

		// Create hidden service entity
		h := entity.NewHiddenService(hiddenAddr, relayID, pubKey)

		// Store with lowercase key for case-insensitive lookup
		s = append(s, h)
	}

	return &hiddenServiceRepositoryImpl{
		s: s,
	}, nil
}

func (r *hiddenServiceRepositoryImpl) FindByAddress(address vo.HiddenAddr) (*entity.HiddenService, error) {
	return r.FindByAddressString(address.String())
}

func (r *hiddenServiceRepositoryImpl) FindByAddressString(address string) (*entity.HiddenService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.ToLower(address)
	for _, hs := range r.s {
		if strings.ToLower(hs.Address().String()) == key {
			return hs, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *hiddenServiceRepositoryImpl) All() ([]*entity.HiddenService, error) {
	return append([]*entity.HiddenService(nil), r.s...), nil
}

func (r *hiddenServiceRepositoryImpl) Save(hiddenService *entity.HiddenService) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Ensure address is lowercase for case-insensitive storage
	address := strings.ToLower(hiddenService.Address().String())
	// Check if hidden service already exists
	for i, hs := range r.s {
		if strings.ToLower(hs.Address().String()) == address {
			// Update existing hidden service
			r.s[i] = hiddenService
			return nil
		}
	}
	r.s = append(r.s, hiddenService)
	return nil
}
