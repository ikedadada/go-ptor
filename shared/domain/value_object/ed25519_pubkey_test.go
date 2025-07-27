package value_object

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestEd25519PubKeyFromPEM(t *testing.T) {
	// Generate a test Ed25519 key pair
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)

	// Marshal to PKIX format and encode as PEM
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	// Test parsing from PEM
	ed25519PubKey, err := Ed25519PubKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("Ed25519PubKeyFromPEM failed: %v", err)
	}

	// Verify the parsed key matches the original
	if !ed25519.PublicKey(ed25519PubKey.PublicKey).Equal(pubKey) {
		t.Error("Parsed public key does not match original")
	}
}

func TestEd25519PubKeyFromPEM_NoPEMData(t *testing.T) {
	// Test with invalid PEM data
	invalidPEM := []byte("not pem data")

	_, err := Ed25519PubKeyFromPEM(invalidPEM)
	if err == nil {
		t.Error("Expected error for invalid PEM data")
	}
	if err.Error() != "no PEM data" {
		t.Errorf("Expected 'no PEM data' error, got: %v", err)
	}
}

func TestEd25519PubKeyFromPEM_InvalidPKIX(t *testing.T) {
	// Create PEM with invalid PKIX data
	invalidPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: []byte("invalid pkix data"),
	})

	_, err := Ed25519PubKeyFromPEM(invalidPEM)
	if err == nil {
		t.Error("Expected error for invalid PKIX data")
	}
}

func TestEd25519PubKeyFromPEM_NotEd25519Key(t *testing.T) {
	// Create a PEM with minimal ASN.1 sequence that will parse but isn't Ed25519
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: []byte{0x30, 0x00}, // Minimal ASN.1 sequence that parses but isn't Ed25519
	})

	_, err := Ed25519PubKeyFromPEM(pemData)
	if err == nil {
		t.Error("Expected error for non-Ed25519 key")
	}
}

func TestEd25519PubKey_ToPEM(t *testing.T) {
	// Generate a test Ed25519 key pair
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)

	// Create Ed25519PubKey
	ed25519PubKey := Ed25519PubKey{PublicKey: pubKey}

	// Convert to PEM
	pemData := ed25519PubKey.ToPEM()

	// Verify the PEM data is valid
	block, _ := pem.Decode(pemData)
	if block == nil {
		t.Fatal("Failed to decode PEM data")
	}
	if block.Type != "PUBLIC KEY" {
		t.Errorf("Expected PEM type 'PUBLIC KEY', got '%s'", block.Type)
	}

	// Parse the PKIX data
	parsedPubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse PKIX public key: %v", err)
	}

	// Verify it's an Ed25519 key and matches the original
	ed25519Parsed, ok := parsedPubKey.(ed25519.PublicKey)
	if !ok {
		t.Fatal("Parsed key is not Ed25519")
	}
	if !ed25519Parsed.Equal(pubKey) {
		t.Error("Parsed key does not match original")
	}
}

func TestEd25519PubKey_RoundTrip(t *testing.T) {
	// Generate a test Ed25519 key pair
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)

	// Create Ed25519PubKey and convert to PEM
	original := Ed25519PubKey{PublicKey: pubKey}
	pemData := original.ToPEM()

	// Parse back from PEM
	parsed, err := Ed25519PubKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("Failed to parse PEM: %v", err)
	}

	// Verify round-trip integrity
	if !ed25519.PublicKey(parsed.PublicKey).Equal(ed25519.PublicKey(original.PublicKey)) {
		t.Error("Round-trip failed: keys do not match")
	}
}

func TestEd25519PubKey_PublicKeyInterface(t *testing.T) {
	// Verify that Ed25519PubKey implements PublicKey interface
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)

	ed25519PubKey := Ed25519PubKey{PublicKey: pubKey}

	// Test that it can be assigned to PublicKey interface
	var pubKeyInterface PublicKey = ed25519PubKey
	if pubKeyInterface == nil {
		t.Error("Ed25519PubKey should implement PublicKey interface")
	}
}

func TestEd25519PubKey_ZeroValue(t *testing.T) {
	// Test with zero value
	var zeroPubKey Ed25519PubKey

	// ToPEM should not panic with zero value
	pemData := zeroPubKey.ToPEM()
	if pemData == nil {
		t.Error("ToPEM should return data even for zero value")
	}

	// Verify it creates valid PEM structure
	block, _ := pem.Decode(pemData)
	if block == nil {
		t.Error("Zero value should still produce valid PEM structure")
	}
	if block.Type != "PUBLIC KEY" {
		t.Errorf("Expected PEM type 'PUBLIC KEY', got '%s'", block.Type)
	}
}

func TestEd25519PubKey_EmptyPEMBlock(t *testing.T) {
	// Test with empty PEM block
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: []byte{},
	})

	_, err := Ed25519PubKeyFromPEM(pemData)
	if err == nil {
		t.Error("Expected error for empty PEM block")
	}
}

func TestEd25519PubKey_MultipleOperations(t *testing.T) {
	// Test multiple operations on the same key
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)

	ed25519PubKey := Ed25519PubKey{PublicKey: pubKey}

	// Multiple ToPEM calls should return the same result
	pem1 := ed25519PubKey.ToPEM()
	pem2 := ed25519PubKey.ToPEM()

	if string(pem1) != string(pem2) {
		t.Error("Multiple ToPEM calls should return identical results")
	}

	// Multiple parsing of the same PEM should work
	parsed1, err1 := Ed25519PubKeyFromPEM(pem1)
	parsed2, err2 := Ed25519PubKeyFromPEM(pem1)

	if err1 != nil || err2 != nil {
		t.Fatalf("Parsing failed: %v, %v", err1, err2)
	}

	if !ed25519.PublicKey(parsed1.PublicKey).Equal(ed25519.PublicKey(parsed2.PublicKey)) {
		t.Error("Multiple parsing operations should return identical keys")
	}
}
