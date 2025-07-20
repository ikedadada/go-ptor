package repository

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/http"
)

type relayRepositoryImpl struct {
	mu sync.RWMutex
	m  map[value_object.RelayID]*entity.Relay
}

// NewRelayRepository creates a RelayRepository pre-loaded with relay data from directory URL
func NewRelayRepository(httpClient http.HTTPClient, directoryURL string) (repository.RelayRepository, error) {

	url := strings.TrimRight(directoryURL, "/") + "/"

	var d entity.Directory
	if err := httpClient.FetchJSON(url, &d); err != nil {
		return nil, err
	}

	var m = make(map[value_object.RelayID]*entity.Relay)

	for id, info := range d.Relays {
		// Validate and construct RelayID
		rid, err := value_object.NewRelayID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid relay id %q: %w", id, err)
		}

		// Parse endpoint
		host, portStr, err := net.SplitHostPort(info.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("parse endpoint %q: %w", info.Endpoint, err)
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("parse port %q: %w", portStr, err)
		}

		// Construct endpoint value object
		ep, err := value_object.NewEndpoint(host, uint16(port))
		if err != nil {
			return nil, fmt.Errorf("new endpoint: %w", err)
		}

		// Parse public key
		pk, err := value_object.RSAPubKeyFromPEM([]byte(info.PubKey))
		if err != nil {
			return nil, fmt.Errorf("parse pubkey: %w", err)
		}

		// Create relay entity and set online
		relay := entity.NewRelay(rid, ep, pk)
		relay.SetOnline()

		m[relay.ID()] = relay
	}

	return &relayRepositoryImpl{
		m: m,
	}, nil
}

func (r *relayRepositoryImpl) Save(rel *entity.Relay) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[rel.ID()] = rel
	return nil
}

func (r *relayRepositoryImpl) FindByID(id value_object.RelayID) (*entity.Relay, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rel, ok := r.m[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return rel, nil
}

func (r *relayRepositoryImpl) AllOnline() ([]*entity.Relay, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*entity.Relay, 0, len(r.m))
	for _, rel := range r.m {
		if rel.Status() == entity.Online {
			out = append(out, rel)
		}
	}
	return out, nil
}
