package service

import (
	"net"
	"time"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// MemDialer is a no-op dialer for tests and demos.
type MemDialer struct{}

// NewMemDialer returns a CircuitDialer that performs no network I/O.
func NewMemDialer() useSvc.CircuitDialer { return &MemDialer{} }

func (MemDialer) Dial(string) (net.Conn, error)                      { return dummyConn{}, nil }
func (MemDialer) SendCell(net.Conn, entity.Cell) error               { return nil }
func (MemDialer) WaitAck(net.Conn) error                             { return nil }
func (MemDialer) SendDestroy(net.Conn, value_object.CircuitID) error { return nil }

// dummyConn implements net.Conn but does nothing.
type dummyConn struct{}

func (dummyConn) Read([]byte) (int, error)         { return 0, nil }
func (dummyConn) Write([]byte) (int, error)        { return 0, nil }
func (dummyConn) Close() error                     { return nil }
func (dummyConn) LocalAddr() net.Addr              { return nil }
func (dummyConn) RemoteAddr() net.Addr             { return nil }
func (dummyConn) SetDeadline(t time.Time) error    { return nil }
func (dummyConn) SetReadDeadline(time.Time) error  { return nil }
func (dummyConn) SetWriteDeadline(time.Time) error { return nil }
