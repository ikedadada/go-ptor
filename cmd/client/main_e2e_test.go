package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/google/uuid"
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"strings"
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

func buildHiddenBin(t *testing.T) string {
	exe := filepath.Join(t.TempDir(), "hidden")
	cmd := exec.Command("go", "build", "-o", exe, "../hidden")
	if err := cmd.Run(); err != nil {
		t.Fatalf("build hidden: %v", err)
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

	targetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer targetSrv.Close()
	targetAddr := targetSrv.Listener.Addr().(*net.TCPAddr)

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

	fmt.Fprintf(w, "GET / HTTP/1.0\r\nHost: %s\r\n\r\n", targetAddr.IP.String())
	w.Flush()
	resp, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestClientMain_HiddenService(t *testing.T) {
	socks := freePort(t)
	relayAddr := freePort(t)
	relay2Addr := freePort(t)
	hiddenAddr := freePort(t)

	hiddenExe := buildHiddenBin(t)
	// simple HTTP server the hidden service will proxy to
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key})
	f, err := os.CreateTemp(t.TempDir(), "key.pem")
	if err != nil {
		t.Fatalf("temp key: %v", err)
	}
	f.Write(pemBytes)
	f.Close()

	hctx, hcancel := context.WithCancel(context.Background())
	hcmd := exec.CommandContext(hctx, hiddenExe, "-key", f.Name(), "-listen", hiddenAddr, "-http", srv.Listener.Addr().String())
	if err := hcmd.Start(); err != nil {
		t.Fatalf("start hidden: %v", err)
	}
	defer func() {
		hcancel()
		hcmd.Wait()
	}()

	if _, err := waitDial(hiddenAddr, 5*time.Second); err != nil {
		t.Fatalf("dial hidden: %v", err)
	}

	relayExe := buildRelayBin(t)
	rctx, rcancel := context.WithCancel(context.Background())
	rcmd := exec.CommandContext(rctx, relayExe, "-listen", relayAddr)
	rcmd.Env = append(os.Environ(), "PTOR_HIDDEN_ADDR="+hiddenAddr)
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

	rctx2, cancel2 := context.WithCancel(context.Background())
	rcmd2 := exec.CommandContext(rctx2, relayExe, "-listen", relay2Addr)
	var rout2 bytes.Buffer
	rcmd2.Stdout = &rout2
	rcmd2.Stderr = &rout2
	if err := rcmd2.Start(); err != nil {
		t.Fatalf("start relay2: %v", err)
	}
	defer func() {
		cancel2()
		rcmd2.Wait()
		t.Log("relay2 log:", rout2.String())
	}()
	if _, err := waitDial(relay2Addr, 5*time.Second); err != nil {
		t.Fatalf("dial relay2: %v", err)
	}

	der, _ := x509.MarshalPKIXPublicKey(key.Public())
	hidPem := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	relKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	relDer, _ := x509.MarshalPKIXPublicKey(&relKey.PublicKey)
	relPem := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: relDer}))
	hidAddr := value_object.NewHiddenAddr(key.Public().(ed25519.PublicKey)).String()
	exitID := uuid.NewString()
	midID := uuid.NewString()
	dirData := entity.Directory{
		Relays: map[string]entity.RelayInfo{
			midID:  {Endpoint: relay2Addr, PubKey: relPem},
			exitID: {Endpoint: relayAddr, PubKey: relPem},
		},
		HiddenServices: map[string]entity.HiddenServiceInfo{
			hidAddr: {Relay: exitID, PubKey: hidPem},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/relays.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entity.Directory{Relays: dirData.Relays})
	})
	mux.HandleFunc("/hidden.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entity.Directory{HiddenServices: dirData.HiddenServices})
	})
	dirSrv := httptest.NewServer(mux)
	defer dirSrv.Close()

	exe := buildBin(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, exe, "-hops", "2", "-socks", socks, "-dir", dirSrv.URL)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start client: %v", err)
	}
	defer func() {
		cancel()
		cmd.Wait()
	}()

	if _, err := waitDial(socks, 5*time.Second); err != nil {
		t.Fatalf("dial socks: %v", err)
	}

	curl := exec.Command("curl", "--socks5-hostname", socks, "http://"+hidAddr)
	out, err := curl.CombinedOutput()
	if err != nil {
		t.Skipf("curl failed: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("ok")) {
		t.Fatalf("unexpected response: %s", out)
	}
	if !strings.Contains(rout.String(), "cmd=2") {
		t.Fatalf("exit relay did not see CONNECT")
	}
	if !strings.Contains(rout2.String(), "cmd=2") {
		t.Fatalf("middle relay did not receive CONNECT")
	}
}
