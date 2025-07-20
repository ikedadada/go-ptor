package aggregate

import (
	"fmt"
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// RelayCell is an aggregate that combines low-level protocol cells
// with high-level circuit and stream concepts for relay operations.
type RelayCell struct {
	// Core protocol cell
	cell entity.Cell
	
	// High-level identifiers
	circuitID value_object.CircuitID
	streamID  value_object.StreamID
	
	// Relay-specific state
	end bool
}

// NewRelayCell creates a new relay cell aggregate
func NewRelayCell(
	cmd value_object.CellCommand,
	circuitID value_object.CircuitID,
	streamID value_object.StreamID,
	data []byte,
) (*RelayCell, error) {
	if len(data) > entity.MaxPayloadSize {
		return nil, fmt.Errorf("data too large: %d > %d", len(data), entity.MaxPayloadSize)
	}
	
	cell := entity.Cell{
		Cmd:     cmd,
		Version: value_object.ProtocolV1,
		Payload: data,
	}
	
	return &RelayCell{
		cell:      cell,
		circuitID: circuitID,
		streamID:  streamID,
		end:       false,
	}, nil
}

// Cell returns the underlying protocol cell
func (rc *RelayCell) Cell() entity.Cell {
	return rc.cell
}

// CircuitID returns the circuit identifier
func (rc *RelayCell) CircuitID() value_object.CircuitID {
	return rc.circuitID
}

// StreamID returns the stream identifier
func (rc *RelayCell) StreamID() value_object.StreamID {
	return rc.streamID
}

// Data returns the cell payload data
func (rc *RelayCell) Data() []byte {
	// Return copy to maintain immutability
	data := make([]byte, len(rc.cell.Payload))
	copy(data, rc.cell.Payload)
	return data
}

// IsEnd returns whether this cell marks the end of a stream
func (rc *RelayCell) IsEnd() bool {
	return rc.end
}

// MarkEnd marks this cell as the end of a stream
func (rc *RelayCell) MarkEnd() {
	rc.end = true
}

// Command returns the cell command
func (rc *RelayCell) Command() value_object.CellCommand {
	return rc.cell.Cmd
}

// Encode serializes the relay cell into wire format
func (rc *RelayCell) Encode() ([]byte, error) {
	return entity.Encode(rc.cell)
}

// IsDataCell returns true if this is a data-carrying cell
func (rc *RelayCell) IsDataCell() bool {
	return rc.cell.Cmd == value_object.CmdData
}

// IsControlCell returns true if this is a control cell
func (rc *RelayCell) IsControlCell() bool {
	switch rc.cell.Cmd {
	case value_object.CmdBegin, value_object.CmdEnd, value_object.CmdConnect, 
		 value_object.CmdExtend, value_object.CmdDestroy, value_object.CmdCreated, 
		 value_object.CmdBeginAck:
		return true
	default:
		return false
	}
}

// ValidateForCircuit validates that this cell is appropriate for the given circuit
func (rc *RelayCell) ValidateForCircuit(expectedCircuitID value_object.CircuitID) error {
	if rc.circuitID != expectedCircuitID {
		return fmt.Errorf("circuit ID mismatch: expected %s, got %s", 
			expectedCircuitID.String(), rc.circuitID.String())
	}
	return nil
}