package service

import (
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// RelayType represents the type of relay in the circuit
type RelayType int

const (
	RelayTypeMiddle RelayType = iota // Middle relay (has downstream connection)
	RelayTypeExit                    // Exit relay (no downstream connection)
)

// CellHandlingInstruction represents what action a relay should take with a cell
type CellHandlingInstruction struct {
	Action      CellAction
	ForwardCell *value_object.Cell
	Connection  ConnectionInfo
	Response    *value_object.Cell
}

// CellAction represents the action a relay should take
type CellAction int

const (
	ActionForwardDownstream CellAction = iota // Forward cell to next hop
	ActionForwardUpstream                     // Forward cell back to previous hop
	ActionTerminate                           // Handle cell locally (exit relay)
	ActionCreateConnection                    // Create new downstream connection
	ActionEncryptAndForward                   // Add encryption layer and forward upstream
)

// ConnectionInfo represents connection establishment parameters
type ConnectionInfo struct {
	Target     string
	StreamID   value_object.StreamID
	IsHidden   bool
	ShouldDial bool
}

// RelayBehaviorService encapsulates the business logic for how relays behave
// This domain service centralizes relay decision-making that was previously
// scattered throughout the relay use case.
type RelayBehaviorService interface {
	// DetermineRelayType determines if this relay is a middle or exit relay
	DetermineRelayType(connState *entity.ConnState) RelayType
	
	// HandleConnectCell processes CONNECT cells based on relay type
	HandleConnectCell(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error)
	
	// HandleBeginCell processes BEGIN cells based on relay type
	HandleBeginCell(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error)
	
	// HandleDataCell processes DATA cells based on relay type and data direction
	HandleDataCell(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error)
	
	// HandleEndCell processes END cells based on relay type
	HandleEndCell(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error)
	
	// ShouldDecryptCell determines if this relay should attempt to decrypt a cell
	ShouldDecryptCell(connState *entity.ConnState, cellType byte) bool
	
	// CreateConnectionInstruction creates instructions for establishing connections
	CreateConnectionInstruction(target string, streamID value_object.StreamID, isHidden bool) *ConnectionInfo
	
	// DetermineDataDirection determines if data is flowing upstream or downstream
	DetermineDataDirection(connState *entity.ConnState, decryptionSuccess bool) DataDirection
}

// DataDirection represents the direction of data flow through the circuit
type DataDirection int

const (
	DataDirectionDownstream DataDirection = iota // Client -> Server
	DataDirectionUpstream                        // Server -> Client
)

// RelayCapabilities represents what operations a relay can perform
type RelayCapabilities struct {
	CanForwardDownstream bool
	CanTerminateStreams  bool
	CanCreateConnections bool
	CanEncryptUpstream   bool
}

// StreamHandlingStrategy represents how a relay should handle stream operations
type StreamHandlingStrategy struct {
	ShouldCreateStream bool
	ShouldForwardToStream bool
	ShouldEncryptResponse bool
	TargetStreamID value_object.StreamID
}