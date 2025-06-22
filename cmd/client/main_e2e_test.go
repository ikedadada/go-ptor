package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/google/uuid"
	"ikedadada/go-ptor/internal/domain/entity"
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

func buildRelayBin(t *testing.T) string {
	exe := filepath.Join(t.TempDir(), "relay")
	cmd := exec.Command("go", "build", "-o", exe, "../relay")
	if err := cmd.Run(); err != nil {
		t.Fatalf("build relay: %v", err)
	}
	return exe
}

func TestClientMain_E2E(t *testing.T) {
	socks := freePort(t)
	relayAddr := freePort(t)

	relayExe := buildRelayBin(t)
	rctx, rcancel := context.WithCancel(context.Background())
	rcmd := exec.CommandContext(rctx, relayExe, "-listen", relayAddr)
	var rout bytes.Buffer
	rcmd.Stdout = &rout
	rcmd.Stderr = &rout
	if err := rcmd.Start(); err != nil {
		t.Fatalf("start relay: %v", err)
	}
	defer func() {
		rcancel()
		rcmd.Wait()
		t.Log("relay log:", rout.String())
	}()

	if _, err := waitDial(relayAddr, 5*time.Second); err != nil {
		t.Fatalf("dial relay: %v", err)
	}

	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("target listen: %v", err)
	}
	defer targetLn.Close()
	targetAddr := targetLn.Addr().(*net.TCPAddr)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}
	pem := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	dirData := entity.Directory{
		Relays: map[string]entity.RelayInfo{
			uuid.NewString(): {Endpoint: relayAddr, PubKey: pem},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/relays.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entity.Directory{Relays: dirData.Relays})
	})
	mux.HandleFunc("/hidden.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entity.Directory{HiddenServices: map[string]entity.HiddenServiceInfo{}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	exe := buildBin(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, exe, "-hops", "1", "-socks", socks, "-dir", srv.URL)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		t.Fatalf("start client: %v", err)
	}
	defer func() {
		cancel()
		cmd.Wait()
		t.Log(buf.String())
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
