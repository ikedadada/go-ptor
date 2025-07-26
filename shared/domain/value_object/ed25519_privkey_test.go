package value_object_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestNewEd25519PrivKey(t *testing.T) {
	// Test with valid Ed25519 key
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	ed25519PrivKey := vo.NewEd25519PrivKey(privKey)
	if ed25519PrivKey == nil {
		t.Fatal("Expected non-nil Ed25519PrivKey")
	}

	// Test with nil key
	nilPrivKey := vo.NewEd25519PrivKey(nil)
	if nilPrivKey != nil {
		t.Error("Expected nil for nil input")
	}
}

func TestEd25519PrivKey_ToPEM(t *testing.T) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	ed25519PrivKey := vo.NewEd25519PrivKey(privKey)
	pem := ed25519PrivKey.ToPEM()

	if len(pem) == 0 {
		t.Error("Expected non-empty PEM data")
	}

	// Verify PEM starts with correct header
	pemStr := string(pem)
	if !strings.Contains(pemStr, "-----BEGIN PRIVATE KEY-----") {
		t.Error("PEM should start with PRIVATE KEY header")
	}
	if !strings.Contains(pemStr, "-----END PRIVATE KEY-----") {
		t.Error("PEM should end with PRIVATE KEY footer")
	}

	// Test with nil key - create object with nil key by setting internal field
	nilPrivKey := vo.NewEd25519PrivKey(nil)
	if nilPrivKey != nil {
		t.Error("NewEd25519PrivKey should return nil for nil key")
	}
}

func TestEd25519PrivKey_PublicKey(t *testing.T) {
	publicKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	ed25519PrivKey := vo.NewEd25519PrivKey(privKey)
	pubKey := ed25519PrivKey.PublicKey()

	if pubKey == nil {
		t.Fatal("Expected non-nil public key")
	}

	ed25519PubKey, ok := pubKey.(vo.Ed25519PubKey)
	if !ok {
		t.Fatalf("Expected Ed25519PubKey, got %T", pubKey)
	}

	// Compare the public keys
	if string(ed25519PubKey.PublicKey) != string(publicKey) {
		t.Error("Public key should match the Ed25519 key's public key")
	}

	// Test with nil key
	nilPrivKey := vo.NewEd25519PrivKey(nil)
	if nilPrivKey != nil {
		t.Error("Expected nil Ed25519PrivKey for nil key")
	}
}

func TestEd25519PrivKey_KeyType(t *testing.T) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	ed25519PrivKey := vo.NewEd25519PrivKey(privKey)
	keyType := ed25519PrivKey.KeyType()

	if keyType != "Ed25519" {
		t.Errorf("Expected key type 'Ed25519', got '%s'", keyType)
	}
}

func TestEd25519PrivKey_Ed25519Key(t *testing.T) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	ed25519PrivKey := vo.NewEd25519PrivKey(privKey)
	retrievedKey := ed25519PrivKey.Ed25519Key()

	if string(retrievedKey) != string(privKey) {
		t.Error("Ed25519Key should return the original Ed25519 key")
	}

	// Test with nil key
	nilPrivKey := vo.NewEd25519PrivKey(nil)
	if nilPrivKey != nil {
		t.Error("Expected nil Ed25519PrivKey for nil key")
	}
}

func TestEd25519PrivKeyFromPEM(t *testing.T) {
	// Generate a test Ed25519 key
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	// Create Ed25519PrivKey and get PEM
	originalPrivKey := vo.NewEd25519PrivKey(privKey)
	pem := originalPrivKey.ToPEM()

	// Parse the PEM back
	parsedPrivKey, err := vo.Ed25519PrivKeyFromPEM(pem)
	if err != nil {
		t.Fatalf("Failed to parse PEM: %v", err)
	}

	if parsedPrivKey == nil {
		t.Fatal("Expected non-nil parsed private key")
	}

	if parsedPrivKey.KeyType() != "Ed25519" {
		t.Errorf("Expected key type 'Ed25519', got '%s'", parsedPrivKey.KeyType())
	}

	// Compare the keys
	if string(parsedPrivKey.Ed25519Key()) != string(privKey) {
		t.Error("Parsed key should match original key")
	}

	// Test invalid PEM
	_, err = vo.Ed25519PrivKeyFromPEM([]byte("invalid pem"))
	if err != vo.ErrNoPEMData {
		t.Errorf("Expected ErrNoPEMData, got %v", err)
	}
}

func TestEd25519PrivKey_Interface(t *testing.T) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	ed25519PrivKey := vo.NewEd25519PrivKey(privKey)

	// Test that it implements PrivateKey interface
	var _ vo.PrivateKey = ed25519PrivKey

	// Test interface methods
	if ed25519PrivKey.KeyType() != "Ed25519" {
		t.Error("Interface method KeyType() failed")
	}

	if ed25519PrivKey.PublicKey() == nil {
		t.Error("Interface method PublicKey() failed")
	}

	if ed25519PrivKey.ToPEM() == nil {
		t.Error("Interface method ToPEM() failed")
	}
}
