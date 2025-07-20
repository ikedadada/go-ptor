package service

import (
	"net"

	"ikedadada/go-ptor/internal/domain/aggregate"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// CircuitDialer abstracts the network operations needed during circuit build.
type CircuitDialer interface {
	// Dial connects to the given relay address.
	Dial(addr string) (net.Conn, error)
	// SendCell writes a cell to the relay.
	SendCell(conn net.Conn, cell *aggregate.RelayCell) error
	// WaitCreated waits for a CREATED payload from the relay.
	WaitCreated(conn net.Conn) ([]byte, error)
	// SendDestroy notifies the relay about circuit teardown.
	SendDestroy(conn net.Conn, cid value_object.CircuitID) error
}
