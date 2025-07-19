package service

import (
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// CircuitPath represents a path through the Tor network
type CircuitPath struct {
	Relays      []*entity.Relay
	Keys        []value_object.AESKey
	Nonces      []value_object.Nonce
	Connections []ConnectionSpec
}

// ConnectionSpec represents connection requirements for a hop
type ConnectionSpec struct {
	RelayID   value_object.RelayID
	Endpoint  value_object.Endpoint
	PublicKey *value_object.RSAPubKey
	IsExit    bool
}

// RelaySelectionCriteria represents criteria for selecting relays
type RelaySelectionCriteria struct {
	RequiredExitRelay *value_object.RelayID
	MinHops           int
	MaxHops           int
	ExcludeRelays     []value_object.RelayID
	RequireCapabilities []RelayCapability
}

// RelayCapability represents capabilities that a relay must have
type RelayCapability int

const (
	CapabilityExit RelayCapability = iota
	CapabilityHiddenService
	CapabilityGuard
)

// CircuitEstablishmentPlan represents a plan for establishing a circuit
type CircuitEstablishmentPlan struct {
	Path           CircuitPath
	HandshakeSteps []HandshakeStep
	ErrorRecovery  ErrorRecoveryPlan
}

// HandshakeStep represents a single step in circuit establishment
type HandshakeStep struct {
	HopIndex      int
	TargetRelay   *entity.Relay
	HandshakeType HandshakeType
	PublicKey     []byte
	ExpectedResponse ResponseType
}

// HandshakeType represents the type of handshake to perform
type HandshakeType int

const (
	HandshakeNtor HandshakeType = iota
	HandshakeTAP
	HandshakeX25519 // Current implementation
)

// ResponseType represents expected response from handshake
type ResponseType int

const (
	ResponseCreated ResponseType = iota
	ResponseDestroy
)

// ErrorRecoveryPlan represents how to handle circuit establishment failures
type ErrorRecoveryPlan struct {
	MaxRetries        int
	BackoffStrategy   BackoffStrategy
	AlternativeRelays []value_object.RelayID
}

// BackoffStrategy represents retry timing strategy
type BackoffStrategy int

const (
	BackoffLinear BackoffStrategy = iota
	BackoffExponential
)

// CircuitTopologyService handles circuit construction and relay selection
// This domain service centralizes circuit topology logic that was previously
// in the circuit build service.
type CircuitTopologyService interface {
	// SelectRelaysForCircuit selects appropriate relays for a circuit
	SelectRelaysForCircuit(criteria RelaySelectionCriteria) ([]*entity.Relay, error)
	
	// CreateEstablishmentPlan creates a plan for establishing a circuit
	CreateEstablishmentPlan(relays []*entity.Relay) (*CircuitEstablishmentPlan, error)
	
	// ValidateCircuitPath validates that a circuit path is valid
	ValidateCircuitPath(path CircuitPath) error
	
	// OptimizeRelaySelection optimizes relay selection based on network conditions
	OptimizeRelaySelection(available []*entity.Relay, criteria RelaySelectionCriteria) []*entity.Relay
	
	// CalculateCircuitMetrics calculates performance metrics for a circuit path
	CalculateCircuitMetrics(path CircuitPath) CircuitMetrics
	
	// DetermineExitRelay determines the appropriate exit relay for a destination
	DetermineExitRelay(destination string, available []*entity.Relay) (*entity.Relay, error)
	
	// CreateAlternativePath creates an alternative path when the primary fails
	CreateAlternativePath(failedPath CircuitPath, reason PathFailureReason) (*CircuitPath, error)
}

// CircuitMetrics represents performance metrics for a circuit
type CircuitMetrics struct {
	EstimatedLatency    int64  // milliseconds
	EstimatedBandwidth  int64  // bytes per second  
	RelayReliability    float64 // 0.0 to 1.0
	GeographicDiversity float64 // 0.0 to 1.0
}

// PathFailureReason represents why a circuit path failed
type PathFailureReason int

const (
	FailureConnectionTimeout PathFailureReason = iota
	FailureHandshakeError
	FailureRelayUnreachable
	FailureCryptographicError
)

// RelayPool represents a pool of available relays with different capabilities
type RelayPool struct {
	GuardRelays  []*entity.Relay
	MiddleRelays []*entity.Relay
	ExitRelays   []*entity.Relay
	HSRelays     []*entity.Relay // Hidden Service relays
}

// CircuitConstraints represents constraints for circuit construction
type CircuitConstraints struct {
	MaxPathLength      int
	MinPathLength      int
	RequireSameCountry bool
	AvoidCountries     []string
	RequireIPv6        bool
}