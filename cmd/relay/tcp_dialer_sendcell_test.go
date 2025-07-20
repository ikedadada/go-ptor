package main

import (
	"bytes"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/domain/aggregate"
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/service"
)

type recordConn struct {
	bytes.Buffer
}

func (r *recordConn) Read(b []byte) (int, error)         { return r.Buffer.Read(b) }
func (r *recordConn) Write(b []byte) (int, error)        { return r.Buffer.Write(b) }
func (r *recordConn) Close() error                       { return nil }
func (r *recordConn) LocalAddr() net.Addr                { return nil }
func (r *recordConn) RemoteAddr() net.Addr               { return nil }
func (r *recordConn) SetDeadline(t time.Time) error      { return nil }
func (r *recordConn) SetReadDeadline(t time.Time) error  { return nil }
func (r *recordConn) SetWriteDeadline(t time.Time) error { return nil }

func TestSendCellWritesFixedPacket(t *testing.T) {
	conn := &recordConn{}
	d := service.NewTCPDialer()
	cid := value_object.NewCircuitID()
	payload := []byte("hello")
	streamID, _ := value_object.StreamIDFrom(0)
	cell, err := aggregate.NewRelayCell(value_object.CmdExtend, cid, streamID, payload)
	if err != nil {
		t.Fatalf("NewRelayCell error: %v", err)
	}

	if err := d.SendCell(conn, cell); err != nil {
		t.Fatalf("SendCell error: %v", err)
	}

	if conn.Len() != 16+entity.MaxCellSize {
		t.Fatalf("expected %d bytes, got %d", 16+entity.MaxCellSize, conn.Len())
	}
}
