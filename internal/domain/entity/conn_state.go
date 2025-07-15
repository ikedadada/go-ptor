package entity

import (
	"net"
	"time"

	"ikedadada/go-ptor/internal/domain/value_object"
)

// ConnState represents per-circuit connection information held by a relay.
type ConnState struct {
	key         value_object.AESKey
	baseNonce   value_object.Nonce
	counter     uint64
	up          net.Conn
	down        net.Conn
	last        time.Time
	tbl         *StreamTable
	hidden      bool
	served      bool
}

// NewConnState returns a new ConnState instance.
func NewConnState(key value_object.AESKey, nonce value_object.Nonce, up, down net.Conn) *ConnState {
	return &ConnState{key: key, baseNonce: nonce, counter: 0, up: up, down: down, last: time.Now(), tbl: NewStreamTable(), hidden: false, served: false}
}

// Key returns the symmetric key for this circuit hop.
func (s *ConnState) Key() value_object.AESKey  { return s.key }
func (s *ConnState) Nonce() value_object.Nonce { return s.baseNonce }

// NextNonce generates the next unique nonce
func (s *ConnState) NextNonce() value_object.Nonce {
	var nonce value_object.Nonce
	nonce = s.baseNonce
	
	// XOR counter into last 8 bytes
	counter := s.counter
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}
	
	s.counter++
	return nonce
}

// Up returns the upstream connection.
func (s *ConnState) Up() net.Conn { return s.up }

// Down returns the downstream connection.
func (s *ConnState) Down() net.Conn { return s.down }

// Streams returns the table of open stream connections.
func (s *ConnState) Streams() *StreamTable { return s.tbl }

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
	if s.tbl != nil {
		s.tbl.DestroyAll()
	}
}
