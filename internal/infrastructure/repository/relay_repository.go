package repository

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	vo "ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/http"
)

type relayRepositoryImpl struct {
	mu sync.RWMutex
	s  []*entity.Relay
}

// NewRelayRepository creates a RelayRepository pre-loaded with relay data from directory URL
func NewRelayRepository(httpClient http.HTTPClient, directoryURL string) (repository.RelayRepository, error) {

	url := strings.TrimRight(directoryURL, "/") + "/relays"
	type relayDTO struct {
		ID       string `json:"id"`
		Endpoint string `json:"endpoint"`
		PubKey   string `json:"pubkey"`
	}

	var rs []relayDTO
	if err := httpClient.FetchJSON(url, &rs); err != nil {
		return nil, err
	}

	var s = make([]*entity.Relay, 0, len(rs))

	for _, r := range rs {
		// Validate and construct RelayID
		rid, err := vo.NewRelayID(r.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid relay id %q: %w", r.ID, err)
		}

		// Parse endpoint
		host, portStr, err := net.SplitHostPort(r.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("parse endpoint %q: %w", r.Endpoint, err)
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("parse port %q: %w", portStr, err)
		}

		// Construct endpoint value object
		ep, err := vo.NewEndpoint(host, uint16(port))
		if err != nil {
			return nil, fmt.Errorf("new endpoint: %w", err)
		}

		// Parse public key
		pk, err := vo.RSAPubKeyFromPEM([]byte(r.PubKey))
		if err != nil {
			return nil, fmt.Errorf("parse pubkey: %w", err)
		}

		// Create relay entity and set online
		relay := entity.NewRelay(rid, ep, pk)
		relay.SetOnline()

		// Append to slice
		s = append(s, relay)
	}

	return &relayRepositoryImpl{
		s: s,
	}, nil
}

func (r *relayRepositoryImpl) Save(rel *entity.Relay) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Check if relay already exists
	for i, existing := range r.s {
		if existing.ID() == rel.ID() {
			// Update existing relay
			r.s[i] = rel
		}
	}
	// If not found, append new relay
	r.s = append(r.s, rel)
	return nil
}

func (r *relayRepositoryImpl) FindByID(id vo.RelayID) (*entity.Relay, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rel := range r.s {
		if rel.ID() == id {
			return rel, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *relayRepositoryImpl) AllOnline() ([]*entity.Relay, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var onlineRelays []*entity.Relay
	for _, rel := range r.s {
		if rel.Status() == entity.Online {
			onlineRelays = append(onlineRelays, rel)
		}
	}
	if len(onlineRelays) == 0 {
		return nil, repository.ErrNotFound
	}
	return onlineRelays, nil
}
