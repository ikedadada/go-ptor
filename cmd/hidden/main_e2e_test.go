package main

import (
	"context"
	"crypto/ed25519"
	"encoding/pem"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
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
	exe := filepath.Join(t.TempDir(), "hidden")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}
	return exe
}

func TestHiddenMain_E2E(t *testing.T) {
	relayAddr := freePort(t)
	httpAddr := freePort(t)
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key})
	f, err := os.CreateTemp(t.TempDir(), "key.pem")
	if err != nil {
		t.Fatalf("temp key: %v", err)
	}
	f.Write(pemBytes)
	f.Close()

	exe := buildBin(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, exe, "-key", f.Name(), "-listen", relayAddr, "-http", httpAddr)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start hidden: %v", err)
	}
	defer func() {
		cancel()
		cmd.Wait()
	}()

	c, err := waitDial(relayAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial hidden: %v", err)
	}
	defer c.Close()

	msg := []byte("ping")
	if _, err := c.Write(msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(c, buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf) != "ping" {
		t.Errorf("echo mismatch: %q", buf)
	}
}
