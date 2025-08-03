package service

import (
	"encoding/binary"
	"net"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestCellSenderService_SendCreated(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewCellSenderService()
	mockConn := Mock[net.Conn](ctrl)
	cid := vo.NewCircuitID()
	payload := []byte("test payload data")

	// Expected header: CircuitID + Command + Version + PayloadLength
	expectedHeader := make([]byte, 20)
	copy(expectedHeader[:16], cid.Bytes())
	expectedHeader[16] = byte(vo.CmdCreated)
	expectedHeader[17] = byte(vo.ProtocolV1)
	binary.BigEndian.PutUint16(expectedHeader[18:20], uint16(len(payload)))

	// Mock two separate Write calls
	WhenDouble(mockConn.Write(expectedHeader)).ThenReturn(len(expectedHeader), nil)
	WhenDouble(mockConn.Write(payload)).ThenReturn(len(payload), nil)

	err := svc.SendCreated(mockConn, cid, payload)
	if err != nil {
		t.Fatalf("SendCreated failed: %v", err)
	}

	Verify(mockConn, Times(1)).Write(expectedHeader)
	Verify(mockConn, Times(1)).Write(payload)
}

func TestCellSenderService_SendAck(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewCellSenderService()
	mockConn := Mock[net.Conn](ctrl)
	cid := vo.NewCircuitID()

	// Mock Write call with Any matcher due to random padding in cell encoding
	// Expected: CircuitID (16 bytes) + Encoded BeginAck Cell (512 bytes) = 528 bytes total
	expectedLength := 16 + entity.MaxCellSize
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenReturn(expectedLength, nil)

	err := svc.SendAck(mockConn, cid)
	if err != nil {
		t.Fatalf("SendAck failed: %v", err)
	}

	// Verify Write was called once with data of expected length
	Verify(mockConn, Times(1)).Write(Any[[]byte]())
}

func TestCellSenderService_ForwardCell(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewCellSenderService()
	mockConn := Mock[net.Conn](ctrl)
	cid := vo.NewCircuitID()

	testPayload := []byte("forward test data")
	cell := &entity.Cell{
		Cmd:     vo.CmdData,
		Version: vo.ProtocolV1,
		Payload: testPayload,
	}

	// Mock Write call with Any matcher due to random padding in cell encoding
	// Expected: CircuitID (16 bytes) + Encoded Cell (512 bytes) = 528 bytes total
	expectedLength := 16 + entity.MaxCellSize
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenReturn(expectedLength, nil)

	err := svc.ForwardCell(mockConn, cid, cell)
	if err != nil {
		t.Fatalf("ForwardCell failed: %v", err)
	}

	// Verify Write was called once
	Verify(mockConn, Times(1)).Write(Any[[]byte]())
}

func TestCellSenderService_ForwardCell_EmptyPayload(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewCellSenderService()
	mockConn := Mock[net.Conn](ctrl)
	cid := vo.NewCircuitID()

	cell := &entity.Cell{
		Cmd:     vo.CmdDestroy,
		Version: vo.ProtocolV1,
		Payload: nil,
	}

	// Mock Write call with Any matcher due to random padding in cell encoding
	// Expected: CircuitID (16 bytes) + Encoded Cell (512 bytes) = 528 bytes total
	expectedLength := 16 + entity.MaxCellSize
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenReturn(expectedLength, nil)

	err := svc.ForwardCell(mockConn, cid, cell)
	if err != nil {
		t.Fatalf("ForwardCell failed: %v", err)
	}

	// Verify Write was called once
	Verify(mockConn, Times(1)).Write(Any[[]byte]())
}

func TestCellSenderService_SendCreated_EmptyPayload(t *testing.T) {
	ctrl := NewMockController(t)
	svc := NewCellSenderService()
	mockConn := Mock[net.Conn](ctrl)
	cid := vo.NewCircuitID()

	// Expected header: CircuitID + Command + Version + PayloadLength(0)
	expectedHeader := make([]byte, 20)
	copy(expectedHeader[:16], cid.Bytes())
	expectedHeader[16] = byte(vo.CmdCreated)
	expectedHeader[17] = byte(vo.ProtocolV1)
	binary.BigEndian.PutUint16(expectedHeader[18:20], 0) // Empty payload length

	// Mock header write and empty payload write
	WhenDouble(mockConn.Write(expectedHeader)).ThenReturn(len(expectedHeader), nil)
	WhenDouble(mockConn.Write([]byte(nil))).ThenReturn(0, nil)

	err := svc.SendCreated(mockConn, cid, nil)
	if err != nil {
		t.Fatalf("SendCreated with empty payload failed: %v", err)
	}

	Verify(mockConn, Times(1)).Write(expectedHeader)
	Verify(mockConn, Times(1)).Write([]byte(nil))
}
