package service

import (
	"net"

	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// TransmitterFactory produces a CircuitTransmitter bound to a given connection.
type TransmitterFactory interface {
	New(conn net.Conn) useSvc.CircuitTransmitter
}

// TCPTransmitterFactory creates TCPTransmitter instances.
type TCPTransmitterFactory struct{}

// New returns a CircuitTransmitter using the provided connection.
func (TCPTransmitterFactory) New(conn net.Conn) useSvc.CircuitTransmitter {
	return NewTCPTransmitter(conn)
}
