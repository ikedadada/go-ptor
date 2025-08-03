package service

import (
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"net"
	"sync"
)

// StreamManagerService provides thread-safe stream management for circuit connections
type StreamManagerService interface {
	Add(id vo.StreamID, conn net.Conn)
	Get(id vo.StreamID) (net.Conn, bool)
	Remove(id vo.StreamID)
	CloseAll()
}

// streamManagerImpl provides a concrete implementation of StreamManagerService
type streamManagerImpl struct {
	mu sync.Mutex
	m  map[vo.StreamID]net.Conn
}

func NewStreamManagerService() StreamManagerService {
	return &streamManagerImpl{m: make(map[vo.StreamID]net.Conn)}
}

func (s *streamManagerImpl) Add(id vo.StreamID, conn net.Conn) {
	s.mu.Lock()
	s.m[id] = conn
	s.mu.Unlock()
}

func (s *streamManagerImpl) Get(id vo.StreamID) (net.Conn, bool) {
	s.mu.Lock()
	conn, ok := s.m[id]
	s.mu.Unlock()
	return conn, ok
}

func (s *streamManagerImpl) Remove(id vo.StreamID) {
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
	s.m = make(map[vo.StreamID]net.Conn)
	s.mu.Unlock()
}
