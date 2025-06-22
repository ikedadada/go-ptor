package entity

import (
	"errors"
	"net"
	"sync"

	"ikedadada/go-ptor/internal/domain/value_object"
)

var (
	ErrDuplicate = errors.New("stream id already exists")
	ErrNotFound  = errors.New("stream id not found")
)

type StreamTable struct {
	mu sync.RWMutex
	m  map[value_object.StreamID]net.Conn
}

func NewStreamTable() *StreamTable {
	return &StreamTable{m: make(map[value_object.StreamID]net.Conn)}
}

func (t *StreamTable) Add(id value_object.StreamID, c net.Conn) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.m[id]; ok {
		return ErrDuplicate
	}
	t.m[id] = c
	return nil
}

func (t *StreamTable) Get(id value_object.StreamID) (net.Conn, error) {
	t.mu.RLock()
	c, ok := t.m[id]
	t.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (t *StreamTable) Remove(id value_object.StreamID) error {
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
