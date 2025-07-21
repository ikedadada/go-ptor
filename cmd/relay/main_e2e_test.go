package main

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ikedadada/go-ptor/shared/domain/aggregate"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"

	"github.com/google/uuid"
)

func freePort(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

func waitDial(addr string, d time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			return c, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, context.DeadlineExceeded
}

func buildBin(t *testing.T) string {
	exe := filepath.Join(t.TempDir(), "relay")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}
	return exe
}

func TestRelayMain_E2E(t *testing.T) {
	addr := freePort(t)
	exe := buildBin(t)
	ctx, cancel := context.WithCancel(context.Background())
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, exe, "-listen", addr)
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start relay: %v", err)
	}
	defer func() {
		cancel()
		cmd.Wait()
	}()

	c, err := waitDial(addr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial relay: %v", err)
	}

	cid := uuid.New()
	sid := uint16(1)
	data := []byte("ok")
	inner, _ := vo.EncodeDataPayload(&vo.DataPayload{StreamID: sid, Data: data})
	cellBuf, _ := entity.Encode(entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: inner})
	outBuf := append(cid[:], cellBuf...)
	c.Write(outBuf)
	c.Close()

	time.Sleep(100 * time.Millisecond)
	cancel()
	cmd.Wait()

	if !strings.Contains(out.String(), cid.String()) {
		t.Errorf("log missing cid: %s", out.String())
	}
}

func TestDefaultTTLFromEnv(t *testing.T) {
	os.Setenv("PTOR_TTL_SECONDS", "10")
	defer os.Unsetenv("PTOR_TTL_SECONDS")
	if got := defaultTTL(); got != 10*time.Second {
		t.Fatalf("expected 10s, got %v", got)
	}
}

// Additional tests from tcp_dialer_sendcell_test.go

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
	d := service.NewTCPCircuitBuildService()
	cid := vo.NewCircuitID()
	payload := []byte("hello")
	streamID, _ := vo.StreamIDFrom(0)
	cell, err := aggregate.NewRelayCell(vo.CmdExtend, cid, streamID, payload)
	if err != nil {
		t.Fatalf("NewRelayCell error: %v", err)
	}

	if err := d.SendExtendCell(conn, cell); err != nil {
		t.Fatalf("SendExtendCell error: %v", err)
	}

	if conn.Len() != 16+entity.MaxCellSize {
		t.Fatalf("expected %d bytes, got %d", 16+entity.MaxCellSize, conn.Len())
	}
}
