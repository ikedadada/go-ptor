package usecase

import (
	"net"
	"sync"
)

// StreamManager provides thread-safe stream management
type StreamManager interface {
	Add(id uint16, conn net.Conn)
	Get(id uint16) (net.Conn, bool)
	Remove(id uint16)
	CloseAll()
}

// streamMapImpl provides a concrete implementation of StreamManager
type streamMapImpl struct {
	mu sync.Mutex
	m  map[uint16]net.Conn
}

func NewStreamManager() StreamManager {
	return &streamMapImpl{m: make(map[uint16]net.Conn)}
}

func (s *streamMapImpl) Add(id uint16, conn net.Conn) {
	s.mu.Lock()
	s.m[id] = conn
	s.mu.Unlock()
}

func (s *streamMapImpl) Get(id uint16) (net.Conn, bool) {
	s.mu.Lock()
	conn, ok := s.m[id]
	s.mu.Unlock()
	return conn, ok
}

func (s *streamMapImpl) Remove(id uint16) {
	s.mu.Lock()
	if conn, ok := s.m[id]; ok {
		conn.Close()
		delete(s.m, id)
	}
	s.mu.Unlock()
}

func (s *streamMapImpl) CloseAll() {
	s.mu.Lock()
	for _, conn := range s.m {
		conn.Close()
	}
	s.m = make(map[uint16]net.Conn)
	s.mu.Unlock()
}