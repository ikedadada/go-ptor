package entity

import (
	"net"
	"time"

	vo "ikedadada/go-ptor/internal/domain/value_object"
)

// MessageType is defined in circuit.go to avoid duplication

// ConnState represents per-circuit connection information held by a relay.
type ConnState struct {
	key                 vo.AESKey
	baseNonce           vo.Nonce
	beginCounter        uint64 // Counter for BEGIN commands
	dataCounter         uint64 // Counter for DATA commands (downstream)
	upstreamDataCounter uint64 // Counter for upstream DATA commands
	up                  net.Conn
	down                net.Conn
	last                time.Time
	hidden              bool
	served              bool
}

// NewConnState returns a new ConnState instance.
func NewConnState(key vo.AESKey, nonce vo.Nonce, up, down net.Conn) *ConnState {
	return &ConnState{key: key, baseNonce: nonce, beginCounter: 0, dataCounter: 0, upstreamDataCounter: 0, up: up, down: down, last: time.Now(), hidden: false, served: false}
}

// NewConnStateWithCounters returns a new ConnState instance preserving counter values.
func NewConnStateWithCounters(key vo.AESKey, nonce vo.Nonce, up, down net.Conn, beginCounter, dataCounter uint64) *ConnState {
	return &ConnState{key: key, baseNonce: nonce, beginCounter: beginCounter, dataCounter: dataCounter, upstreamDataCounter: 0, up: up, down: down, last: time.Now(), hidden: false, served: false}
}

// Key returns the symmetric key for this circuit hop.
func (s *ConnState) Key() vo.AESKey  { return s.key }
func (s *ConnState) Nonce() vo.Nonce { return s.baseNonce }

// GetCounters returns the current counter values
func (s *ConnState) GetCounters() (beginCounter, dataCounter uint64) {
	return s.beginCounter, s.dataCounter
}

// BeginNonce generates the next unique nonce for BEGIN commands
func (s *ConnState) BeginNonce() vo.Nonce {
	nonce := s.baseNonce

	// XOR begin counter into last 8 bytes
	counter := s.beginCounter
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}

	s.beginCounter++
	return nonce
}

// DataNonce generates the next unique nonce for DATA commands
func (s *ConnState) DataNonce() vo.Nonce {
	nonce := s.baseNonce

	// XOR data counter into last 8 bytes
	counter := s.dataCounter
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}

	s.dataCounter++
	return nonce
}

// UpstreamDataNonce generates the next unique nonce for upstream DATA commands
func (s *ConnState) UpstreamDataNonce() vo.Nonce {
	nonce := s.baseNonce

	// XOR upstream data counter into last 8 bytes
	counter := s.upstreamDataCounter
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}

	s.upstreamDataCounter++
	return nonce
}

// Up returns the upstream connection.
func (s *ConnState) Up() net.Conn { return s.up }

// Down returns the downstream connection.
func (s *ConnState) Down() net.Conn { return s.down }

// Touch updates the last-used time to now.
func (s *ConnState) Touch() { s.last = time.Now() }

// LastUsed reports the last time the state was accessed.
func (s *ConnState) LastUsed() time.Time { return s.last }

// SetHidden marks whether the downstream connection targets a hidden service.
func (s *ConnState) SetHidden(v bool) { s.hidden = v }

// IsHidden reports whether the downstream connection is for a hidden service.
func (s *ConnState) IsHidden() bool { return s.hidden }

// MarkServed records that the downstream ServeConn loop has started.
func (s *ConnState) MarkServed() { s.served = true }

// IsServed reports whether the downstream ServeConn loop has started.
func (s *ConnState) IsServed() bool { return s.served }

// Close closes both sides of the connection.
func (s *ConnState) Close() {
	if s.up != nil {
		s.up.Close()
	}
	if s.down != nil {
		s.down.Close()
	}
}

// GetMessageTypeNonce returns the next nonce for the given message type
func (s *ConnState) GetMessageTypeNonce(messageType MessageType) vo.Nonce {
	switch messageType {
	case MessageTypeBegin, MessageTypeConnect:
		return s.BeginNonce()
	case MessageTypeData:
		return s.DataNonce()
	case MessageTypeUpstreamData:
		return s.UpstreamDataNonce()
	default:
		return s.DataNonce()
	}
}

// IncrementCounter increments the counter for the given message type
func (s *ConnState) IncrementCounter(messageType MessageType) {
	// Note: The individual nonce methods already increment counters
	// This method is for interface compliance and future extensibility
	switch messageType {
	case MessageTypeBegin, MessageTypeConnect:
		s.beginCounter++
	case MessageTypeData:
		s.dataCounter++
	case MessageTypeUpstreamData:
		s.upstreamDataCounter++
	}
}
