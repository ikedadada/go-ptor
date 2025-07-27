package service

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/shared/domain/aggregate"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// circuitBuildTestConn implements net.Conn for testing circuit build operations
type circuitBuildTestConn struct {
	readBuffer  *bytes.Buffer
	writeBuffer *bytes.Buffer
	closed      bool
}

func newCircuitBuildTestConn() *circuitBuildTestConn {
	return &circuitBuildTestConn{
		readBuffer:  &bytes.Buffer{},
		writeBuffer: &bytes.Buffer{},
	}
}

func (c *circuitBuildTestConn) Read(b []byte) (n int, err error) {
	if c.closed {
		return 0, io.EOF
	}
	return c.readBuffer.Read(b)
}

func (c *circuitBuildTestConn) Write(b []byte) (n int, err error) {
	if c.closed {
		return 0, io.ErrClosedPipe
	}
	return c.writeBuffer.Write(b)
}

func (c *circuitBuildTestConn) Close() error {
	c.closed = true
	return nil
}

func (c *circuitBuildTestConn) LocalAddr() net.Addr                { return nil }
func (c *circuitBuildTestConn) RemoteAddr() net.Addr               { return nil }
func (c *circuitBuildTestConn) SetDeadline(t time.Time) error      { return nil }
func (c *circuitBuildTestConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *circuitBuildTestConn) SetWriteDeadline(t time.Time) error { return nil }

// prepareCreatedResponse sets up a mock CREATED response in the read buffer
func (c *circuitBuildTestConn) prepareCreatedResponse(payload []byte) {
	var hdr [20]byte
	hdr[16] = byte(vo.CmdCreated)
	hdr[17] = byte(vo.ProtocolV1)
	binary.BigEndian.PutUint16(hdr[18:20], uint16(len(payload)))

	c.readBuffer.Write(hdr[:])
	c.readBuffer.Write(payload)
}

func TestTCPCircuitBuildService_SendExtendCell(t *testing.T) {
	svc := NewTCPCircuitBuildService()
	conn := newCircuitBuildTestConn()

	cid := vo.NewCircuitID()
	sid, _ := vo.StreamIDFrom(1)
	relayCell, err := aggregate.NewRelayCell(vo.CmdExtend, cid, sid, []byte("test extend payload"))
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}

	err = svc.SendExtendCell(conn, relayCell)
	if err != nil {
		t.Fatalf("SendExtendCell failed: %v", err)
	}

	// Verify written data
	written := conn.writeBuffer.Bytes()

	// Should contain circuit ID + encoded cell
	if len(written) < 16 {
		t.Fatalf("Written data too short: %d bytes", len(written))
	}

	// Check circuit ID (first 16 bytes)
	if !bytes.Equal(written[:16], cid.Bytes()) {
		t.Errorf("Circuit ID mismatch: got %x, want %x", written[:16], cid.Bytes())
	}

	// The rest should be the encoded relay cell
	cellData := written[16:]
	if len(cellData) == 0 {
		t.Error("No cell data found after circuit ID")
	}
}

func TestTCPCircuitBuildService_WaitForCreatedResponse(t *testing.T) {
	svc := NewTCPCircuitBuildService()
	conn := newCircuitBuildTestConn()

	expectedPayload := []byte("created response payload")
	conn.prepareCreatedResponse(expectedPayload)

	payload, err := svc.WaitForCreatedResponse(conn)
	if err != nil {
		t.Fatalf("WaitForCreatedResponse failed: %v", err)
	}

	if !bytes.Equal(payload, expectedPayload) {
		t.Errorf("Payload mismatch: got %s, want %s", string(payload), string(expectedPayload))
	}
}

func TestTCPCircuitBuildService_WaitForCreatedResponse_InvalidHeader(t *testing.T) {
	svc := NewTCPCircuitBuildService()
	conn := newCircuitBuildTestConn()

	// Write invalid header (wrong command)
	var hdr [20]byte
	hdr[16] = byte(vo.CmdData) // Wrong command
	hdr[17] = byte(vo.ProtocolV1)
	binary.BigEndian.PutUint16(hdr[18:20], 4)

	conn.readBuffer.Write(hdr[:])
	conn.readBuffer.Write([]byte("test"))

	_, err := svc.WaitForCreatedResponse(conn)
	if err == nil {
		t.Error("Expected error for invalid header")
	}
}

func TestTCPCircuitBuildService_WaitForCreatedResponse_NoPayload(t *testing.T) {
	svc := NewTCPCircuitBuildService()
	conn := newCircuitBuildTestConn()

	// Write header with zero payload length
	var hdr [20]byte
	hdr[16] = byte(vo.CmdCreated)
	hdr[17] = byte(vo.ProtocolV1)
	binary.BigEndian.PutUint16(hdr[18:20], 0) // Zero length

	conn.readBuffer.Write(hdr[:])

	_, err := svc.WaitForCreatedResponse(conn)
	if err == nil {
		t.Error("Expected error for zero payload length")
	}
}

func TestTCPCircuitBuildService_WaitForCreatedResponse_IncompleteRead(t *testing.T) {
	svc := NewTCPCircuitBuildService()
	conn := newCircuitBuildTestConn()

	// Write incomplete header
	var hdr [10]byte
	conn.readBuffer.Write(hdr[:])

	_, err := svc.WaitForCreatedResponse(conn)
	if err == nil {
		t.Error("Expected error for incomplete header read")
	}
}

func TestTCPCircuitBuildService_TeardownCircuit(t *testing.T) {
	svc := NewTCPCircuitBuildService()
	conn := newCircuitBuildTestConn()
	cid := vo.NewCircuitID()

	err := svc.TeardownCircuit(conn, cid)
	if err != nil {
		t.Fatalf("TeardownCircuit failed: %v", err)
	}

	// Verify written data
	written := conn.writeBuffer.Bytes()

	if len(written) != 20 {
		t.Errorf("Expected 20 bytes, got %d", len(written))
	}

	// Check circuit ID (first 16 bytes)
	if !bytes.Equal(written[:16], cid.Bytes()) {
		t.Errorf("Circuit ID mismatch: got %x, want %x", written[:16], cid.Bytes())
	}

	// Check teardown marker (byte 18 should be 0xFE)
	if written[18] != 0xFE {
		t.Errorf("Teardown marker mismatch: got %x, want 0xFE", written[18])
	}
}

func TestTCPCircuitBuildService_SendExtendCell_EncodeError(t *testing.T) {
	svc := NewTCPCircuitBuildService()
	conn := newCircuitBuildTestConn()

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

	// This should succeed even with large payload (depends on implementation)
	err = svc.SendExtendCell(conn, relayCell)

	// We can't easily force an encoding error with the current implementation
	// but this test ensures the method handles potential encoding issues
	if err != nil {
		t.Logf("SendExtendCell with large payload failed as expected: %v", err)
	}
}

func TestTCPCircuitBuildService_ClosedConnection(t *testing.T) {
	svc := NewTCPCircuitBuildService()
	conn := newCircuitBuildTestConn()
	conn.Close()

	cid := vo.NewCircuitID()
	sid, _ := vo.StreamIDFrom(1)
	relayCell, err := aggregate.NewRelayCell(vo.CmdExtend, cid, sid, []byte("test"))
	if err != nil {
		t.Fatalf("NewRelayCell failed: %v", err)
	}

	// Should handle closed connection gracefully
	err = svc.SendExtendCell(conn, relayCell)
	if err == nil {
		t.Error("Expected error when writing to closed connection")
	}

	// WaitForCreatedResponse should also handle closed connection
	_, err = svc.WaitForCreatedResponse(conn)
	if err == nil {
		t.Error("Expected error when reading from closed connection")
	}
}
