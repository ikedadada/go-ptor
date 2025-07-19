package service

import (
	"fmt"
	"os"
	
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// relayBehaviorServiceImpl implements RelayBehaviorService
type relayBehaviorServiceImpl struct {
	cryptoService CircuitCryptographyService
}

// NewRelayBehaviorService creates a new RelayBehaviorService
func NewRelayBehaviorService(cryptoService CircuitCryptographyService) RelayBehaviorService {
	return &relayBehaviorServiceImpl{
		cryptoService: cryptoService,
	}
}

// DetermineRelayType determines if this relay is a middle or exit relay
func (s *relayBehaviorServiceImpl) DetermineRelayType(connState *entity.ConnState) RelayType {
	if connState.Down() != nil {
		return RelayTypeMiddle
	}
	return RelayTypeExit
}

// HandleConnectCell processes CONNECT cells based on relay type
func (s *relayBehaviorServiceImpl) HandleConnectCell(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error) {
	relayType := s.DetermineRelayType(connState)
	
	switch relayType {
	case RelayTypeMiddle:
		return s.handleConnectAsMiddleRelay(connState, cell)
	case RelayTypeExit:
		return s.handleConnectAsExitRelay(connState, cell)
	default:
		return nil, fmt.Errorf("unknown relay type")
	}
}

// HandleBeginCell processes BEGIN cells based on relay type
func (s *relayBehaviorServiceImpl) HandleBeginCell(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error) {
	relayType := s.DetermineRelayType(connState)
	
	switch relayType {
	case RelayTypeMiddle:
		return s.handleBeginAsMiddleRelay(connState, cell)
	case RelayTypeExit:
		return s.handleBeginAsExitRelay(connState, cell)
	default:
		return nil, fmt.Errorf("unknown relay type")
	}
}

// HandleDataCell processes DATA cells based on relay type and data direction
func (s *relayBehaviorServiceImpl) HandleDataCell(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error) {
	relayType := s.DetermineRelayType(connState)
	
	// Try to decrypt the data first
	decrypted, success, err := s.cryptoService.DecryptAtRelay(connState, entity.MessageTypeData, cell.Payload)
	
	if success && err == nil {
		// Successfully decrypted - this is downstream data
		return s.handleDownstreamData(connState, relayType, cell, decrypted)
	} else if relayType == RelayTypeMiddle {
		// Decryption failed on middle relay - likely upstream data
		return s.handleUpstreamData(connState, cell)
	} else {
		// Exit relay decryption failed - this is an error
		return nil, fmt.Errorf("exit relay failed to decrypt data: %w", err)
	}
}

// HandleEndCell processes END cells based on relay type
func (s *relayBehaviorServiceImpl) HandleEndCell(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error) {
	relayType := s.DetermineRelayType(connState)
	
	switch relayType {
	case RelayTypeMiddle:
		// Middle relay forwards END cells
		return &CellHandlingInstruction{
			Action:      ActionForwardDownstream,
			ForwardCell: cell,
		}, nil
	case RelayTypeExit:
		// Exit relay handles END cells locally
		return &CellHandlingInstruction{
			Action: ActionTerminate,
		}, nil
	default:
		return nil, fmt.Errorf("unknown relay type")
	}
}

// ShouldDecryptCell determines if this relay should attempt to decrypt a cell
func (s *relayBehaviorServiceImpl) ShouldDecryptCell(connState *entity.ConnState, cellType byte) bool {
	switch cellType {
	case value_object.CmdBegin, value_object.CmdConnect, value_object.CmdData:
		return true
	default:
		return false
	}
}

// CreateConnectionInstruction creates instructions for establishing connections
func (s *relayBehaviorServiceImpl) CreateConnectionInstruction(target string, streamID value_object.StreamID, isHidden bool) *ConnectionInfo {
	return &ConnectionInfo{
		Target:     target,
		StreamID:   streamID,
		IsHidden:   isHidden,
		ShouldDial: target != "",
	}
}

// DetermineDataDirection determines if data is flowing upstream or downstream
func (s *relayBehaviorServiceImpl) DetermineDataDirection(connState *entity.ConnState, decryptionSuccess bool) DataDirection {
	if decryptionSuccess {
		return DataDirectionDownstream
	}
	
	// If decryption failed and we're a middle relay, it's likely upstream data
	if s.DetermineRelayType(connState) == RelayTypeMiddle {
		return DataDirectionUpstream
	}
	
	return DataDirectionDownstream
}

// handleConnectAsMiddleRelay handles CONNECT cells for middle relays
func (s *relayBehaviorServiceImpl) handleConnectAsMiddleRelay(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error) {
	// Decrypt one layer and forward
	decrypted, _, err := s.cryptoService.DecryptAtRelay(connState, entity.MessageTypeConnect, cell.Payload)
	if err != nil {
		return nil, fmt.Errorf("middle relay connect decrypt failed: %w", err)
	}
	
	forwardCell := &value_object.Cell{
		Cmd:     value_object.CmdConnect,
		Version: value_object.Version,
		Payload: decrypted,
	}
	
	return &CellHandlingInstruction{
		Action:      ActionForwardDownstream,
		ForwardCell: forwardCell,
	}, nil
}

// handleConnectAsExitRelay handles CONNECT cells for exit relays
func (s *relayBehaviorServiceImpl) handleConnectAsExitRelay(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error) {
	// Decrypt final payload and create connection
	decrypted, _, err := s.cryptoService.DecryptAtRelay(connState, entity.MessageTypeConnect, cell.Payload)
	if err != nil {
		return nil, fmt.Errorf("exit relay connect decrypt failed: %w", err)
	}
	
	// Determine target address
	target := s.getHiddenServiceAddress()
	if len(decrypted) > 0 {
		payload, err := value_object.DecodeConnectPayload(decrypted)
		if err == nil && payload.Target != "" {
			target = payload.Target
		}
	}
	
	connectionInfo := s.CreateConnectionInstruction(target, 0, true)
	
	return &CellHandlingInstruction{
		Action:     ActionCreateConnection,
		Connection: *connectionInfo,
		Response: &value_object.Cell{
			Cmd:     value_object.CmdBeginAck,
			Version: value_object.Version,
		},
	}, nil
}

// handleBeginAsMiddleRelay handles BEGIN cells for middle relays
func (s *relayBehaviorServiceImpl) handleBeginAsMiddleRelay(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error) {
	// Decrypt and forward
	decrypted, _, err := s.cryptoService.DecryptAtRelay(connState, entity.MessageTypeBegin, cell.Payload)
	if err != nil {
		return nil, fmt.Errorf("middle relay begin decrypt failed: %w", err)
	}
	
	forwardCell := &value_object.Cell{
		Cmd:     value_object.CmdBegin,
		Version: value_object.Version,
		Payload: decrypted,
	}
	
	return &CellHandlingInstruction{
		Action:      ActionForwardDownstream,
		ForwardCell: forwardCell,
	}, nil
}

// handleBeginAsExitRelay handles BEGIN cells for exit relays
func (s *relayBehaviorServiceImpl) handleBeginAsExitRelay(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error) {
	// Decrypt and establish connection
	decrypted, _, err := s.cryptoService.DecryptAtRelay(connState, entity.MessageTypeBegin, cell.Payload)
	if err != nil {
		return nil, fmt.Errorf("exit relay begin decrypt failed: %w", err)
	}
	
	if connState.IsHidden() {
		// Hidden service - send ACK and start forwarding
		return &CellHandlingInstruction{
			Action: ActionTerminate,
			Response: &value_object.Cell{
				Cmd:     value_object.CmdBeginAck,
				Version: value_object.Version,
			},
		}, nil
	}
	
	// Regular exit relay - parse target and create connection
	beginPayload, err := value_object.DecodeBeginPayload(decrypted)
	if err != nil {
		return nil, fmt.Errorf("decode begin payload failed: %w", err)
	}
	
	streamID, err := value_object.StreamIDFrom(beginPayload.StreamID)
	if err != nil {
		return nil, fmt.Errorf("invalid stream ID: %w", err)
	}
	
	connectionInfo := s.CreateConnectionInstruction(beginPayload.Target, streamID, false)
	
	return &CellHandlingInstruction{
		Action:     ActionCreateConnection,
		Connection: *connectionInfo,
		Response: &value_object.Cell{
			Cmd:     value_object.CmdBeginAck,
			Version: value_object.Version,
		},
	}, nil
}

// handleDownstreamData handles successfully decrypted downstream data
func (s *relayBehaviorServiceImpl) handleDownstreamData(connState *entity.ConnState, relayType RelayType, originalCell *value_object.Cell, decrypted []byte) (*CellHandlingInstruction, error) {
	switch relayType {
	case RelayTypeMiddle:
		// Forward with one layer removed
		forwardCell := &value_object.Cell{
			Cmd:     value_object.CmdData,
			Version: value_object.Version,
			Payload: decrypted,
		}
		return &CellHandlingInstruction{
			Action:      ActionForwardDownstream,
			ForwardCell: forwardCell,
		}, nil
		
	case RelayTypeExit:
		// Handle locally (write to stream)
		return &CellHandlingInstruction{
			Action: ActionTerminate,
		}, nil
		
	default:
		return nil, fmt.Errorf("unknown relay type")
	}
}

// handleUpstreamData handles upstream data that failed decryption (middle relay only)
func (s *relayBehaviorServiceImpl) handleUpstreamData(connState *entity.ConnState, cell *value_object.Cell) (*CellHandlingInstruction, error) {
	// Parse the data payload to access the inner data
	dataPayload, err := value_object.DecodeDataPayload(cell.Payload)
	if err != nil {
		return nil, fmt.Errorf("decode upstream data payload failed: %w", err)
	}
	
	// Add encryption layer for upstream flow
	encrypted, err := s.cryptoService.EncryptAtRelay(connState, entity.MessageTypeUpstreamData, dataPayload.Data)
	if err != nil {
		return nil, fmt.Errorf("encrypt upstream data failed: %w", err)
	}
	
	// Create new payload with encrypted data
	newPayload, err := value_object.EncodeDataPayload(&value_object.DataPayload{
		StreamID: dataPayload.StreamID,
		Data:     encrypted,
	})
	if err != nil {
		return nil, fmt.Errorf("encode upstream data payload failed: %w", err)
	}
	
	forwardCell := &value_object.Cell{
		Cmd:     value_object.CmdData,
		Version: value_object.Version,
		Payload: newPayload,
	}
	
	return &CellHandlingInstruction{
		Action:      ActionEncryptAndForward,
		ForwardCell: forwardCell,
	}, nil
}

// getHiddenServiceAddress gets the hidden service address from environment
func (s *relayBehaviorServiceImpl) getHiddenServiceAddress() string {
	addr := os.Getenv("PTOR_HIDDEN_ADDR")
	if addr == "" {
		addr = os.Getenv("HIDDEN_ADDR")
	}
	if addr == "" {
		addr = "hidden:5000"
	}
	return addr
}