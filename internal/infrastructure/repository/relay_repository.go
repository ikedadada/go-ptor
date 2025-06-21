package repository

import (
	"sync"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type relayRepositoryImpl struct {
	mu sync.RWMutex
	m  map[value_object.RelayID]*entity.Relay
}

func NewRelayRepository() repository.RelayRepository {
	return &relayRepositoryImpl{m: make(map[value_object.RelayID]*entity.Relay)}
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
