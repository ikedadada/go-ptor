package repository

import (
	"sync"
	"time"

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type circuitTableRepository struct {
	mu  sync.RWMutex
	ttl time.Duration
	m   map[value_object.CircuitID]*entity.ConnState
}

// NewCircuitTableRepository creates an in-memory circuit table with automatic cleanup.
func NewCircuitTableRepository(ttl time.Duration) repoif.CircuitTableRepository {
	r := &circuitTableRepository{ttl: ttl, m: make(map[value_object.CircuitID]*entity.ConnState)}
	go r.gc()
	return r
}

func (r *circuitTableRepository) Add(id value_object.CircuitID, st *entity.ConnState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	st.Touch()
	r.m[id] = st
	return nil
}

func (r *circuitTableRepository) Find(id value_object.CircuitID) (*entity.ConnState, error) {
	r.mu.RLock()
	st, ok := r.m[id]
	if ok {
		st.Touch()
	}
	r.mu.RUnlock()
	if !ok {
		return nil, repoif.ErrNotFound
	}
	return st, nil
}

func (r *circuitTableRepository) Delete(id value_object.CircuitID) error {
	r.mu.Lock()
	st, ok := r.m[id]
	if ok {
		st.Close()
		delete(r.m, id)
	}
	r.mu.Unlock()
	return nil
}

func (r *circuitTableRepository) gc() {
	interval := r.ttl / 2
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	for range ticker.C {
		r.mu.Lock()
		for id, st := range r.m {
			if time.Since(st.LastUsed()) > r.ttl {
				st.Close()
				delete(r.m, id)
			}
		}
		r.mu.Unlock()
	}
}
