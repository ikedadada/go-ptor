package repository

import (
	"sync"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type CircuitRepo struct {
	mu sync.RWMutex
	m  map[value_object.CircuitID]*entity.Circuit
}

func NewCircuitRepo() *CircuitRepo {
	return &CircuitRepo{m: make(map[value_object.CircuitID]*entity.Circuit)}
}

func (r *CircuitRepo) Save(c *entity.Circuit) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[c.ID()] = c
	return nil
}
func (r *CircuitRepo) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.m[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return c, nil
}
func (r *CircuitRepo) Delete(id value_object.CircuitID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.m, id)
	return nil
}
func (r *CircuitRepo) ListActive() ([]*entity.Circuit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*entity.Circuit, 0, len(r.m))
	for _, c := range r.m {
		out = append(out, c)
	}
	return out, nil
}
