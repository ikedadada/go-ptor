package service

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func TestCryptoService_RSAEncryptDecrypt(t *testing.T) {
	crypto := NewCryptoService()

	// Generate test key pair
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pub := &priv.PublicKey

	testData := []byte("Hello, RSA encryption!")

	// Test encryption
	encrypted, err := crypto.RSAEncrypt(pub, testData)
	if err != nil {
		t.Fatalf("RSA encryption failed: %v", err)
	}

	if len(encrypted) == 0 {
		t.Error("Encrypted data should not be empty")
	}

	// Test decryption
	decrypted, err := crypto.RSADecrypt(priv, encrypted)
	if err != nil {
		t.Fatalf("RSA decryption failed: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Decrypted data mismatch. Expected: %s, Got: %s", testData, decrypted)
	}
}

func TestCryptoService_RSADecrypt_InvalidInput(t *testing.T) {
	crypto := NewCryptoService()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Test with invalid encrypted data
	invalidData := []byte("invalid encrypted data")
	_, err = crypto.RSADecrypt(priv, invalidData)
	if err == nil {
		t.Error("Expected RSA decryption to fail with invalid input")
	}
}

func TestCryptoService_AESSealOpen(t *testing.T) {
	crypto := NewCryptoService()

	// Test key and nonce
	var key [32]byte
	var nonce [12]byte
	rand.Read(key[:])
	rand.Read(nonce[:])

	testData := []byte("Hello, AES encryption!")

	// Test sealing
	sealed, err := crypto.AESSeal(key, nonce, testData)
	if err != nil {
		t.Fatalf("AES seal failed: %v", err)
	}

	if len(sealed) == 0 {
		t.Error("Sealed data should not be empty")
	}

	// Test opening
	opened, err := crypto.AESOpen(key, nonce, sealed)
	if err != nil {
		t.Fatalf("AES open failed: %v", err)
	}

	if string(opened) != string(testData) {
		t.Errorf("Opened data mismatch. Expected: %s, Got: %s", testData, opened)
	}
}

func TestCryptoService_AESOpen_InvalidInput(t *testing.T) {
	crypto := NewCryptoService()

	var key [32]byte
	var nonce [12]byte
	rand.Read(key[:])
	rand.Read(nonce[:])

	// Test with invalid encrypted data
	invalidData := []byte("invalid encrypted data")
	_, err := crypto.AESOpen(key, nonce, invalidData)
	if err == nil {
		t.Error("Expected AES open to fail with invalid input")
	}
}

func TestCryptoService_AESMultiSeal_OnionEncryption(t *testing.T) {
	crypto := NewCryptoService()

	// Test 3-hop onion encryption
	hops := 3
	keys := make([][32]byte, hops)
	nonces := make([][12]byte, hops)

	for i := 0; i < hops; i++ {
		rand.Read(keys[i][:])
		rand.Read(nonces[i][:])
	}

	testData := []byte("Onion routing message")

	// Multi-layer encryption
	encrypted, err := crypto.AESMultiSeal(keys, nonces, testData)
	if err != nil {
		t.Fatalf("AES multi-seal failed: %v", err)
	}

	if len(encrypted) == 0 {
		t.Error("Multi-sealed data should not be empty")
	}

	// Multi-layer decryption
	decrypted, err := crypto.AESMultiOpen(keys, nonces, encrypted)
	if err != nil {
		t.Fatalf("AES multi-open failed: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Multi-layer decryption mismatch. Expected: %s, Got: %s", testData, decrypted)
	}
}

func TestCryptoService_AESMultiSeal_MismatchedLengths(t *testing.T) {
	crypto := NewCryptoService()

	// Mismatched keys and nonces lengths
	keys := make([][32]byte, 2)
	nonces := make([][12]byte, 3)

	testData := []byte("test")

	_, err := crypto.AESMultiSeal(keys, nonces, testData)
	if err == nil {
		t.Error("Expected AES multi-seal to fail with mismatched lengths")
	}

	_, err = crypto.AESMultiOpen(keys, nonces, testData)
	if err == nil {
		t.Error("Expected AES multi-open to fail with mismatched lengths")
	}
}

func TestCryptoService_X25519Generate(t *testing.T) {
	crypto := NewCryptoService()

	priv, pub, err := crypto.X25519Generate()
	if err != nil {
		t.Fatalf("X25519 key generation failed: %v", err)
	}

	if len(priv) != 32 {
		t.Errorf("Private key should be 32 bytes, got %d", len(priv))
	}

	if len(pub) != 32 {
		t.Errorf("Public key should be 32 bytes, got %d", len(pub))
	}

	// Keys should be different
	if string(priv) == string(pub) {
		t.Error("Private and public keys should be different")
	}
}

func TestCryptoService_X25519Shared(t *testing.T) {
	crypto := NewCryptoService()

	// Generate Alice's key pair
	alicePriv, alicePub, err := crypto.X25519Generate()
	if err != nil {
		t.Fatalf("Failed to generate Alice's keys: %v", err)
	}

	// Generate Bob's key pair
	bobPriv, bobPub, err := crypto.X25519Generate()
	if err != nil {
		t.Fatalf("Failed to generate Bob's keys: %v", err)
	}

	// Alice computes shared secret with Bob's public key
	aliceShared, err := crypto.X25519Shared(alicePriv, bobPub)
	if err != nil {
		t.Fatalf("Alice's shared secret computation failed: %v", err)
	}

	// Bob computes shared secret with Alice's public key
	bobShared, err := crypto.X25519Shared(bobPriv, alicePub)
	if err != nil {
		t.Fatalf("Bob's shared secret computation failed: %v", err)
	}

	// Shared secrets should be identical
	if string(aliceShared) != string(bobShared) {
		t.Error("Shared secrets should be identical")
	}

	if len(aliceShared) != 32 {
		t.Errorf("Shared secret should be 32 bytes, got %d", len(aliceShared))
	}
}

func TestCryptoService_X25519Shared_InvalidInput(t *testing.T) {
	crypto := NewCryptoService()

	// Test with invalid key sizes
	invalidPriv := []byte("invalid")
	invalidPub := []byte("invalid")

	_, err := crypto.X25519Shared(invalidPriv, invalidPub)
	if err == nil {
		t.Error("Expected X25519 shared secret computation to fail with invalid input")
	}
}

func TestCryptoService_DeriveKeyNonce(t *testing.T) {
	crypto := NewCryptoService()

	secret := []byte("shared secret for key derivation")

	key, nonce, err := crypto.DeriveKeyNonce(secret)
	if err != nil {
		t.Fatalf("Key/nonce derivation failed: %v", err)
	}

	// Check key length
	if len(key) != 32 {
		t.Errorf("Key should be 32 bytes, got %d", len(key))
	}

	// Check nonce length
	if len(nonce) != 12 {
		t.Errorf("Nonce should be 12 bytes, got %d", len(nonce))
	}

	// Derivation should be deterministic
	key2, nonce2, err := crypto.DeriveKeyNonce(secret)
	if err != nil {
		t.Fatalf("Second key/nonce derivation failed: %v", err)
	}

	if key != key2 {
		t.Error("Key derivation should be deterministic")
	}

	if nonce != nonce2 {
		t.Error("Nonce derivation should be deterministic")
	}
}

func TestCryptoService_ModifyNonceWithSequence(t *testing.T) {
	crypto := NewCryptoService()

	var baseNonce [12]byte
	rand.Read(baseNonce[:])

	// Test with different sequences
	seq1 := uint64(1)
	seq2 := uint64(2)
	seq3 := uint64(0xFFFFFFFFFFFFFFFF) // Max uint64

	nonce1 := crypto.ModifyNonceWithSequence(baseNonce, seq1)
	nonce2 := crypto.ModifyNonceWithSequence(baseNonce, seq2)
	nonce3 := crypto.ModifyNonceWithSequence(baseNonce, seq3)

	// Nonces should be different for different sequences
	if nonce1 == nonce2 {
		t.Error("Nonces should be different for different sequences")
	}

	if nonce1 == nonce3 {
		t.Error("Nonces should be different for different sequences")
	}

	// Test sequence 0 should return base nonce
	nonce0 := crypto.ModifyNonceWithSequence(baseNonce, 0)
	if nonce0 != baseNonce {
		t.Error("Sequence 0 should return base nonce")
	}
}

func TestCryptoService_Integration_CompleteOnionFlow(t *testing.T) {
	crypto := NewCryptoService()

	// Test complete onion routing encryption flow
	// 1. Generate X25519 key pairs for 3 hops
	hops := 3
	privKeys := make([][]byte, hops)
	pubKeys := make([][]byte, hops)

	for i := 0; i < hops; i++ {
		priv, pub, err := crypto.X25519Generate()
		if err != nil {
			t.Fatalf("Failed to generate keys for hop %d: %v", i, err)
		}
		privKeys[i] = priv
		pubKeys[i] = pub
	}

	// 2. Client generates ephemeral key pair
	clientPriv, _, err := crypto.X25519Generate()
	if err != nil {
		t.Fatalf("Failed to generate client keys: %v", err)
	}

	// 3. Derive shared secrets with each hop
	keys := make([][32]byte, hops)
	nonces := make([][12]byte, hops)

	for i := 0; i < hops; i++ {
		shared, err := crypto.X25519Shared(clientPriv, pubKeys[i])
		if err != nil {
			t.Fatalf("Failed to compute shared secret for hop %d: %v", i, err)
		}

		key, nonce, err := crypto.DeriveKeyNonce(shared)
		if err != nil {
			t.Fatalf("Failed to derive key/nonce for hop %d: %v", i, err)
		}

		keys[i] = key
		nonces[i] = nonce
	}

	// 4. Test onion encryption/decryption
	originalMessage := []byte("Secret message through onion routing")

	encrypted, err := crypto.AESMultiSeal(keys, nonces, originalMessage)
	if err != nil {
		t.Fatalf("Onion encryption failed: %v", err)
	}

	decrypted, err := crypto.AESMultiOpen(keys, nonces, encrypted)
	if err != nil {
		t.Fatalf("Onion decryption failed: %v", err)
	}

	if string(decrypted) != string(originalMessage) {
		t.Errorf("Complete onion flow failed. Expected: %s, Got: %s", originalMessage, decrypted)
	}

	// 5. Verify that hop-by-hop decryption works (relay simulation)
	currentData := encrypted
	for i := 0; i < hops; i++ {
		currentData, err = crypto.AESOpen(keys[i], nonces[i], currentData)
		if err != nil {
			t.Fatalf("Hop %d decryption failed: %v", i, err)
		}
	}

	if string(currentData) != string(originalMessage) {
		t.Errorf("Hop-by-hop decryption failed. Expected: %s, Got: %s", originalMessage, currentData)
	}
}

func TestCryptoService_ModifyNonceSequenceUniqueness(t *testing.T) {
	crypto := NewCryptoService()

	var baseNonce [12]byte
	rand.Read(baseNonce[:])

	// Test that different sequences produce unique nonces
	sequences := []uint64{1, 2, 100, 65536, 0xFFFFFFFF}
	nonces := make(map[[12]byte]bool)

	for _, seq := range sequences {
		modifiedNonce := crypto.ModifyNonceWithSequence(baseNonce, seq)

		if nonces[modifiedNonce] {
			t.Errorf("Duplicate nonce found for sequence %d", seq)
		}
		nonces[modifiedNonce] = true
	}

	// Test sequence increment pattern
	seq := uint64(0)
	nonce1 := crypto.ModifyNonceWithSequence(baseNonce, seq)
	seq++
	nonce2 := crypto.ModifyNonceWithSequence(baseNonce, seq)

	if nonce1 == nonce2 {
		t.Error("Sequential nonces should be different")
	}
}
