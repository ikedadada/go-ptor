package main

import (
	"bytes"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/handler"
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
	cell := entity.Cell{CircID: cid, Data: payload}

	if err := d.SendCell(conn, cell); err != nil {
		t.Fatalf("SendCell error: %v", err)
	}

	if conn.Len() != 16+value_object.MaxCellSize {
		t.Fatalf("expected %d bytes, got %d", 16+value_object.MaxCellSize, conn.Len())
	}

	gotCID, gotCell, err := handler.ReadCell(bytes.NewReader(conn.Bytes()))
	if err != nil {
		t.Fatalf("readCell: %v", err)
	}
	if !cid.Equal(gotCID) {
		t.Fatalf("cid mismatch")
	}
	if gotCell.Cmd != value_object.CmdExtend {
		t.Fatalf("cmd mismatch: %d", gotCell.Cmd)
	}
	if string(gotCell.Payload) != string(payload) {
		t.Fatalf("payload mismatch")
	}
}
