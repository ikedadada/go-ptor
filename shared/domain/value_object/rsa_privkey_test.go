package value_object_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestNewRSAPrivKey(t *testing.T) {
	// Test with valid RSA key
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	privKey := vo.NewRSAPrivKey(rsaKey)
	if privKey == nil {
		t.Fatal("Expected non-nil RSAPrivKey")
	}

	// Test with nil key
	nilPrivKey := vo.NewRSAPrivKey(nil)
	if nilPrivKey != nil {
		t.Error("Expected nil for nil input")
	}
}

func TestRSAPrivKey_ToPEM(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	privKey := vo.NewRSAPrivKey(rsaKey)
	pem := privKey.ToPEM()

	if len(pem) == 0 {
		t.Error("Expected non-empty PEM data")
	}

	// Verify PEM starts with correct header
	pemStr := string(pem)
	if !containsString(pemStr, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Error("PEM should start with RSA PRIVATE KEY header")
	}
	if !containsString(pemStr, "-----END RSA PRIVATE KEY-----") {
		t.Error("PEM should end with RSA PRIVATE KEY footer")
	}

	// Test with nil key - NewRSAPrivKey returns nil for nil input
	nilPrivKey := vo.NewRSAPrivKey(nil)
	if nilPrivKey != nil {
		t.Error("Expected nil RSAPrivKey for nil key")
	}
}

func TestRSAPrivKey_PublicKey(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	privKey := vo.NewRSAPrivKey(rsaKey)
	pubKey := privKey.PublicKey()

	if pubKey == nil {
		t.Fatal("Expected non-nil public key")
	}

	rsaPubKey, ok := pubKey.(vo.RSAPubKey)
	if !ok {
		t.Fatalf("Expected RSAPubKey, got %T", pubKey)
	}

	if rsaPubKey.PublicKey != &rsaKey.PublicKey {
		t.Error("Public key should match the RSA key's public key")
	}

	// Test with nil key - NewRSAPrivKey returns nil for nil input
	nilPrivKey := vo.NewRSAPrivKey(nil)
	if nilPrivKey != nil {
		t.Error("Expected nil RSAPrivKey for nil key")
	}
}

func TestRSAPrivKey_KeyType(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	privKey := vo.NewRSAPrivKey(rsaKey)
	keyType := privKey.KeyType()

	if keyType != "RSA" {
		t.Errorf("Expected key type 'RSA', got '%s'", keyType)
	}
}

func TestRSAPrivKey_RSAKey(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	privKey := vo.NewRSAPrivKey(rsaKey)
	retrievedKey := privKey.RSAKey()

	if retrievedKey != rsaKey {
		t.Error("RSAKey should return the original RSA key")
	}

	// Test with nil key - NewRSAPrivKey returns nil for nil input
	nilPrivKey := vo.NewRSAPrivKey(nil)
	if nilPrivKey != nil {
		t.Error("Expected nil RSAPrivKey for nil key")
	}
}

func TestRSAPrivKeyFromPEM(t *testing.T) {
	// Generate a test RSA key
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create RSAPrivKey and get PEM
	originalPrivKey := vo.NewRSAPrivKey(rsaKey)
	pem := originalPrivKey.ToPEM()

	// Parse the PEM back
	parsedPrivKey, err := vo.RSAPrivKeyFromPEM(pem)
	if err != nil {
		t.Fatalf("Failed to parse PEM: %v", err)
	}

	if parsedPrivKey == nil {
		t.Fatal("Expected non-nil parsed private key")
	}

	if parsedPrivKey.KeyType() != "RSA" {
		t.Errorf("Expected key type 'RSA', got '%s'", parsedPrivKey.KeyType())
	}

	// Test invalid PEM
	_, err = vo.RSAPrivKeyFromPEM([]byte("invalid pem"))
	if err != vo.ErrNoPEMData {
		t.Errorf("Expected ErrNoPEMData, got %v", err)
	}
}

func TestRSAPrivKey_Interface(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	privKey := vo.NewRSAPrivKey(rsaKey)

	// Test that it implements PrivateKey interface
	var _ vo.PrivateKey = privKey

	// Test interface methods
	if privKey.KeyType() != "RSA" {
		t.Error("Interface method KeyType() failed")
	}

	if privKey.PublicKey() == nil {
		t.Error("Interface method PublicKey() failed")
	}

	if privKey.ToPEM() == nil {
		t.Error("Interface method ToPEM() failed")
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsString(s[1:], substr)))
}
