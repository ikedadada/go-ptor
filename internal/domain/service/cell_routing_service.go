package service

import (
	"net"
	
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// RoutingDecision represents how a cell should be routed
type RoutingDecision struct {
	Action      RoutingAction
	Destination RoutingDestination
	Cell        *value_object.Cell
	Error       error
}

// RoutingAction represents the routing action to take
type RoutingAction int

const (
	RouteForward RoutingAction = iota // Forward the cell
	RouteTerminate                    // Handle the cell locally
	RouteReject                       // Reject/drop the cell
	RouteEncryptAndForward            // Add encryption and forward
)

// RoutingDestination represents where to route the cell
type RoutingDestination struct {
	Connection   net.Conn
	Direction    RoutingDirection
	RequiresAuth bool
}

// RoutingDirection represents the direction of routing
type RoutingDirection int

const (
	RoutingUpstream RoutingDirection = iota   // Towards client
	RoutingDownstream                         // Towards server
)

// CellRoutingService handles all cell routing decisions
// This domain service centralizes routing logic that was previously
// embedded in the relay use case.
type CellRoutingService interface {
	// RouteCell determines how to route a cell based on relay state and cell type
	RouteCell(connState *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) (*RoutingDecision, error)
	
	// CreateForwardingCell creates a cell for forwarding to the next hop
	CreateForwardingCell(originalCell *value_object.Cell, payload []byte) *value_object.Cell
	
	// CreateResponseCell creates a response cell to send back
	CreateResponseCell(command byte, payload []byte) *value_object.Cell
	
	// DetermineForwardingDestination determines where to forward a cell
	DetermineForwardingDestination(connState *entity.ConnState, direction RoutingDirection) (net.Conn, error)
	
	// ValidateCircuitID validates if a circuit ID is known and active
	ValidateCircuitID(cid value_object.CircuitID) bool
	
	// ShouldCreateCircuit determines if a new circuit should be created
	ShouldCreateCircuit(cell *value_object.Cell) bool
}

// ForwardingInstruction represents instructions for forwarding cells
type ForwardingInstruction struct {
	TargetConnection net.Conn
	ModifiedPayload  []byte
	ShouldEncrypt    bool
	EncryptionKeys   []value_object.AESKey
	EncryptionNonces []value_object.Nonce
}

// CircuitPathInfo represents information about the circuit path
type CircuitPathInfo struct {
	IsFirstHop bool
	IsLastHop  bool
	HopIndex   int
	TotalHops  int
}

// RoutingContext provides context for routing decisions
type RoutingContext struct {
	CircuitID     value_object.CircuitID
	ConnState     *entity.ConnState
	PathInfo      CircuitPathInfo
	MessageType   entity.MessageType
	IsNewCircuit  bool
}

// CellForwardingPolicy defines policies for cell forwarding
type CellForwardingPolicy struct {
	AllowUpstreamForwarding   bool
	AllowDownstreamForwarding bool
	RequireEncryption         bool
	MaxForwardingHops         int
}