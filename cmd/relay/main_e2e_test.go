package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	buf := new(bytes.Buffer)
	buf.Write(cid[:])
	binary.Write(buf, binary.BigEndian, sid)
	binary.Write(buf, binary.BigEndian, uint16(len(data)))
	buf.Write(data)
	c.Write(buf.Bytes())
	c.Close()

	time.Sleep(100 * time.Millisecond)
	cancel()
	cmd.Wait()

	if !strings.Contains(out.String(), cid.String()) {
		t.Errorf("log missing cid: %s", out.String())
	}
}
