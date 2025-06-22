package service

import (
	"net"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// CircuitDialer abstracts the network operations needed during circuit build.
type CircuitDialer interface {
	// Dial connects to the given relay address.
	Dial(addr string) (net.Conn, error)
	// SendCell writes a cell to the relay.
	SendCell(conn net.Conn, cell entity.Cell) error
	// WaitAck blocks until an ACK for the given circuit is received.
	WaitAck(conn net.Conn) error
	// SendDestroy notifies the relay about circuit teardown.
	SendDestroy(conn net.Conn, cid value_object.CircuitID) error
}
