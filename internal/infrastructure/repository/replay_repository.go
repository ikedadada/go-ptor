package repository

import (
	"sync"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type RelayRepo struct {
	mu sync.RWMutex
	m  map[value_object.RelayID]*entity.Relay
}

func NewRelayRepo() repository.RelayRepository {
	return &RelayRepo{m: make(map[value_object.RelayID]*entity.Relay)}
}

func (r *RelayRepo) Save(rel *entity.Relay) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[rel.ID()] = rel
	return nil
}

func (r *RelayRepo) FindByID(id value_object.RelayID) (*entity.Relay, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rel, ok := r.m[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return rel, nil
}

func (r *RelayRepo) AllOnline() ([]*entity.Relay, error) {
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
