package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
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
	exe := filepath.Join(t.TempDir(), "hidden")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}
	return exe
}

func TestHiddenMain_E2E(t *testing.T) {
	relayAddr := freePort(t)
	// start a simple HTTP server the hidden service will proxy to
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	httpAddr := srv.Listener.Addr().String()
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

	req, _ := http.NewRequest("GET", "/", nil)
	req.Host = "example"
	req.Header.Set("Connection", "close")
	if err := req.Write(c); err != nil {
		t.Fatalf("write request: %v", err)
	}

	res, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(body) != "ok" {
		t.Errorf("unexpected body: %q", body)
	}
}
