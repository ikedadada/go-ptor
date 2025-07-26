package value_object_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestParsePrivateKeyFromPEM(t *testing.T) {
	// Test RSA private key parsing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	rsaPrivKey := vo.NewRSAPrivKey(rsaKey)
	rsaPEM := rsaPrivKey.ToPEM()

	parsedRSA, err := vo.ParsePrivateKeyFromPEM(rsaPEM)
	if err != nil {
		t.Fatalf("Failed to parse RSA PEM: %v", err)
	}
	if parsedRSA.KeyType() != "RSA" {
		t.Errorf("Expected RSA key type, got %s", parsedRSA.KeyType())
	}

	// Test Ed25519 private key parsing
	_, ed25519Key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	ed25519PrivKey := vo.NewEd25519PrivKey(ed25519Key)
	ed25519PEM := ed25519PrivKey.ToPEM()

	parsedEd25519, err := vo.ParsePrivateKeyFromPEM(ed25519PEM)
	if err != nil {
		t.Fatalf("Failed to parse Ed25519 PEM: %v", err)
	}
	if parsedEd25519.KeyType() != "Ed25519" {
		t.Errorf("Expected Ed25519 key type, got %s", parsedEd25519.KeyType())
	}

	// Test invalid PEM
	_, err = vo.ParsePrivateKeyFromPEM([]byte("invalid pem"))
	if err != vo.ErrNoPEMData {
		t.Errorf("Expected ErrNoPEMData, got %v", err)
	}

	// Test unsupported PEM block type
	unsupportedPEM := `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIH7N+RXGJqjQ9UqaRjYkR7YWJXvhSixPLZUGHjl/RjE5oAoGCCqGSM49
AwEHoUQDQgAEWnrzaFiRY4Nt9W7GnU9hNyPcOw8nOlZUVGQDLaFNL8BtZHBjMoYW
XYLYeRdXVDUaWCCsKEwNNaQHzD8mLJNRRw==
-----END EC PRIVATE KEY-----`
	_, err = vo.ParsePrivateKeyFromPEM([]byte(unsupportedPEM))
	if err != vo.ErrUnsupportedPEMBlock {
		t.Errorf("Expected ErrUnsupportedPEMBlock, got %v", err)
	}
}
