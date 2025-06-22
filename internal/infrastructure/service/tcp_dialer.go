package service

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// TCPDialer implements service.CircuitDialer over raw TCP connections.
type TCPDialer struct{}

// NewTCPDialer returns a CircuitDialer using TCP.
func NewTCPDialer() useSvc.CircuitDialer { return &TCPDialer{} }

func (TCPDialer) Dial(addr string) (net.Conn, error) { return net.Dial("tcp", addr) }

func (TCPDialer) SendCell(conn net.Conn, c entity.Cell) error {
	var hdr [20]byte
	copy(hdr[:16], c.CircID.Bytes())
	binary.BigEndian.PutUint16(hdr[16:18], c.StreamID.UInt16())
	if c.End {
		binary.BigEndian.PutUint16(hdr[18:20], 0xFFFF)
		_, err := conn.Write(hdr[:])
		return err
	}
	binary.BigEndian.PutUint16(hdr[18:20], uint16(len(c.Data)))
	if _, err := conn.Write(hdr[:]); err != nil {
		return err
	}
	_, err := conn.Write(c.Data)
	return err
}

func (TCPDialer) WaitCreated(conn net.Conn) ([]byte, error) {
	var hdr [20]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return nil, err
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

func (TCPDialer) SendDestroy(conn net.Conn, cid value_object.CircuitID) error {
	var buf [20]byte
	copy(buf[:16], cid.Bytes())
	buf[18] = 0xFE
	_, err := conn.Write(buf[:])
	return err
}
