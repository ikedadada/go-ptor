package service_test

import (
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/service"
)

func startTestTCPServer(t *testing.T) (addr string, received chan []byte, closeFn func()) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	received = make(chan []byte, 10)
	stop := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if n > 0 {
				received <- buf[:n]
			}
			if err != nil {
				return
			}
			select {
			case <-stop:
				return
			default:
			}
		}
	}()
	return ln.Addr().String(), received, func() { close(stop); ln.Close() }
}

func TestTCPTransmitter_SendData_SendEnd_realConn(t *testing.T) {
	addr, received, closeFn := startTestTCPServer(t)
	defer closeFn()

	tx, err := service.NewTCPTransmitter(addr)
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	cid := value_object.NewCircuitID()
	sid := value_object.NewStreamIDAuto()
	data := []byte("hello")

	err = tx.SendData(cid, sid, data)
	if err != nil {
		t.Fatalf("SendData error: %v", err)
	}
	select {
	case msg := <-received:
		if len(msg) != value_object.MaxCellSize {
			t.Fatalf("unexpected cell size %d", len(msg))
		}
		cell, err := value_object.Decode(msg)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if cell.Cmd != value_object.CmdData {
			t.Errorf("unexpected cmd %d", cell.Cmd)
		}
		p, err := value_object.DecodeDataPayload(cell.Payload)
		if err != nil {
			t.Fatalf("payload: %v", err)
		}
		if string(p.Data) != string(data) || p.StreamID != sid.UInt16() {
			t.Errorf("payload mismatch")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for SendData")
	}

	err = tx.SendBegin(cid, sid, data)
	if err != nil {
		t.Fatalf("SendBegin error: %v", err)
	}
	select {
	case msg := <-received:
		cell, err := value_object.Decode(msg)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if cell.Cmd != value_object.CmdBegin {
			t.Errorf("unexpected cmd %d", cell.Cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for SendBegin")
	}

	err = tx.SendEnd(cid, sid)
	if err != nil {
		t.Fatalf("SendEnd error: %v", err)
	}
	select {
	case msg := <-received:
		cell, err := value_object.Decode(msg)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if cell.Cmd != value_object.CmdEnd {
			t.Errorf("expected END cmd, got %d", cell.Cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for SendEnd")
	}

	err = tx.SendDestroy(cid)
	if err != nil {
		t.Fatalf("SendDestroy error: %v", err)
	}
	select {
	case msg := <-received:
		cell, err := value_object.Decode(msg)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if cell.Cmd != value_object.CmdDestroy {
			t.Errorf("expected DESTROY cmd, got %d", cell.Cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for SendDestroy")
	}
}

func TestTCPTransmitter_SendData_tooBig_realConn(t *testing.T) {
	addr, _, closeFn := startTestTCPServer(t)
	defer closeFn()
	tx, err := service.NewTCPTransmitter(addr)
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	cid := value_object.NewCircuitID()
	sid := value_object.NewStreamIDAuto()
	big := make([]byte, value_object.MaxPayloadSize+1)
	err = tx.SendData(cid, sid, big)
	if err == nil || err.Error() != "data too big" {
		t.Errorf("expected data too big error, got %v", err)
	}
}
