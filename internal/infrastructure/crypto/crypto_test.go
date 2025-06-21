package crypto_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"ikedadada/go-ptor/internal/infrastructure/crypto"
)

func TestAESRoundTrip(t *testing.T) {
	var key [32]byte
	var nonce [12]byte
	if _, err := rand.Read(key[:]); err != nil {
		t.Fatalf("rand: %v", err)
	}
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatalf("rand: %v", err)
	}
	plain := []byte("hello")
	enc, err := crypto.AESSeal(key, nonce, plain)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	out, err := crypto.AESOpen(key, nonce, enc)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if string(out) != string(plain) {
		t.Fatalf("round-trip mismatch: %q", out)
	}
}

func TestRSARoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	msg := []byte("secret")
	enc, err := crypto.RSAEncrypt(&priv.PublicKey, msg)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	out, err := crypto.RSADecrypt(priv, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(out) != string(msg) {
		t.Fatalf("round-trip mismatch")
	}
}
