package service

import (
	"net"
	"sync"
)

// StreamManagerService provides thread-safe stream management for circuit connections
type StreamManagerService interface {
	Add(id uint16, conn net.Conn)
	Get(id uint16) (net.Conn, bool)
	Remove(id uint16)
	CloseAll()
}

// streamManagerImpl provides a concrete implementation of StreamManagerService
type streamManagerImpl struct {
	mu sync.Mutex
	m  map[uint16]net.Conn
}

func NewStreamManagerService() StreamManagerService {
	return &streamManagerImpl{m: make(map[uint16]net.Conn)}
}

func (s *streamManagerImpl) Add(id uint16, conn net.Conn) {
	s.mu.Lock()
	s.m[id] = conn
	s.mu.Unlock()
}

func (s *streamManagerImpl) Get(id uint16) (net.Conn, bool) {
	s.mu.Lock()
	conn, ok := s.m[id]
	s.mu.Unlock()
	return conn, ok
}

func (s *streamManagerImpl) Remove(id uint16) {
	s.mu.Lock()
	if conn, ok := s.m[id]; ok {
		conn.Close()
		delete(s.m, id)
	}
	s.mu.Unlock()
}

func (s *streamManagerImpl) CloseAll() {
	s.mu.Lock()
	for _, conn := range s.m {
		conn.Close()
	}
	s.m = make(map[uint16]net.Conn)
	s.mu.Unlock()
}