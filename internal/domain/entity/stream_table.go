package entity

import (
	"net"
	"sync"

	vo "ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/util"
)

var (
	ErrDuplicate = util.ErrDuplicate
	ErrNotFound  = util.ErrNotFound
)

type StreamTable struct {
	mu sync.RWMutex
	m  map[vo.StreamID]net.Conn
}

func NewStreamTable() *StreamTable {
	return &StreamTable{m: make(map[vo.StreamID]net.Conn)}
}

func (t *StreamTable) Add(id vo.StreamID, c net.Conn) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.m[id]; ok {
		return ErrDuplicate
	}
	t.m[id] = c
	return nil
}

func (t *StreamTable) Get(id vo.StreamID) (net.Conn, error) {
	t.mu.RLock()
	c, ok := t.m[id]
	t.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (t *StreamTable) Remove(id vo.StreamID) error {
	t.mu.Lock()
	c, ok := t.m[id]
	if !ok {
		t.mu.Unlock()
		return ErrNotFound
	}
	delete(t.m, id)
	t.mu.Unlock()
	if c != nil {
		c.Close()
	}
	return nil
}

func (t *StreamTable) DestroyAll() {
	t.mu.Lock()
	for id, c := range t.m {
		if c != nil {
			c.Close()
		}
		delete(t.m, id)
	}
	t.mu.Unlock()
}
