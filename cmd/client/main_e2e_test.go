package main

import (
	"bufio"
	"context"
	"io"
	"net"
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
	exe := filepath.Join(t.TempDir(), "client")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}
	return exe
}

func TestClientMain_E2E(t *testing.T) {
	socks := freePort(t)
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("target listen: %v", err)
	}
	defer targetLn.Close()
	targetAddr := targetLn.Addr().(*net.TCPAddr)

	exe := buildBin(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, exe, "-hops", "1", "-socks", socks)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start client: %v", err)
	}
	defer func() {
		cancel()
		cmd.Wait()
	}()

	c, err := waitDial(socks, 5*time.Second)
	if err != nil {
		t.Fatalf("dial socks: %v", err)
	}
	defer c.Close()

	go func() {
		conn, err := targetLn.Accept()
		if err == nil {
			conn.Close()
		}
	}()

	w := bufio.NewWriter(c)
	r := bufio.NewReader(c)
	w.Write([]byte{5, 1, 0})
	w.Flush()
	io.ReadFull(r, make([]byte, 2))

	ip := targetAddr.IP.To4()
	req := []byte{5, 1, 0, 1}
	req = append(req, ip...)
	req = append(req, byte(targetAddr.Port>>8), byte(targetAddr.Port))
	w.Write(req)
	w.Flush()
	io.ReadFull(r, make([]byte, 10))
}
