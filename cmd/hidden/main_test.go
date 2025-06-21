package main

import (
	"crypto/ed25519"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLoadEDPriv(t *testing.T) {
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	b := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key})
	f, err := os.CreateTemp(t.TempDir(), "key.pem")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer f.Close()
	f.Write(b)
	f.Close()

	got, err := loadEDPriv(f.Name())
	if err != nil {
		t.Fatalf("loadEDPriv error: %v", err)
	}
	if len(got) != ed25519.PrivateKeySize {
		t.Errorf("unexpected key size")
	}
}

func TestDemoMux(t *testing.T) {
	srv := httptest.NewServer(demoMux())
	defer srv.Close()
	res, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	defer res.Body.Close()
	buf := make([]byte, 32)
	n, _ := res.Body.Read(buf)
	if string(buf[:n]) != "hello from hidden service" {
		t.Errorf("unexpected body: %q", buf[:n])
	}
}
