package repository

import (
	"sync"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	vo "ikedadada/go-ptor/internal/domain/value_object"
)

type circuitRepository struct {
	mu sync.RWMutex
	m  map[vo.CircuitID]*entity.Circuit
}

func NewCircuitRepository() repository.CircuitRepository {
	return &circuitRepository{m: make(map[vo.CircuitID]*entity.Circuit)}
}

func (r *circuitRepository) Save(c *entity.Circuit) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[c.ID()] = c
	return nil
}
func (r *circuitRepository) Find(id vo.CircuitID) (*entity.Circuit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.m[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return c, nil
}
func (r *circuitRepository) Delete(id vo.CircuitID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.m[id]; ok {
		c.WipeKeys()
		delete(r.m, id)
	}
	return nil
}
func (r *circuitRepository) ListActive() ([]*entity.Circuit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*entity.Circuit, 0, len(r.m))
	for _, c := range r.m {
		out = append(out, c)
	}
	return out, nil
}
