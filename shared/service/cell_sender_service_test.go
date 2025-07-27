package service

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// cellSenderTestConn implements net.Conn for testing cell sender write operations
type cellSenderTestConn struct {
	buffer *bytes.Buffer
}

func newCellSenderTestConn() *cellSenderTestConn {
	return &cellSenderTestConn{buffer: &bytes.Buffer{}}
}

func (m *cellSenderTestConn) Read(b []byte) (n int, err error) {
	return m.buffer.Read(b)
}

func (m *cellSenderTestConn) Write(b []byte) (n int, err error) {
	return m.buffer.Write(b)
}

func (m *cellSenderTestConn) Close() error                       { return nil }
func (m *cellSenderTestConn) LocalAddr() net.Addr                { return nil }
func (m *cellSenderTestConn) RemoteAddr() net.Addr               { return nil }
func (m *cellSenderTestConn) SetDeadline(t time.Time) error      { return nil }
func (m *cellSenderTestConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *cellSenderTestConn) SetWriteDeadline(t time.Time) error { return nil }

func TestCellSenderService_SendCreated(t *testing.T) {
	svc := NewCellSenderService()
	conn := newCellSenderTestConn()
	cid := vo.NewCircuitID()
	payload := []byte("test payload data")

	err := svc.SendCreated(conn, cid, payload)
	if err != nil {
		t.Fatalf("SendCreated failed: %v", err)
	}

	// Verify the written data
	written := conn.buffer.Bytes()

	// Check circuit ID (first 16 bytes)
	if !bytes.Equal(written[:16], cid.Bytes()) {
		t.Errorf("Circuit ID mismatch: got %x, want %x", written[:16], cid.Bytes())
	}

	// Check command (byte 16)
	if written[16] != byte(vo.CmdCreated) {
		t.Errorf("Command mismatch: got %d, want %d", written[16], vo.CmdCreated)
	}

	// Check version (byte 17)
	if written[17] != byte(vo.ProtocolV1) {
		t.Errorf("Version mismatch: got %d, want %d", written[17], vo.ProtocolV1)
	}

	// Check payload length (bytes 18-19)
	payloadLen := binary.BigEndian.Uint16(written[18:20])
	if payloadLen != uint16(len(payload)) {
		t.Errorf("Payload length mismatch: got %d, want %d", payloadLen, len(payload))
	}

	// Check payload (bytes 20+)
	if !bytes.Equal(written[20:], payload) {
		t.Errorf("Payload mismatch: got %s, want %s", string(written[20:]), string(payload))
	}
}

func TestCellSenderService_SendAck(t *testing.T) {
	svc := NewCellSenderService()
	conn := newCellSenderTestConn()
	cid := vo.NewCircuitID()

	err := svc.SendAck(conn, cid)
	if err != nil {
		t.Fatalf("SendAck failed: %v", err)
	}

	// Verify the written data
	written := conn.buffer.Bytes()

	// Should be circuit ID + encoded cell
	expectedSize := 16 + entity.MaxCellSize // Circuit ID + Cell
	if len(written) != expectedSize {
		t.Errorf("Written data length mismatch: got %d, want %d", len(written), expectedSize)
	}

	// Check circuit ID (first 16 bytes)
	if !bytes.Equal(written[:16], cid.Bytes()) {
		t.Errorf("Circuit ID mismatch: got %x, want %x", written[:16], cid.Bytes())
	}

	// Decode the cell part
	cellData := written[16:]
	cell, err := entity.Decode(cellData)
	if err != nil {
		t.Fatalf("Failed to decode cell: %v", err)
	}

	if cell.Cmd != vo.CmdBeginAck {
		t.Errorf("Command mismatch: got %d, want %d", cell.Cmd, vo.CmdBeginAck)
	}
	if cell.Version != vo.ProtocolV1 {
		t.Errorf("Version mismatch: got %d, want %d", cell.Version, vo.ProtocolV1)
	}
}

func TestCellSenderService_ForwardCell(t *testing.T) {
	svc := NewCellSenderService()
	conn := newCellSenderTestConn()
	cid := vo.NewCircuitID()

	testPayload := []byte("forward test data")
	cell := &entity.Cell{
		Cmd:     vo.CmdData,
		Version: vo.ProtocolV1,
		Payload: testPayload,
	}

	err := svc.ForwardCell(conn, cid, cell)
	if err != nil {
		t.Fatalf("ForwardCell failed: %v", err)
	}

	// Verify the written data
	written := conn.buffer.Bytes()

	// Should be circuit ID + encoded cell
	expectedSize := 16 + entity.MaxCellSize
	if len(written) != expectedSize {
		t.Errorf("Written data length mismatch: got %d, want %d", len(written), expectedSize)
	}

	// Check circuit ID (first 16 bytes)
	if !bytes.Equal(written[:16], cid.Bytes()) {
		t.Errorf("Circuit ID mismatch: got %x, want %x", written[:16], cid.Bytes())
	}

	// Decode the cell part
	cellData := written[16:]
	decodedCell, err := entity.Decode(cellData)
	if err != nil {
		t.Fatalf("Failed to decode cell: %v", err)
	}

	if decodedCell.Cmd != cell.Cmd {
		t.Errorf("Command mismatch: got %d, want %d", decodedCell.Cmd, cell.Cmd)
	}
	if decodedCell.Version != cell.Version {
		t.Errorf("Version mismatch: got %d, want %d", decodedCell.Version, cell.Version)
	}
	if !bytes.Equal(decodedCell.Payload[:len(testPayload)], testPayload) {
		t.Errorf("Payload mismatch: got %v, want %v", decodedCell.Payload[:len(testPayload)], testPayload)
	}
}

func TestCellSenderService_ForwardCell_EmptyPayload(t *testing.T) {
	svc := NewCellSenderService()
	conn := newCellSenderTestConn()
	cid := vo.NewCircuitID()

	cell := &entity.Cell{
		Cmd:     vo.CmdDestroy,
		Version: vo.ProtocolV1,
		Payload: nil,
	}

	err := svc.ForwardCell(conn, cid, cell)
	if err != nil {
		t.Fatalf("ForwardCell failed: %v", err)
	}

	// Verify the written data contains circuit ID and encoded cell
	written := conn.buffer.Bytes()
	if len(written) != 16+entity.MaxCellSize {
		t.Errorf("Expected %d bytes, got %d", 16+entity.MaxCellSize, len(written))
	}
}

func TestCellSenderService_SendCreated_EmptyPayload(t *testing.T) {
	svc := NewCellSenderService()
	conn := newCellSenderTestConn()
	cid := vo.NewCircuitID()

	err := svc.SendCreated(conn, cid, nil)
	if err != nil {
		t.Fatalf("SendCreated with empty payload failed: %v", err)
	}

	// Verify the written data
	written := conn.buffer.Bytes()

	// Should have 20 bytes total (16 for circuit ID + 4 for header)
	if len(written) != 20 {
		t.Errorf("Expected 20 bytes, got %d", len(written))
	}

	// Check payload length is 0
	payloadLen := binary.BigEndian.Uint16(written[18:20])
	if payloadLen != 0 {
		t.Errorf("Expected payload length 0, got %d", payloadLen)
	}
}
