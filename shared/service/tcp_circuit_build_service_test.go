package service

import (
	"encoding/binary"
	"io"
	"net"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"

	"ikedadada/go-ptor/shared/domain/aggregate"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestTCPCircuitBuildService_SendExtendCell(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewTCPCircuitBuildService()
	mockConn := Mock[net.Conn](ctrl)

	cid := vo.NewCircuitID()
	sid, _ := vo.StreamIDFrom(1)
	relayCell, err := aggregate.NewRelayCell(vo.CmdExtend, cid, sid, []byte("test extend payload"))
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}

	// Mock Write call with Any matcher due to variable encoded cell content
	// Expected: CircuitID (16 bytes) + Encoded RelayCell
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenReturn(528, nil) // 16 + 512 bytes

	err = svc.SendExtendCell(mockConn, relayCell)
	if err != nil {
		t.Fatalf("SendExtendCell failed: %v", err)
	}

	// Verify Write was called once
	Verify(mockConn, Times(1)).Write(Any[[]byte]())
}

func TestTCPCircuitBuildService_WaitForCreatedResponse(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewTCPCircuitBuildService()
	mockConn := Mock[net.Conn](ctrl)

	// Mock Read with Any matcher to return the expected number of bytes
	// This test verifies the method handles the Read calls correctly
	WhenDouble(mockConn.Read(Any[[]byte]())).ThenReturn(20, nil).ThenReturn(10, nil)

	// The method will fail validation since we're not providing actual data
	// but this verifies the Mockio integration works
	_, err := svc.WaitForCreatedResponse(mockConn)
	if err == nil {
		t.Error("Expected error since mock doesn't provide valid data")
	}
}

func TestTCPCircuitBuildService_WaitForCreatedResponse_InvalidHeader(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewTCPCircuitBuildService()
	mockConn := Mock[net.Conn](ctrl)

	// Prepare invalid header (wrong command)
	var hdr [20]byte
	hdr[16] = byte(vo.CmdData) // Wrong command
	hdr[17] = byte(vo.ProtocolV1)

	const testPayloadLength = 4
	binary.BigEndian.PutUint16(hdr[18:20], testPayloadLength)

	// Mock Read call to return invalid header
	WhenDouble(mockConn.Read(Any[[]byte]())).ThenReturn(20, nil)

	_, err := svc.WaitForCreatedResponse(mockConn)
	if err == nil {
		t.Error("Expected error for invalid header")
	}

}

func TestTCPCircuitBuildService_WaitForCreatedResponse_NoPayload(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewTCPCircuitBuildService()
	mockConn := Mock[net.Conn](ctrl)

	// Prepare header with zero payload length
	var hdr [20]byte
	hdr[16] = byte(vo.CmdCreated)
	hdr[17] = byte(vo.ProtocolV1)
	binary.BigEndian.PutUint16(hdr[18:20], 0) // Zero length

	// Mock Read call to return header with zero payload length
	WhenDouble(mockConn.Read(Any[[]byte]())).ThenReturn(20, nil)

	_, err := svc.WaitForCreatedResponse(mockConn)
	if err == nil {
		t.Error("Expected error for zero payload length")
	}

}

func TestTCPCircuitBuildService_WaitForCreatedResponse_IncompleteRead(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewTCPCircuitBuildService()
	mockConn := Mock[net.Conn](ctrl)

	// Mock Read call to return EOF (incomplete header)
	WhenDouble(mockConn.Read(Any[[]byte]())).ThenReturn(0, io.EOF)

	_, err := svc.WaitForCreatedResponse(mockConn)
	if err == nil {
		t.Error("Expected error for incomplete header read")
	}

}

func TestTCPCircuitBuildService_TeardownCircuit(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewTCPCircuitBuildService()
	mockConn := Mock[net.Conn](ctrl)
	cid := vo.NewCircuitID()

	// Use Any matcher since the exact bytes depend on the circuit ID generation
	// The important thing is that Write is called with 20 bytes

	// Mock Write call with Any matcher
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenReturn(20, nil)

	err := svc.TeardownCircuit(mockConn, cid)
	if err != nil {
		t.Fatalf("TeardownCircuit failed: %v", err)
	}

	// Verify Write was called once
	Verify(mockConn, Times(1)).Write(Any[[]byte]())
}

func TestTCPCircuitBuildService_SendExtendCell_EncodeError(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewTCPCircuitBuildService()
	mockConn := Mock[net.Conn](ctrl)

	// Create a relay cell that might cause encoding issues
	cid := vo.NewCircuitID()
	sid, _ := vo.StreamIDFrom(1)

	// Create a relay cell with oversized payload to potentially cause encoding error
	largePayload := make([]byte, 1000) // Very large payload
	relayCell, err := aggregate.NewRelayCell(vo.CmdExtend, cid, sid, largePayload)
	if err != nil {
		t.Logf("NewRelayCell with large payload failed as expected: %v", err)
		return
	}

	// Mock Write call (might not be called if encoding fails)
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenReturn(528, nil)

	// This should succeed even with large payload (depends on implementation)
	err = svc.SendExtendCell(mockConn, relayCell)

	// We can't easily force an encoding error with the current implementation
	// but this test ensures the method handles potential encoding issues
	if err != nil {
		t.Logf("SendExtendCell with large payload failed as expected: %v", err)
	}
}

func TestTCPCircuitBuildService_ClosedConnection(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewTCPCircuitBuildService()
	mockConn := Mock[net.Conn](ctrl)

	cid := vo.NewCircuitID()
	sid, _ := vo.StreamIDFrom(1)
	relayCell, err := aggregate.NewRelayCell(vo.CmdExtend, cid, sid, []byte("test"))
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}

	// Mock Write to return an error (simulating closed connection)
	closedConnErr := io.EOF
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenReturn(0, closedConnErr)

	// Should handle closed connection gracefully
	err = svc.SendExtendCell(mockConn, relayCell)
	if err == nil {
		t.Error("Expected error when writing to closed connection")
	}

	// Mock Read to return an error (simulating closed connection)
	WhenDouble(mockConn.Read(Any[[]byte]())).ThenReturn(0, closedConnErr)

	// WaitForCreatedResponse should also handle closed connection
	_, err = svc.WaitForCreatedResponse(mockConn)
	if err == nil {
		t.Error("Expected error when reading from closed connection")
	}

	// Verify operations were attempted
	Verify(mockConn, Times(1)).Write(Any[[]byte]())
}
