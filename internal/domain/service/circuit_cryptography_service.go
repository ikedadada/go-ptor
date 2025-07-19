package service

import (
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// EncryptionDirection represents the direction of data flow
type EncryptionDirection int

const (
	DirectionDownstream EncryptionDirection = iota // Client -> Relay
	DirectionUpstream                              // Relay -> Client
)

// CircuitCryptographyService handles all cryptographic operations for Tor circuits
// This domain service centralizes encryption/decryption logic that was previously
// scattered across multiple use cases and entities.
type CircuitCryptographyService interface {
	// EncryptForCircuit encrypts payload for multi-hop circuit transmission
	// Used by clients when sending data through the circuit
	EncryptForCircuit(circuit *entity.Circuit, messageType entity.MessageType, payload []byte) ([]byte, error)
	
	// DecryptAtRelay decrypts one layer of encryption at a relay
	// Returns decrypted payload and whether this relay should handle the command
	DecryptAtRelay(connState *entity.ConnState, messageType entity.MessageType, payload []byte) ([]byte, bool, error)
	
	// EncryptAtRelay adds one layer of encryption at a relay for upstream data
	// Used when relays forward response data back to client
	EncryptAtRelay(connState *entity.ConnState, messageType entity.MessageType, payload []byte) ([]byte, error)
	
	// DecryptMultiLayer performs multi-layer decryption for client
	// Peels off all encryption layers applied by circuit hops
	DecryptMultiLayer(circuit *entity.Circuit, messageType entity.MessageType, payload []byte) ([]byte, error)
	
	// GenerateNonceForHop generates the appropriate nonce for a specific hop and message type
	GenerateNonceForHop(circuit *entity.Circuit, hopIndex int, messageType entity.MessageType) value_object.Nonce
	
	// GenerateNonceForRelay generates the appropriate nonce for relay operations
	GenerateNonceForRelay(connState *entity.ConnState, messageType entity.MessageType) value_object.Nonce
	
	// AdvanceNonce increments the nonce counter for the given message type
	AdvanceNonce(entity NonceCounterEntity, messageType entity.MessageType)
}

// NonceCounterEntity represents entities that maintain nonce counters
type NonceCounterEntity interface {
	GetMessageTypeNonce(messageType entity.MessageType) value_object.Nonce
	IncrementCounter(messageType entity.MessageType)
}

// CryptographicOperation represents a single cryptographic operation
type CryptographicOperation struct {
	Key       value_object.AESKey
	Nonce     value_object.Nonce
	Direction EncryptionDirection
	HopIndex  int
}

// EncryptionPlan represents a sequence of cryptographic operations
type EncryptionPlan struct {
	Operations []CryptographicOperation
	MessageType entity.MessageType
}