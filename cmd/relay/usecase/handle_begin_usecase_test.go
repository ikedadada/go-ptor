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

func TestHandleBeginUseCase_BeginForward(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleBeginUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")
	decryptedPayload := []byte("decrypted-payload")

	// Create mock connections
	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)

	cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: encryptedPayload}

	// Mock ensureServeDown function
	serveDownCalled := false
	ensureServeDown := func(st *entity.ConnState) {
		serveDownCalled = true
	}

	// Configure mocks for forward scenario (st.Down() != nil and not hidden)
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(decryptedPayload, nil)
	WhenSingle(mockSender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)

	// Execute
	err := uc.Begin(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	if !serveDownCalled {
		t.Errorf("ensureServeDown not called")
	}

	// Verify mock interactions
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedPayload)
	Verify(mockSender, Times(1)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
	st.Down().Close()
}

func TestHandleBeginUseCase_BeginExit(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleBeginUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")
	decryptedPayload := []byte("decrypted-payload")

	// Create mock connection (exit node - no down connection)
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: encryptedPayload}

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Mock begin payload DTO
	beginDTO := &service.BeginPayloadDTO{StreamID: 1, Target: "example.com:80"}

	// Configure mocks for exit scenario (st.Down() == nil and not hidden)
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(decryptedPayload, nil)
	WhenDouble(mockEncoder.DecodeBeginPayload(decryptedPayload)).ThenReturn(beginDTO, nil)
	WhenSingle(mockRepo.AddStream(Any[vo.CircuitID](), Any[vo.StreamID](), Any[net.Conn]())).ThenReturn(nil)
	WhenSingle(mockSender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)

	// Execute
	err := uc.Begin(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedPayload)
	Verify(mockEncoder, Times(1)).DecodeBeginPayload(decryptedPayload)
	Verify(mockRepo, Times(1)).AddStream(Any[vo.CircuitID](), Any[vo.StreamID](), Any[net.Conn]())
	Verify(mockSender, Times(1)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
}

func TestHandleBeginUseCase_BeginHidden(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleBeginUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")
	decryptedPayload := []byte("decrypted-payload")

	// Create mock connections
	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)
	st.SetHidden(true) // Set as hidden service

	cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: encryptedPayload}

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Mock begin payload DTO
	beginDTO := &service.BeginPayloadDTO{StreamID: 1, Target: "svc"}

	// Configure mocks for hidden scenario
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(decryptedPayload, nil)
	WhenDouble(mockEncoder.DecodeBeginPayload(decryptedPayload)).ThenReturn(beginDTO, nil)
	WhenSingle(mockSender.SendAck(Any[net.Conn](), Any[vo.CircuitID]())).ThenReturn(nil)

	// Execute
	err := uc.Begin(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedPayload)
	Verify(mockEncoder, Times(1)).DecodeBeginPayload(decryptedPayload)
	Verify(mockSender, Times(1)).SendAck(Any[net.Conn](), Any[vo.CircuitID]())
	// Verify ForwardCell is NOT called for hidden services
	Verify(mockSender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
	st.Down().Close()
}

// Test crypto service failure
func TestHandleBeginUseCase_CryptoFailure(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleBeginUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")

	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)

	cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: encryptedPayload}

	ensureServeDown := func(st *entity.ConnState) {}

	// Configure mock to return error
	cryptoError := errors.New("decryption failed")
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(nil, cryptoError)

	// Execute
	err := uc.Begin(st, cid, cell, ensureServeDown)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, cryptoError) {
		t.Fatalf("Expected crypto error, got: %v", err)
	}

	// Verify only AESOpen was called
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedPayload)
	Verify(mockEncoder, Times(0)).DecodeBeginPayload(Any[[]byte]())
	Verify(mockSender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
	st.Down().Close()
}

// Test payload decoding failure
func TestHandleBeginUseCase_PayloadDecodingFailure(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleBeginUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")
	decryptedPayload := []byte("decrypted-payload")

	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil) // Exit node scenario

	cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: encryptedPayload}

	ensureServeDown := func(st *entity.ConnState) {}

	// Configure mocks
	decodingError := errors.New("payload decoding failed")
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(decryptedPayload, nil)
	WhenDouble(mockEncoder.DecodeBeginPayload(decryptedPayload)).ThenReturn(nil, decodingError)

	// Execute
	err := uc.Begin(st, cid, cell, ensureServeDown)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, decodingError) {
		t.Fatalf("Expected decoding error, got: %v", err)
	}

	// Verify interactions
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedPayload)
	Verify(mockEncoder, Times(1)).DecodeBeginPayload(decryptedPayload)
	Verify(mockRepo, Times(0)).AddStream(Any[vo.CircuitID](), Any[vo.StreamID](), Any[net.Conn]())

	st.Up().Close()
}
