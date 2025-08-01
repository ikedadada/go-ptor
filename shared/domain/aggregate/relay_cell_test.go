package aggregate

import (
	"testing"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestNewRelayCell_Success(t *testing.T) {
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)
	testData := []byte("test payload")

	relayCell, err := NewRelayCell(vo.CmdData, circuitID, streamID, testData)
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
	if relayCell.Command() != vo.CmdData {
		t.Errorf("Command mismatch. Expected: %d, Got: %d", vo.CmdData, relayCell.Command())
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
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)

	// Create data larger than MaxPayloadSize
	largeData := make([]byte, entity.MaxPayloadSize+1)

	_, err := NewRelayCell(vo.CmdData, circuitID, streamID, largeData)
	if err == nil {
		t.Error("Expected NewRelayCell to fail with data too large")
	}

	expectedError := "data too large"
	if err != nil && len(err.Error()) > 0 && err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("Expected error message to start with '%s', got: %s", expectedError, err.Error())
	}
}

func TestNewRelayCell_EmptyData(t *testing.T) {
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)

	relayCell, err := NewRelayCell(vo.CmdConnect, circuitID, streamID, []byte{})
	if err != nil {
		t.Fatalf("NewRelayCell with empty data failed: %v", err)
	}

	if len(relayCell.Data()) != 0 {
		t.Error("Data should be empty")
	}
}

func TestRelayCell_DataImmutability(t *testing.T) {
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)
	originalData := []byte("original data")

	relayCell, err := NewRelayCell(vo.CmdData, circuitID, streamID, originalData)
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
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)

	tests := []struct {
		name     string
		cmd      vo.CellCommand
		expected bool
	}{
		{"Data cell", vo.CmdData, true},
		{"Begin cell", vo.CmdBegin, false},
		{"End cell", vo.CmdEnd, false},
		{"Connect cell", vo.CmdConnect, false},
		{"Extend cell", vo.CmdExtend, false},
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
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)

	tests := []struct {
		name     string
		cmd      vo.CellCommand
		expected bool
	}{
		{"Begin cell", vo.CmdBegin, true},
		{"End cell", vo.CmdEnd, true},
		{"Connect cell", vo.CmdConnect, true},
		{"Extend cell", vo.CmdExtend, true},
		{"Destroy cell", vo.CmdDestroy, true},
		{"Created cell", vo.CmdCreated, true},
		{"BeginAck cell", vo.CmdBeginAck, true},
		{"Data cell", vo.CmdData, false},
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
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)

	relayCell, err := NewRelayCell(vo.CmdData, circuitID, streamID, []byte("test"))
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
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)

	relayCell, err := NewRelayCell(vo.CmdData, circuitID, streamID, []byte("test"))
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
	circuitID1 := vo.NewCircuitID()
	circuitID2 := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)

	relayCell, err := NewRelayCell(vo.CmdData, circuitID1, streamID, []byte("test"))
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
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)
	testData := []byte("test payload for encoding")

	relayCell, err := NewRelayCell(vo.CmdData, circuitID, streamID, testData)
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
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)
	testData := []byte("test payload")

	relayCell, err := NewRelayCell(vo.CmdExtend, circuitID, streamID, testData)
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}

	cell := relayCell.Cell()

	// Verify the underlying cell properties
	if cell.Cmd != vo.CmdExtend {
		t.Errorf("Cell command mismatch. Expected: %d, Got: %d", vo.CmdExtend, cell.Cmd)
	}

	if cell.Version != vo.ProtocolV1 {
		t.Errorf("Cell version mismatch. Expected: %d, Got: %d", vo.ProtocolV1, cell.Version)
	}

	if string(cell.Payload) != string(testData) {
		t.Errorf("Cell payload mismatch. Expected: %s, Got: %s", testData, cell.Payload)
	}
}

func TestRelayCell_DifferentCommands(t *testing.T) {
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)

	tests := []struct {
		name string
		cmd  vo.CellCommand
	}{
		{"DATA command", vo.CmdData},
		{"BEGIN command", vo.CmdBegin},
		{"END command", vo.CmdEnd},
		{"CONNECT command", vo.CmdConnect},
		{"EXTEND command", vo.CmdExtend},
		{"DESTROY command", vo.CmdDestroy},
		{"CREATED command", vo.CmdCreated},
		{"BEGIN_ACK command", vo.CmdBeginAck},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			relayCell, err := NewRelayCell(test.cmd, circuitID, streamID, []byte("test"))
			if err != nil {
				t.Fatalf("NewRelayCell failed for command %d: %v", test.cmd, err)
			}

			if relayCell.Command() != test.cmd {
				t.Errorf("Command mismatch. Expected: %d, Got: %d", test.cmd, relayCell.Command())
			}
		})
	}
}

func TestRelayCell_MaxPayloadBoundary(t *testing.T) {
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(1)

	// Test with exactly MaxPayloadSize
	maxData := make([]byte, entity.MaxPayloadSize)
	for i := range maxData {
		maxData[i] = byte(i % 256)
	}

	relayCell, err := NewRelayCell(vo.CmdData, circuitID, streamID, maxData)
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
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(0)

	relayCell, err := NewRelayCell(vo.CmdConnect, circuitID, streamID, []byte("control message"))
	if err != nil {
		t.Fatalf("NewRelayCell with zero stream ID failed: %v", err)
	}

	if relayCell.StreamID().UInt16() != 0 {
		t.Errorf("Stream ID should be 0, got %d", relayCell.StreamID().UInt16())
	}
}

func TestRelayCell_EncodeDecodeRoundTrip(t *testing.T) {
	circuitID := vo.NewCircuitID()
	streamID, _ := vo.StreamIDFrom(42)
	originalData := []byte("round trip test data")

	// Create original relay cell
	originalCell, err := NewRelayCell(vo.CmdData, circuitID, streamID, originalData)
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

	if decodedCell.Version != vo.ProtocolV1 {
		t.Errorf("Version mismatch after round trip. Expected: %d, Got: %d", vo.ProtocolV1, decodedCell.Version)
	}

	if string(decodedCell.Payload) != string(originalData) {
		t.Errorf("Payload mismatch after round trip. Expected: %s, Got: %s", originalData, decodedCell.Payload)
	}
}
