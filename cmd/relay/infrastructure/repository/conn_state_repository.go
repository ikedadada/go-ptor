package repository

import (
	"net"
	"sync"
	"time"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

type connStateRepository struct {
	mu      sync.RWMutex
	ttl     time.Duration
	m       map[vo.CircuitID]*entity.ConnState
	streams map[vo.CircuitID]map[vo.StreamID]net.Conn
}

// NewConnStateRepository creates an in-memory connection state repository with automatic cleanup.
func NewConnStateRepository(ttl time.Duration) repository.ConnStateRepository {
	r := &connStateRepository{
		ttl:     ttl,
		m:       make(map[vo.CircuitID]*entity.ConnState),
		streams: make(map[vo.CircuitID]map[vo.StreamID]net.Conn),
	}
	go r.gc()
	return r
}

func (r *connStateRepository) Add(id vo.CircuitID, st *entity.ConnState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	st.Touch()
	r.m[id] = st
	return nil
}

func (r *connStateRepository) Find(id vo.CircuitID) (*entity.ConnState, error) {
	r.mu.RLock()
	st, ok := r.m[id]
	if ok {
		st.Touch()
	}
	r.mu.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return st, nil
}

func (r *connStateRepository) Delete(id vo.CircuitID) error {
	r.mu.Lock()
	st, ok := r.m[id]
	if ok {
		st.Close()
		delete(r.m, id)
		// Close all streams for this circuit
		if streams, exists := r.streams[id]; exists {
			for _, conn := range streams {
				if conn != nil {
					conn.Close()
				}
			}
			delete(r.streams, id)
		}
	}
	r.mu.Unlock()
	return nil
}

func (r *connStateRepository) gc() {
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

// Stream management methods

func (r *connStateRepository) AddStream(circuitID vo.CircuitID, streamID vo.StreamID, conn net.Conn) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.streams[circuitID] == nil {
		r.streams[circuitID] = make(map[vo.StreamID]net.Conn)
	}

	r.streams[circuitID][streamID] = conn
	return nil
}

func (r *connStateRepository) GetStream(circuitID vo.CircuitID, streamID vo.StreamID) (net.Conn, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	streams, exists := r.streams[circuitID]
	if !exists {
		return nil, repository.ErrNotFound
	}

	conn, exists := streams[streamID]
	if !exists {
		return nil, repository.ErrNotFound
	}

	return conn, nil
}

func (r *connStateRepository) RemoveStream(circuitID vo.CircuitID, streamID vo.StreamID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	streams, exists := r.streams[circuitID]
	if !exists {
		return nil
	}

	if conn, exists := streams[streamID]; exists {
		if conn != nil {
			conn.Close()
		}
		delete(streams, streamID)
	}

	// Clean up empty circuit stream map
	if len(streams) == 0 {
		delete(r.streams, circuitID)
	}

	return nil
}

func (r *connStateRepository) DestroyAllStreams(circuitID vo.CircuitID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if streams, exists := r.streams[circuitID]; exists {
		for _, conn := range streams {
			if conn != nil {
				conn.Close()
			}
		}
		delete(r.streams, circuitID)
	}
}
