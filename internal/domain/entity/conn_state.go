package entity

import (
	"net"
	"time"

	"ikedadada/go-ptor/internal/domain/value_object"
)

// ConnState represents per-circuit connection information held by a relay.
type ConnState struct {
	key   value_object.AESKey
	nonce value_object.Nonce
	up    net.Conn
	down  net.Conn
	last  time.Time
	tbl   *StreamTable
}

// NewConnState returns a new ConnState instance.
func NewConnState(key value_object.AESKey, nonce value_object.Nonce, up, down net.Conn) *ConnState {
	return &ConnState{key: key, nonce: nonce, up: up, down: down, last: time.Now(), tbl: NewStreamTable()}
}

// Key returns the symmetric key for this circuit hop.
func (s *ConnState) Key() value_object.AESKey  { return s.key }
func (s *ConnState) Nonce() value_object.Nonce { return s.nonce }

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
