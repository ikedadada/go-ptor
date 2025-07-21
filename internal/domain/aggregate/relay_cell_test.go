package aggregate

import (
	"fmt"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

func TestNewRelayCell_Success(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	testData := []byte("test payload")
	
	relayCell, err := NewRelayCell(value_object.CmdData, circuitID, streamID, testData)
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}
	
	if relayCell == nil {
		t.Fatal("RelayCell should not be nil")
	}
	
	// Verify circuit ID
	if relayCell.CircuitID() != circuitID {
		t.Errorf("Circuit ID mismatch. Expected: %s, Got: %s", circuitID.String(), relayCell.CircuitID().String())
	}
	
	// Verify stream ID
	if relayCell.StreamID() != streamID {
		t.Errorf("Stream ID mismatch. Expected: %d, Got: %d", streamID.UInt16(), relayCell.StreamID().UInt16())
	}
	
	// Verify command
	if relayCell.Command() != value_object.CmdData {
		t.Errorf("Command mismatch. Expected: %d, Got: %d", value_object.CmdData, relayCell.Command())
	}
	
	// Verify data
	retrievedData := relayCell.Data()
	if string(retrievedData) != string(testData) {
		t.Errorf("Data mismatch. Expected: %s, Got: %s", testData, retrievedData)
	}
	
	// Verify initial state
	if relayCell.IsEnd() {
		t.Error("RelayCell should not be marked as end initially")
	}
}

func TestNewRelayCell_DataTooLarge(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	
	// Create data larger than MaxPayloadSize
	largeData := make([]byte, entity.MaxPayloadSize+1)
	
	_, err := NewRelayCell(value_object.CmdData, circuitID, streamID, largeData)
	if err == nil {
		t.Error("Expected NewRelayCell to fail with data too large")
	}
	
	expectedError := "data too large"
	if err != nil && len(err.Error()) > 0 && err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("Expected error message to start with '%s', got: %s", expectedError, err.Error())
	}
}

func TestNewRelayCell_EmptyData(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	
	relayCell, err := NewRelayCell(value_object.CmdConnect, circuitID, streamID, []byte{})
	if err != nil {
		t.Fatalf("NewRelayCell with empty data failed: %v", err)
	}
	
	if len(relayCell.Data()) != 0 {
		t.Error("Data should be empty")
	}
}

func TestRelayCell_DataImmutability(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	originalData := []byte("original data")
	
	relayCell, err := NewRelayCell(value_object.CmdData, circuitID, streamID, originalData)
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}
	
	// Get data and modify it
	retrievedData := relayCell.Data()
	retrievedData[0] = 'X'
	
	// Original data in RelayCell should be unchanged
	unchangedData := relayCell.Data()
	if string(unchangedData) != string(originalData) {
		t.Error("RelayCell data should be immutable")
	}
}

func TestRelayCell_IsDataCell(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	
	tests := []struct {
		name     string
		cmd      value_object.CellCommand
		expected bool
	}{
		{"Data cell", value_object.CmdData, true},
		{"Begin cell", value_object.CmdBegin, false},
		{"End cell", value_object.CmdEnd, false},
		{"Connect cell", value_object.CmdConnect, false},
		{"Extend cell", value_object.CmdExtend, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relayCell, err := NewRelayCell(tt.cmd, circuitID, streamID, []byte("test"))
			if err != nil {
				t.Fatalf("NewRelayCell failed: %v", err)
			}
			
			if relayCell.IsDataCell() != tt.expected {
				t.Errorf("IsDataCell() = %v, expected %v for command %d", relayCell.IsDataCell(), tt.expected, tt.cmd)
			}
		})
	}
}

func TestRelayCell_IsControlCell(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	
	tests := []struct {
		name     string
		cmd      value_object.CellCommand
		expected bool
	}{
		{"Begin cell", value_object.CmdBegin, true},
		{"End cell", value_object.CmdEnd, true},
		{"Connect cell", value_object.CmdConnect, true},
		{"Extend cell", value_object.CmdExtend, true},
		{"Destroy cell", value_object.CmdDestroy, true},
		{"Created cell", value_object.CmdCreated, true},
		{"BeginAck cell", value_object.CmdBeginAck, true},
		{"Data cell", value_object.CmdData, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relayCell, err := NewRelayCell(tt.cmd, circuitID, streamID, []byte("test"))
			if err != nil {
				t.Fatalf("NewRelayCell failed: %v", err)
			}
			
			if relayCell.IsControlCell() != tt.expected {
				t.Errorf("IsControlCell() = %v, expected %v for command %d", relayCell.IsControlCell(), tt.expected, tt.cmd)
			}
		})
	}
}

func TestRelayCell_EndMarking(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	
	relayCell, err := NewRelayCell(value_object.CmdData, circuitID, streamID, []byte("test"))
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}
	
	// Initially should not be marked as end
	if relayCell.IsEnd() {
		t.Error("RelayCell should not be marked as end initially")
	}
	
	// Mark as end
	relayCell.MarkEnd()
	
	// Should now be marked as end
	if !relayCell.IsEnd() {
		t.Error("RelayCell should be marked as end after MarkEnd()")
	}
}

func TestRelayCell_ValidateForCircuit_Success(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	
	relayCell, err := NewRelayCell(value_object.CmdData, circuitID, streamID, []byte("test"))
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}
	
	// Validation should succeed with the same circuit ID
	err = relayCell.ValidateForCircuit(circuitID)
	if err != nil {
		t.Errorf("ValidateForCircuit should succeed with same circuit ID: %v", err)
	}
}

func TestRelayCell_ValidateForCircuit_Mismatch(t *testing.T) {
	circuitID1 := value_object.NewCircuitID()
	circuitID2 := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	
	relayCell, err := NewRelayCell(value_object.CmdData, circuitID1, streamID, []byte("test"))
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}
	
	// Validation should fail with different circuit ID
	err = relayCell.ValidateForCircuit(circuitID2)
	if err == nil {
		t.Error("ValidateForCircuit should fail with different circuit ID")
	}
	
	expectedError := "circuit ID mismatch"
	if err != nil && len(err.Error()) > 0 && err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("Expected error message to start with '%s', got: %s", expectedError, err.Error())
	}
}

func TestRelayCell_Encode(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	testData := []byte("test payload for encoding")
	
	relayCell, err := NewRelayCell(value_object.CmdData, circuitID, streamID, testData)
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}
	
	encoded, err := relayCell.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	
	if len(encoded) == 0 {
		t.Error("Encoded data should not be empty")
	}
	
	// Encoded size should be the standard cell size
	if len(encoded) != entity.MaxCellSize {
		t.Errorf("Encoded size should be %d, got %d", entity.MaxCellSize, len(encoded))
	}
}

func TestRelayCell_Cell(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	testData := []byte("test payload")
	
	relayCell, err := NewRelayCell(value_object.CmdExtend, circuitID, streamID, testData)
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}
	
	cell := relayCell.Cell()
	
	// Verify the underlying cell properties
	if cell.Cmd != value_object.CmdExtend {
		t.Errorf("Cell command mismatch. Expected: %d, Got: %d", value_object.CmdExtend, cell.Cmd)
	}
	
	if cell.Version != value_object.ProtocolV1 {
		t.Errorf("Cell version mismatch. Expected: %d, Got: %d", value_object.ProtocolV1, cell.Version)
	}
	
	if string(cell.Payload) != string(testData) {
		t.Errorf("Cell payload mismatch. Expected: %s, Got: %s", testData, cell.Payload)
	}
}

func TestRelayCell_DifferentCommands(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	
	commands := []value_object.CellCommand{
		value_object.CmdData,
		value_object.CmdBegin,
		value_object.CmdEnd,
		value_object.CmdConnect,
		value_object.CmdExtend,
		value_object.CmdDestroy,
		value_object.CmdCreated,
		value_object.CmdBeginAck,
	}
	
	for _, cmd := range commands {
		t.Run(fmt.Sprintf("Command_%d", cmd), func(t *testing.T) {
			relayCell, err := NewRelayCell(cmd, circuitID, streamID, []byte("test"))
			if err != nil {
				t.Fatalf("NewRelayCell failed for command %d: %v", cmd, err)
			}
			
			if relayCell.Command() != cmd {
				t.Errorf("Command mismatch. Expected: %d, Got: %d", cmd, relayCell.Command())
			}
		})
	}
}

func TestRelayCell_MaxPayloadBoundary(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(1)
	
	// Test with exactly MaxPayloadSize
	maxData := make([]byte, entity.MaxPayloadSize)
	for i := range maxData {
		maxData[i] = byte(i % 256)
	}
	
	relayCell, err := NewRelayCell(value_object.CmdData, circuitID, streamID, maxData)
	if err != nil {
		t.Fatalf("NewRelayCell should succeed with exactly MaxPayloadSize: %v", err)
	}
	
	retrievedData := relayCell.Data()
	if len(retrievedData) != entity.MaxPayloadSize {
		t.Errorf("Data length mismatch. Expected: %d, Got: %d", entity.MaxPayloadSize, len(retrievedData))
	}
	
	// Verify data integrity
	for i, b := range retrievedData {
		if b != byte(i%256) {
			t.Errorf("Data corruption at index %d. Expected: %d, Got: %d", i, byte(i%256), b)
			break
		}
	}
}

func TestRelayCell_ZeroStreamID(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(0)
	
	relayCell, err := NewRelayCell(value_object.CmdConnect, circuitID, streamID, []byte("control message"))
	if err != nil {
		t.Fatalf("NewRelayCell with zero stream ID failed: %v", err)
	}
	
	if relayCell.StreamID().UInt16() != 0 {
		t.Errorf("Stream ID should be 0, got %d", relayCell.StreamID().UInt16())
	}
}

func TestRelayCell_EncodeDecodeRoundTrip(t *testing.T) {
	circuitID := value_object.NewCircuitID()
	streamID, _ := value_object.StreamIDFrom(42)
	originalData := []byte("round trip test data")
	
	// Create original relay cell
	originalCell, err := NewRelayCell(value_object.CmdData, circuitID, streamID, originalData)
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}
	
	// Encode
	encoded, err := originalCell.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	
	// Decode back to entity.Cell
	decodedCell, err := entity.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	
	// Verify the decoded cell matches original
	if decodedCell.Cmd != originalCell.Command() {
		t.Errorf("Command mismatch after round trip. Expected: %d, Got: %d", originalCell.Command(), decodedCell.Cmd)
	}
	
	if decodedCell.Version != value_object.ProtocolV1 {
		t.Errorf("Version mismatch after round trip. Expected: %d, Got: %d", value_object.ProtocolV1, decodedCell.Version)
	}
	
	if string(decodedCell.Payload) != string(originalData) {
		t.Errorf("Payload mismatch after round trip. Expected: %s, Got: %s", originalData, decodedCell.Payload)
	}
}