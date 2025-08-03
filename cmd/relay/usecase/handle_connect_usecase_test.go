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

func TestHandleConnectUseCase_ConnectMiddle(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleConnectUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")
	decryptedPayload := []byte("decrypted-payload")

	// Create mock connections (middle relay - has down connection)
	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)

	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: encryptedPayload}

	// Mock ensureServeDown function
	serveDownCalled := false
	ensureServeDown := func(st *entity.ConnState) {
		serveDownCalled = true
	}

	// Configure mocks for middle relay scenario (st.Down() != nil)
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(decryptedPayload, nil)
	WhenSingle(mockSender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)

	// Execute
	err := uc.Connect(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
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

func TestHandleConnectUseCase_ConnectExit(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleConnectUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")
	decryptedPayload := []byte("decrypted-payload")

	// Create mock connection (exit node - no down connection)
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: encryptedPayload}

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Mock connect payload DTO
	connectDTO := &service.ConnectPayloadDTO{Target: "example.com:80"}

	// Configure mocks for exit scenario (st.Down() == nil)
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(decryptedPayload, nil)
	WhenDouble(mockEncoder.DecodeConnectPayload(decryptedPayload)).ThenReturn(connectDTO, nil)
	WhenSingle(mockRepo.Add(Any[vo.CircuitID](), Any[*entity.ConnState]())).ThenReturn(nil)
	WhenSingle(mockSender.SendAck(Any[net.Conn](), Any[vo.CircuitID]())).ThenReturn(nil)

	// Execute
	err := uc.Connect(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedPayload)
	Verify(mockEncoder, Times(1)).DecodeConnectPayload(decryptedPayload)
	Verify(mockRepo, Times(1)).Add(Any[vo.CircuitID](), Any[*entity.ConnState]())
	Verify(mockSender, Times(1)).SendAck(Any[net.Conn](), Any[vo.CircuitID]())

	st.Up().Close()
}

func TestHandleConnectUseCase_ConnectExitWithEmptyPayload(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleConnectUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")
	emptyDecryptedPayload := []byte{} // Empty payload should trigger default address logic

	// Create mock connection (exit node - no down connection)
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: encryptedPayload}

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Configure mocks for exit scenario with empty payload
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(emptyDecryptedPayload, nil)
	WhenSingle(mockRepo.Add(Any[vo.CircuitID](), Any[*entity.ConnState]())).ThenReturn(nil)
	WhenSingle(mockSender.SendAck(Any[net.Conn](), Any[vo.CircuitID]())).ThenReturn(nil)

	// Execute
	err := uc.Connect(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedPayload)
	// Empty payload should not trigger DecodeConnectPayload call
	Verify(mockEncoder, Times(0)).DecodeConnectPayload(Any[[]byte]())
	Verify(mockRepo, Times(1)).Add(Any[vo.CircuitID](), Any[*entity.ConnState]())
	Verify(mockSender, Times(1)).SendAck(Any[net.Conn](), Any[vo.CircuitID]())

	st.Up().Close()
}

// Test crypto service failure
func TestHandleConnectUseCase_CryptoFailure(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleConnectUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")

	// Create mock connection (middle relay)
	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)

	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: encryptedPayload}

	ensureServeDown := func(st *entity.ConnState) {}

	// Configure mock to return error
	cryptoError := errors.New("decryption failed")
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(nil, cryptoError)

	// Execute
	err := uc.Connect(st, cid, cell, ensureServeDown)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, cryptoError) {
		t.Fatalf("Expected crypto error, got: %v", err)
	}

	// Verify only AESOpen was called
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedPayload)
	Verify(mockSender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
	st.Down().Close()
}

// Test payload decoding failure
func TestHandleConnectUseCase_PayloadDecodingFailure(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleConnectUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedPayload := []byte("encrypted-payload")
	decryptedPayload := []byte("decrypted-payload")

	// Create mock connection (exit node)
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: encryptedPayload}

	ensureServeDown := func(st *entity.ConnState) {}

	// Configure mocks
	decodingError := errors.New("payload decoding failed")
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedPayload)).ThenReturn(decryptedPayload, nil)
	WhenDouble(mockEncoder.DecodeConnectPayload(decryptedPayload)).ThenReturn(nil, decodingError)

	// Execute
	err := uc.Connect(st, cid, cell, ensureServeDown)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, decodingError) {
		t.Fatalf("Expected decoding error, got: %v", err)
	}

	// Verify interactions
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedPayload)
	Verify(mockEncoder, Times(1)).DecodeConnectPayload(decryptedPayload)
	Verify(mockRepo, Times(0)).Add(Any[vo.CircuitID](), Any[*entity.ConnState]())

	st.Up().Close()
}
