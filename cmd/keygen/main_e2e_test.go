package main

import (
	"encoding/pem"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestKeygenCommand(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "key.pem")

	cmd := exec.Command("go", "run", "./cmd/keygen", "-out", out)
	cmd.Dir = "../.."
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run command: %v, output: %s", err, b)
	}

	privData, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read priv: %v", err)
	}
	blk, _ := pem.Decode(privData)
	if blk == nil || blk.Type != "RSA PRIVATE KEY" {
		t.Errorf("invalid private key")
	}

	pubData, err := os.ReadFile(out + ".pub")
	if err != nil {
		t.Fatalf("read pub: %v", err)
	}
	blk, _ = pem.Decode(pubData)
	if blk == nil || blk.Type != "RSA PUBLIC KEY" {
		t.Errorf("invalid public key")
	}
}
