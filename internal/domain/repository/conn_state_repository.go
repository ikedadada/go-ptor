package repository

import (
	"net"

	"ikedadada/go-ptor/internal/domain/entity"
	vo "ikedadada/go-ptor/internal/domain/value_object"
)

// ConnStateRepository manages per-hop connection states held by a relay.
type ConnStateRepository interface {
	Add(vo.CircuitID, *entity.ConnState) error
	Find(vo.CircuitID) (*entity.ConnState, error)
	Delete(vo.CircuitID) error

	// Stream management methods
	AddStream(circuitID vo.CircuitID, streamID vo.StreamID, conn net.Conn) error
	GetStream(circuitID vo.CircuitID, streamID vo.StreamID) (net.Conn, error)
	RemoveStream(circuitID vo.CircuitID, streamID vo.StreamID) error
	DestroyAllStreams(circuitID vo.CircuitID)
}
