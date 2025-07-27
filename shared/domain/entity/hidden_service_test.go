package entity

import (
	"crypto/ed25519"
	"testing"

	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// equalPublicKeys compares two PublicKey instances by their underlying bytes
func equalPublicKeys(a, b vo.PublicKey) bool {
	ed25519A, okA := a.(vo.Ed25519PubKey)
	ed25519B, okB := b.(vo.Ed25519PubKey)
	if !okA || !okB {
		return false
	}
	return ed25519.PublicKey(ed25519A.PublicKey).Equal(ed25519.PublicKey(ed25519B.PublicKey))
}

// createTestPublicKey creates a test Ed25519 public key with optional custom bytes
func createTestPublicKey(customBytes ...byte) vo.Ed25519PubKey {
	keyBytes := make([]byte, 32)
	if len(customBytes) > 0 {
		copy(keyBytes, customBytes)
	}
	return vo.Ed25519PubKey{PublicKey: ed25519.PublicKey(keyBytes)}
}

func TestNewHiddenService(t *testing.T) {
	// Create test data
	address := vo.HiddenAddrFromString("test-hidden-service.onion")

	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("Failed to create relay ID: %v", err)
	}

	pubKey := createTestPublicKey(1, 2, 3, 4, 5)

	// Create hidden service
	hs := NewHiddenService(address, relayID, pubKey)

	// Verify all fields
	if hs.Address() != address {
		t.Errorf("Address mismatch: got %v, want %v", hs.Address(), address)
	}

	if hs.RelayID() != relayID {
		t.Errorf("RelayID mismatch: got %v, want %v", hs.RelayID(), relayID)
	}

	if !equalPublicKeys(hs.PubKey(), pubKey) {
		t.Errorf("PubKey mismatch: got %v, want %v", hs.PubKey(), pubKey)
	}
}

func TestHiddenService_Address(t *testing.T) {
	address := vo.HiddenAddrFromString("example.onion")
	relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	pubKey := createTestPublicKey()

	hs := NewHiddenService(address, relayID, pubKey)

	result := hs.Address()
	if result != address {
		t.Errorf("Address() returned %v, want %v", result, address)
	}
}

func TestHiddenService_RelayID(t *testing.T) {
	address := vo.HiddenAddrFromString("example.onion")
	relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	pubKey := createTestPublicKey()

	hs := NewHiddenService(address, relayID, pubKey)

	result := hs.RelayID()
	if result != relayID {
		t.Errorf("RelayID() returned %v, want %v", result, relayID)
	}
}

func TestHiddenService_PubKey(t *testing.T) {
	address := vo.HiddenAddrFromString("example.onion")
	relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	pubKey := createTestPublicKey(42, 43, 44, 45)

	hs := NewHiddenService(address, relayID, pubKey)

	result := hs.PubKey()
	if !equalPublicKeys(result, pubKey) {
		t.Errorf("PubKey() returned %v, want %v", result, pubKey)
	}
}

func TestHiddenService_UpdateRelay(t *testing.T) {
	address := vo.HiddenAddrFromString("example.onion")
	originalRelayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	newRelayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440001")
	pubKey := createTestPublicKey()

	hs := NewHiddenService(address, originalRelayID, pubKey)

	// Verify original relay ID
	if hs.RelayID() != originalRelayID {
		t.Errorf("Initial RelayID mismatch: got %v, want %v", hs.RelayID(), originalRelayID)
	}

	// Update relay ID
	hs.UpdateRelay(newRelayID)

	// Verify updated relay ID
	if hs.RelayID() != newRelayID {
		t.Errorf("Updated RelayID mismatch: got %v, want %v", hs.RelayID(), newRelayID)
	}

	// Verify other fields unchanged
	if hs.Address() != address {
		t.Errorf("Address should remain unchanged after UpdateRelay")
	}
	if !equalPublicKeys(hs.PubKey(), pubKey) {
		t.Errorf("PubKey should remain unchanged after UpdateRelay")
	}
}

func TestHiddenService_Immutability(t *testing.T) {
	address := vo.HiddenAddrFromString("test.onion")
	relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	pubKey := createTestPublicKey(1, 2, 3)

	hs := NewHiddenService(address, relayID, pubKey)

	// Get references to the values
	gotAddress := hs.Address()
	gotRelayID := hs.RelayID()
	gotPubKey := hs.PubKey()

	// Verify they match the originals
	if gotAddress != address {
		t.Error("Address reference mismatch")
	}
	if gotRelayID != relayID {
		t.Error("RelayID reference mismatch")
	}
	if !equalPublicKeys(gotPubKey, pubKey) {
		t.Error("PubKey reference mismatch")
	}
}

func TestHiddenService_MultipleUpdates(t *testing.T) {
	address := vo.HiddenAddrFromString("multi-update.onion")
	relay1, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440001")
	relay2, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440002")
	relay3, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440003")
	pubKey := createTestPublicKey()

	hs := NewHiddenService(address, relay1, pubKey)

	// First update
	hs.UpdateRelay(relay2)
	if hs.RelayID() != relay2 {
		t.Errorf("First update failed: got %v, want %v", hs.RelayID(), relay2)
	}

	// Second update
	hs.UpdateRelay(relay3)
	if hs.RelayID() != relay3 {
		t.Errorf("Second update failed: got %v, want %v", hs.RelayID(), relay3)
	}

	// Update back to original
	hs.UpdateRelay(relay1)
	if hs.RelayID() != relay1 {
		t.Errorf("Update back to original failed: got %v, want %v", hs.RelayID(), relay1)
	}
}

func TestHiddenService_ZeroValues(t *testing.T) {
	// Test with zero/empty values to ensure no panics
	var address vo.HiddenAddr
	var relayID vo.RelayID
	pubKey := createTestPublicKey() // Create an empty but valid Ed25519 key

	hs := NewHiddenService(address, relayID, pubKey)

	// Should not panic and should return the zero values
	if hs.Address() != address {
		t.Error("Zero address not handled correctly")
	}
	if hs.RelayID() != relayID {
		t.Error("Zero relayID not handled correctly")
	}
	if !equalPublicKeys(hs.PubKey(), pubKey) {
		t.Error("Zero pubKey not handled correctly")
	}
}

func TestHiddenService_EntityIdentity(t *testing.T) {
	address := vo.HiddenAddrFromString("identity-test.onion")
	relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	pubKey := createTestPublicKey()

	hs1 := NewHiddenService(address, relayID, pubKey)
	hs2 := NewHiddenService(address, relayID, pubKey)

	// Two entities with same values should be different instances
	if hs1 == hs2 {
		t.Error("Two different HiddenService instances should not be equal")
	}

	// But their values should be equal
	if hs1.Address() != hs2.Address() {
		t.Error("Addresses should be equal")
	}
	if hs1.RelayID() != hs2.RelayID() {
		t.Error("RelayIDs should be equal")
	}
	if !equalPublicKeys(hs1.PubKey(), hs2.PubKey()) {
		t.Error("PubKeys should be equal")
	}
}
