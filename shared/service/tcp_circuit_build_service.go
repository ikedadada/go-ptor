package service

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"ikedadada/go-ptor/shared/domain/aggregate"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// CircuitBuildService abstracts the network operations needed during circuit build.
type CircuitBuildService interface {
	// ConnectToRelay connects to the given relay address.
	ConnectToRelay(addr string) (net.Conn, error)
	// SendExtendCell writes a cell to the relay.
	SendExtendCell(conn net.Conn, cell *aggregate.RelayCell) error
	// WaitForCreatedResponse waits for a CREATED payload from the relay.
	WaitForCreatedResponse(conn net.Conn) ([]byte, error)
	// TeardownCircuit notifies the relay about circuit teardown.
	TeardownCircuit(conn net.Conn, cid vo.CircuitID) error
}

// TCPCircuitBuildService implements service.CircuitBuildService over raw TCP connections.
type TCPCircuitBuildService struct{}

// NewTCPCircuitBuildService returns a CircuitBuildService using TCP.
func NewTCPCircuitBuildService() CircuitBuildService { return &TCPCircuitBuildService{} }

func (TCPCircuitBuildService) ConnectToRelay(addr string) (net.Conn, error) {
	return net.Dial("tcp", addr)
}

func (TCPCircuitBuildService) SendExtendCell(conn net.Conn, c *aggregate.RelayCell) error {
	buf, err := c.Encode()
	if err != nil {
		return err
	}
	packet := append(c.CircuitID().Bytes(), buf...)
	_, err = conn.Write(packet)
	return err
}

func (TCPCircuitBuildService) WaitForCreatedResponse(conn net.Conn) ([]byte, error) {
	var hdr [20]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return nil, err
	}
	if vo.CellCommand(hdr[16]) != vo.CmdCreated || vo.ProtocolVersion(hdr[17]) != vo.ProtocolV1 {
		return nil, fmt.Errorf("invalid created header")
	}
	l := binary.BigEndian.Uint16(hdr[18:20])
	if l == 0 {
		return nil, fmt.Errorf("no payload")
	}
	payload := make([]byte, l)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (TCPCircuitBuildService) TeardownCircuit(conn net.Conn, cid vo.CircuitID) error {
	var buf [20]byte
	copy(buf[:16], cid.Bytes())
	buf[18] = 0xFE
	_, err := conn.Write(buf[:])
	return err
}
