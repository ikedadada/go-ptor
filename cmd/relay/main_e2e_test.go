package main

import (
	"bytes"
	"context"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"

	"github.com/google/uuid"
)

// safeBuffer wraps bytes.Buffer with mutex for concurrent access
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *safeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *safeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

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
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			c, err := net.Dial("tcp", addr)
			if err == nil {
				return c, nil
			}
		}
	}
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
	out := &safeBuffer{}
	cmd := exec.CommandContext(ctx, exe, "-listen", addr)
	cmd.Stderr = out
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
	payloadEncoder := service.NewPayloadEncodingService()
	inner, _ := payloadEncoder.EncodeDataPayload(&service.DataPayloadDTO{StreamID: sid, Data: data})
	cellBuf, _ := entity.Encode(entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: inner})
	outBuf := append(cid[:], cellBuf...)
	c.Write(outBuf)
	c.Close()

	// Wait for log output with timeout
	timeout := time.After(500 * time.Millisecond)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			cancel()
			cmd.Wait()
			t.Errorf("timeout waiting for log output with cid: %s", out.String())
			return
		case <-ticker.C:
			if strings.Contains(out.String(), cid.String()) {
				cancel()
				cmd.Wait()
				return // Success: found expected log output
			}
		}
	}
}
