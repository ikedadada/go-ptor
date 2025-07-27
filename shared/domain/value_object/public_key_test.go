package value_object

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestParsePublicKeyFromPEM_Ed25519(t *testing.T) {
	// Generate Ed25519 key pair
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)

	// Marshal to PKIX and create PEM
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	// Test parsing
	parsedKey, err := ParsePublicKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("ParsePublicKeyFromPEM failed: %v", err)
	}

	// Verify type and content
	ed25519Key, ok := parsedKey.(Ed25519PubKey)
	if !ok {
		t.Fatalf("Expected Ed25519PubKey, got %T", parsedKey)
	}

	if !ed25519.PublicKey(ed25519Key.PublicKey).Equal(pubKey) {
		t.Error("Parsed Ed25519 key does not match original")
	}
}

func TestParsePublicKeyFromPEM_RSA(t *testing.T) {
	// Generate RSA key pair
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pubKey := &privKey.PublicKey

	// Marshal to PKIX and create PEM
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	// Test parsing
	parsedKey, err := ParsePublicKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("ParsePublicKeyFromPEM failed: %v", err)
	}

	// Verify type and content
	rsaKey, ok := parsedKey.(RSAPubKey)
	if !ok {
		t.Fatalf("Expected RSAPubKey, got %T", parsedKey)
	}

	if rsaKey.PublicKey.N.Cmp(pubKey.N) != 0 || rsaKey.PublicKey.E != pubKey.E {
		t.Error("Parsed RSA key does not match original")
	}
}

func TestParsePublicKeyFromPEM_NoPEMData(t *testing.T) {
	invalidPEM := []byte("not a pem data")

	_, err := ParsePublicKeyFromPEM(invalidPEM)
	if err == nil {
		t.Error("Expected error for invalid PEM data")
	}
	if err.Error() != "no PEM data" {
		t.Errorf("Expected 'no PEM data' error, got: %v", err)
	}
}

func TestParsePublicKeyFromPEM_InvalidPKIX(t *testing.T) {
	// Create PEM with invalid PKIX data
	invalidPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: []byte("invalid pkix data"),
	})

	_, err := ParsePublicKeyFromPEM(invalidPEM)
	if err == nil {
		t.Error("Expected error for invalid PKIX data")
	}
}

func TestParsePublicKeyFromPEM_UnsupportedKeyType(t *testing.T) {
	// Create a minimal ASN.1 structure that parses but isn't RSA or Ed25519
	// This is a minimal valid ASN.1 sequence that x509.ParsePKIXPublicKey will accept
	// but will return an unsupported key type
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: []byte{0x30, 0x00}, // Empty ASN.1 sequence
	})

	_, err := ParsePublicKeyFromPEM(pemData)
	if err == nil {
		t.Error("Expected error for unsupported key type")
	}
}

func TestParsePublicKeyFromPEM_EmptyPEMBlock(t *testing.T) {
	// Create PEM with empty data
	emptyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: []byte{},
	})

	_, err := ParsePublicKeyFromPEM(emptyPEM)
	if err == nil {
		t.Error("Expected error for empty PEM block")
	}
}

func TestParsePublicKeyFromPEM_RoundTripEd25519(t *testing.T) {
	// Generate Ed25519 key
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	original := Ed25519PubKey{PublicKey: privKey.Public().(ed25519.PublicKey)}

	// Convert to PEM
	pemData := original.ToPEM()

	// Parse back from PEM
	parsed, err := ParsePublicKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("Failed to parse PEM: %v", err)
	}

	// Verify round-trip integrity
	ed25519Parsed, ok := parsed.(Ed25519PubKey)
	if !ok {
		t.Fatalf("Expected Ed25519PubKey, got %T", parsed)
	}

	if !ed25519.PublicKey(ed25519Parsed.PublicKey).Equal(ed25519.PublicKey(original.PublicKey)) {
		t.Error("Round-trip failed: keys do not match")
	}
}

func TestParsePublicKeyFromPEM_RoundTripRSA(t *testing.T) {
	// Generate RSA key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pubKey := &privKey.PublicKey

	// Marshal to PKIX (not PKCS1) for compatibility with ParsePublicKeyFromPEM
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	// Parse back from PEM
	parsed, err := ParsePublicKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("Failed to parse PEM: %v", err)
	}

	// Verify round-trip integrity
	rsaParsed, ok := parsed.(RSAPubKey)
	if !ok {
		t.Fatalf("Expected RSAPubKey, got %T", parsed)
	}

	if rsaParsed.PublicKey.N.Cmp(pubKey.N) != 0 || rsaParsed.PublicKey.E != pubKey.E {
		t.Error("Round-trip failed: keys do not match")
	}
}

func TestParsePublicKeyFromPEM_Interface(t *testing.T) {
	// Test that parsed keys implement PublicKey interface
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	ed25519Key := Ed25519PubKey{PublicKey: privKey.Public().(ed25519.PublicKey)}
	pemData := ed25519Key.ToPEM()

	parsed, err := ParsePublicKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("Failed to parse PEM: %v", err)
	}

	// Test interface assignment
	var pubKeyInterface PublicKey = parsed
	if pubKeyInterface == nil {
		t.Error("Parsed key should implement PublicKey interface")
	}

	// Test ToPEM method through interface
	interfacePEM := pubKeyInterface.ToPEM()
	if len(interfacePEM) == 0 {
		t.Error("ToPEM through interface should return data")
	}
}

func TestParsePublicKeyFromPEM_DifferentPEMTypes(t *testing.T) {
	// Generate Ed25519 key
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)

	// Test with PKIX formatted public key (standard format)
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}

	// Note: ParsePublicKeyFromPEM doesn't check PEM block type, only the content
	// It accepts any block type as long as the content is valid PKIX
	tests := []struct {
		name      string
		blockType string
	}{
		{"Standard PUBLIC KEY", "PUBLIC KEY"},
		{"RSA PUBLIC KEY format", "RSA PUBLIC KEY"},
		{"CERTIFICATE format", "CERTIFICATE"},
		{"PRIVATE KEY format", "PRIVATE KEY"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pemData := pem.EncodeToMemory(&pem.Block{
				Type:  test.blockType,
				Bytes: pubKeyBytes,
			})

			parsed, err := ParsePublicKeyFromPEM(pemData)
			if err != nil {
				t.Errorf("ParsePublicKeyFromPEM failed for block type %q: %v", test.blockType, err)
			}

			// Verify we got the correct key type
			if _, ok := parsed.(Ed25519PubKey); !ok {
				t.Errorf("Expected Ed25519PubKey for block type %q, got %T", test.blockType, parsed)
			}
		})
	}
}

func TestParsePublicKeyFromPEM_MultipleBlocks(t *testing.T) {
	// Generate two different keys
	_, privKey1, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate first Ed25519 key: %v", err)
	}
	_, privKey2, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate second Ed25519 key: %v", err)
	}

	// Create PEM with multiple blocks (should parse first one)
	pubKeyBytes1, _ := x509.MarshalPKIXPublicKey(privKey1.Public())
	pubKeyBytes2, _ := x509.MarshalPKIXPublicKey(privKey2.Public())

	pem1 := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes1})
	pem2 := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes2})
	multiPEM := append(pem1, pem2...)

	parsed, err := ParsePublicKeyFromPEM(multiPEM)
	if err != nil {
		t.Fatalf("Failed to parse multi-block PEM: %v", err)
	}

	// Should parse the first key
	ed25519Key, ok := parsed.(Ed25519PubKey)
	if !ok {
		t.Fatalf("Expected Ed25519PubKey, got %T", parsed)
	}

	if !ed25519.PublicKey(ed25519Key.PublicKey).Equal(privKey1.Public().(ed25519.PublicKey)) {
		t.Error("Should have parsed the first key from multi-block PEM")
	}
}

func TestParsePublicKeyFromPEM_MalformedPEMStructure(t *testing.T) {
	tests := []struct {
		name    string
		pemData []byte
	}{
		{
			"Broken content",
			[]byte("-----BEGIN PUBLIC KEY-----\nbroken\n-----END PUBLIC KEY-----"),
		},
		{
			"Invalid base64",
			[]byte("-----BEGIN PUBLIC KEY-----\n!!invalid base64!!\n-----END PUBLIC KEY-----"),
		},
		{
			"Empty content",
			[]byte("-----BEGIN PUBLIC KEY-----\n-----END PUBLIC KEY-----"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParsePublicKeyFromPEM(test.pemData)
			if err == nil {
				t.Error("Expected error for malformed PEM structure")
			}
		})
	}
}
