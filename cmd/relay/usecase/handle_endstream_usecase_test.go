package usecase_test

import (
	"errors"
	"net"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"

	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestHandleEndStreamUseCase_EndStreamSpecific(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleEndStreamUseCase(mockRepo, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	endPayload := []byte("end-payload")

	// Create mock connection (exit node - no down connection)
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: endPayload}

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Mock data payload DTO for specific stream
	sid, _ := vo.StreamIDFrom(1)
	dataDTO := &service.DataPayloadDTO{StreamID: 1}

	// Configure mocks for specific stream termination (st.Down() == nil)
	WhenDouble(mockEncoder.DecodeDataPayload(endPayload)).ThenReturn(dataDTO, nil)
	WhenSingle(mockRepo.RemoveStream(cid, sid)).ThenReturn(nil)

	// Execute
	err := uc.EndStream(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("EndStream failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockEncoder, Times(1)).DecodeDataPayload(endPayload)
	Verify(mockRepo, Times(1)).RemoveStream(cid, sid)
	// Verify ForwardCell is not called for exit nodes
	Verify(mockSender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
}

func TestHandleEndStreamUseCase_EndAllStreams(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleEndStreamUseCase(mockRepo, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()

	// Create mock connection (exit node - no down connection)
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	// Empty payload means end all streams (StreamID = 0)
	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: []byte{}}

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Configure mocks for ending all streams (empty payload)
	WhenSingle(mockRepo.Delete(cid)).ThenReturn(nil)

	// Execute
	err := uc.EndStream(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("EndStream failed: %v", err)
	}

	// Verify mock interactions
	// No payload decoding for empty payload
	Verify(mockEncoder, Times(0)).DecodeDataPayload(Any[[]byte]())
	Verify(mockRepo, Times(1)).Delete(cid)
	// Verify ForwardCell is not called for exit nodes
	Verify(mockSender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
}

func TestHandleEndStreamUseCase_EndStreamWithForward(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleEndStreamUseCase(mockRepo, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	endPayload := []byte("end-payload")

	// Create mock connections (middle relay - has down connection)
	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)

	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: endPayload}

	// Mock ensureServeDown function
	serveDownCalled := false
	ensureServeDown := func(st *entity.ConnState) {
		serveDownCalled = true
	}

	// Mock data payload DTO for specific stream
	sid, _ := vo.StreamIDFrom(1)
	dataDTO := &service.DataPayloadDTO{StreamID: 1}

	// Configure mocks for middle relay scenario (st.Down() != nil)
	WhenDouble(mockEncoder.DecodeDataPayload(endPayload)).ThenReturn(dataDTO, nil)
	WhenSingle(mockRepo.RemoveStream(cid, sid)).ThenReturn(nil)
	WhenSingle(mockSender.ForwardCell(st.Down(), cid, cell)).ThenReturn(nil)

	// Execute
	err := uc.EndStream(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("EndStream failed: %v", err)
	}

	if !serveDownCalled {
		t.Errorf("ensureServeDown not called")
	}

	// Verify mock interactions
	Verify(mockEncoder, Times(1)).DecodeDataPayload(endPayload)
	Verify(mockRepo, Times(1)).RemoveStream(cid, sid)
	Verify(mockSender, Times(1)).ForwardCell(st.Down(), cid, cell)

	st.Up().Close()
	st.Down().Close()
}

// Test payload decoding failure
func TestHandleEndStreamUseCase_PayloadDecodingFailure(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleEndStreamUseCase(mockRepo, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	invalidPayload := []byte("invalid-payload")

	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: invalidPayload}

	ensureServeDown := func(st *entity.ConnState) {}

	// Configure mocks
	decodingError := errors.New("payload decoding failed")
	WhenDouble(mockEncoder.DecodeDataPayload(invalidPayload)).ThenReturn(nil, decodingError)

	// Execute
	err := uc.EndStream(st, cid, cell, ensureServeDown)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, decodingError) {
		t.Fatalf("Expected decoding error, got: %v", err)
	}

	// Verify interactions
	Verify(mockEncoder, Times(1)).DecodeDataPayload(invalidPayload)
	Verify(mockRepo, Times(0)).RemoveStream(Any[vo.CircuitID](), Any[vo.StreamID]())

	st.Up().Close()
}

// Test ending all streams with downstream connection
func TestHandleEndStreamUseCase_EndAllStreamsWithForward(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleEndStreamUseCase(mockRepo, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()

	// Create mock connections (middle relay - has down connection)
	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)

	// Empty payload means end all streams (StreamID = 0)
	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: []byte{}}

	// Mock ensureServeDown function
	serveDownCalled := false
	ensureServeDown := func(st *entity.ConnState) {
		serveDownCalled = true
	}

	// Configure mocks for ending all streams with downstream forwarding
	WhenSingle(mockSender.ForwardCell(st.Down(), cid, cell)).ThenReturn(nil)
	WhenSingle(mockRepo.Delete(cid)).ThenReturn(nil)

	// Execute
	err := uc.EndStream(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("EndStream failed: %v", err)
	}

	if !serveDownCalled {
		t.Errorf("ensureServeDown not called")
	}

	// Verify mock interactions
	Verify(mockEncoder, Times(0)).DecodeDataPayload(Any[[]byte]())
	Verify(mockSender, Times(1)).ForwardCell(st.Down(), cid, cell)
	Verify(mockRepo, Times(1)).Delete(cid)

	st.Up().Close()
	st.Down().Close()
}
