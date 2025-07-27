package service_test

import (
	"net"
	"testing"
	"time"

	"github.com/google/uuid"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func startTestTCPServer(t *testing.T) (addr string, received chan []byte, closeFn func()) {
	t.Helper()
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

func TestTCPCircuitMessagingService_TransmitData_TerminateStream_realConn(t *testing.T) {
	addr, received, closeFn := startTestTCPServer(t)
	defer closeFn()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	peSvc := service.NewPayloadEncodingService()
	tx := service.NewTCPCircuitMessagingService(conn, peSvc)
	cid := vo.NewCircuitID()
	sid := vo.NewStreamIDAuto()
	data := []byte("hello")

	err = tx.TransmitData(cid, sid, data)
	if err != nil {
		t.Fatalf("TransmitData error: %v", err)
	}
	select {
	case msg := <-received:
		if len(msg) != 16+entity.MaxCellSize {
			t.Fatalf("unexpected cell size %d", len(msg))
		}
		var cidBuf [16]byte
		copy(cidBuf[:], msg[:16])
		var u uuid.UUID
		copy(u[:], cidBuf[:])
		gotCID, err := vo.CircuitIDFrom(u.String())
		if err != nil {
			t.Fatalf("cid parse: %v", err)
		}
		if !gotCID.Equal(cid) {
			t.Errorf("cid mismatch")
		}
		cell, err := entity.Decode(msg[16:])
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if cell.Cmd != vo.CmdData {
			t.Errorf("unexpected cmd %d", cell.Cmd)
		}
		p, err := peSvc.DecodeDataPayload(cell.Payload)
		if err != nil {
			t.Fatalf("payload: %v", err)
		}
		if string(p.Data) != string(data) || p.StreamID != sid.UInt16() {
			t.Errorf("payload mismatch")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for TransmitData")
	}

	err = tx.InitiateStream(cid, sid, data)
	if err != nil {
		t.Fatalf("InitiateStream error: %v", err)
	}
	select {
	case msg := <-received:
		if len(msg) != 16+entity.MaxCellSize {
			t.Fatalf("unexpected cell size %d", len(msg))
		}
		cell, err := entity.Decode(msg[16:])
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if cell.Cmd != vo.CmdBegin {
			t.Errorf("unexpected cmd %d", cell.Cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for InitiateStream")
	}

	err = tx.EstablishConnection(cid, []byte("target"))
	if err != nil {
		t.Fatalf("EstablishConnection error: %v", err)
	}
	select {
	case msg := <-received:
		if len(msg) != 16+entity.MaxCellSize {
			t.Fatalf("unexpected cell size %d", len(msg))
		}
		cell, err := entity.Decode(msg[16:])
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if cell.Cmd != vo.CmdConnect {
			t.Errorf("expected CONNECT cmd, got %d", cell.Cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for EstablishConnection")
	}

	err = tx.TerminateStream(cid, sid)
	if err != nil {
		t.Fatalf("TerminateStream error: %v", err)
	}
	select {
	case msg := <-received:
		if len(msg) != 16+entity.MaxCellSize {
			t.Fatalf("unexpected cell size %d", len(msg))
		}
		cell, err := entity.Decode(msg[16:])
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if cell.Cmd != vo.CmdEnd {
			t.Errorf("expected END cmd, got %d", cell.Cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for TerminateStream")
	}

	err = tx.DestroyCircuit(cid)
	if err != nil {
		t.Fatalf("DestroyCircuit error: %v", err)
	}
	select {
	case msg := <-received:
		if len(msg) != 16+entity.MaxCellSize {
			t.Fatalf("unexpected cell size %d", len(msg))
		}
		cell, err := entity.Decode(msg[16:])
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if cell.Cmd != vo.CmdDestroy {
			t.Errorf("expected DESTROY cmd, got %d", cell.Cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for DestroyCircuit")
	}
}

func TestTCPCircuitMessagingService_TransmitData_tooBig_realConn(t *testing.T) {
	addr, _, closeFn := startTestTCPServer(t)
	defer closeFn()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	peSvc := service.NewPayloadEncodingService()
	tx := service.NewTCPCircuitMessagingService(conn, peSvc)
	cid := vo.NewCircuitID()
	sid := vo.NewStreamIDAuto()
	big := make([]byte, entity.MaxPayloadSize+1)
	err = tx.TransmitData(cid, sid, big)
	if err == nil || err.Error() != "data too big" {
		t.Errorf("expected data too big error, got %v", err)
	}
}
