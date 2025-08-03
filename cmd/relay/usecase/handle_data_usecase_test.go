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

func TestHandleDataUseCase_DataForwardMiddle(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleDataUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedData := []byte("encrypted-data")
	decryptedData := []byte("decrypted-data")
	dataPayload := []byte("data-payload")

	// Create mock connections (middle relay - has down connection)
	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)

	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: dataPayload}

	// Mock ensureServeDown function
	serveDownCalled := false
	ensureServeDown := func(st *entity.ConnState) {
		serveDownCalled = true
	}

	// Mock data payload DTO
	dataDTO := &service.DataPayloadDTO{StreamID: 1, Data: encryptedData}
	forwardPayload := []byte("forward-payload")

	// Configure mocks for middle relay scenario (st.Down() != nil and not hidden)
	WhenDouble(mockEncoder.DecodeDataPayload(dataPayload)).ThenReturn(dataDTO, nil)
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedData)).ThenReturn(decryptedData, nil)
	WhenDouble(mockEncoder.EncodeDataPayload(Any[*service.DataPayloadDTO]())).ThenReturn(forwardPayload, nil)
	WhenSingle(mockSender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)

	// Execute
	err := uc.Data(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	if !serveDownCalled {
		t.Errorf("ensureServeDown not called")
	}

	// Verify mock interactions
	Verify(mockEncoder, Times(1)).DecodeDataPayload(dataPayload)
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedData)
	Verify(mockEncoder, Times(1)).EncodeDataPayload(Any[*service.DataPayloadDTO]())
	Verify(mockSender, Times(1)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
	st.Down().Close()
}

func TestHandleDataUseCase_DataExit(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleDataUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedData := []byte("encrypted-data")
	decryptedData := []byte("decrypted-data")
	dataPayload := []byte("data-payload")

	// Create mock connection (exit node - no down connection)
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: dataPayload}

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Mock data payload DTO and stream connection
	sid, _ := vo.StreamIDFrom(1)
	dataDTO := &service.DataPayloadDTO{StreamID: 1, Data: encryptedData}
	mockConn := Mock[net.Conn](ctrl)

	// Configure mocks for exit scenario (st.Down() == nil and not hidden)
	WhenDouble(mockEncoder.DecodeDataPayload(dataPayload)).ThenReturn(dataDTO, nil)
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedData)).ThenReturn(decryptedData, nil)
	WhenDouble(mockRepo.GetStream(cid, sid)).ThenReturn(mockConn, nil)
	WhenDouble(mockConn.Write(decryptedData)).ThenReturn(len(decryptedData), nil)

	// Execute
	err := uc.Data(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockEncoder, Times(1)).DecodeDataPayload(dataPayload)
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedData)
	Verify(mockRepo, Times(1)).GetStream(cid, sid)
	Verify(mockConn, Times(1)).Write(decryptedData)

	st.Up().Close()
}

func TestHandleDataUseCase_DataHidden(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleDataUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	encryptedData := []byte("encrypted-data")
	decryptedData := []byte("decrypted-data")
	dataPayload := []byte("data-payload")

	// Create mock connections (hidden service)
	up1, _ := net.Pipe()
	mockDown := Mock[net.Conn](ctrl)
	st := entity.NewConnState(key, nonce, up1, mockDown)
	st.SetHidden(true) // Set as hidden service

	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: dataPayload}

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Mock data payload DTO
	dataDTO := &service.DataPayloadDTO{StreamID: 1, Data: encryptedData}

	// Configure mocks for hidden scenario
	WhenDouble(mockEncoder.DecodeDataPayload(dataPayload)).ThenReturn(dataDTO, nil)
	WhenDouble(mockCrypto.AESOpen(key, nonce, encryptedData)).ThenReturn(decryptedData, nil)
	WhenDouble(mockDown.Write(decryptedData)).ThenReturn(len(decryptedData), nil)

	// Execute
	err := uc.Data(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockEncoder, Times(1)).DecodeDataPayload(dataPayload)
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, encryptedData)
	Verify(mockDown, Times(1)).Write(decryptedData)
	// Verify ForwardCell is NOT called for hidden services
	Verify(mockSender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
}

func TestHandleDataUseCase_DataUpstream(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleDataUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	invalidData := []byte("invalid encrypted data")
	dataPayload := []byte("data-payload")
	upstreamEncrypted := []byte("upstream-encrypted")
	upstreamPayload := []byte("upstream-payload")

	// Create mock connections (middle relay with down connection)
	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)

	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: dataPayload}

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Mock data payload DTO
	dataDTO := &service.DataPayloadDTO{StreamID: 1, Data: invalidData}

	// Configure mocks for upstream scenario (decryption fails, but has down connection)
	cryptoError := errors.New("decryption failed")
	WhenDouble(mockEncoder.DecodeDataPayload(dataPayload)).ThenReturn(dataDTO, nil)
	WhenDouble(mockCrypto.AESOpen(key, nonce, invalidData)).ThenReturn(nil, cryptoError)
	WhenDouble(mockCrypto.AESSeal(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(upstreamEncrypted, nil)
	WhenDouble(mockEncoder.EncodeDataPayload(Any[*service.DataPayloadDTO]())).ThenReturn(upstreamPayload, nil)
	WhenSingle(mockSender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)

	// Execute
	err := uc.Data(st, cid, cell, ensureServeDown)

	// Verify
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockEncoder, Times(1)).DecodeDataPayload(dataPayload)
	Verify(mockCrypto, Times(1)).AESOpen(key, nonce, invalidData)
	Verify(mockCrypto, Times(1)).AESSeal(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())
	Verify(mockEncoder, Times(1)).EncodeDataPayload(Any[*service.DataPayloadDTO]())
	Verify(mockSender, Times(1)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
	st.Down().Close()
}

// Test payload decoding failure
func TestHandleDataUseCase_PayloadDecodingFailure(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleDataUseCase(mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	dataPayload := []byte("invalid-payload")

	// Create mock connection
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: dataPayload}

	ensureServeDown := func(st *entity.ConnState) {}

	// Configure mocks
	decodingError := errors.New("payload decoding failed")
	WhenDouble(mockEncoder.DecodeDataPayload(dataPayload)).ThenReturn(nil, decodingError)

	// Execute
	err := uc.Data(st, cid, cell, ensureServeDown)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, decodingError) {
		t.Fatalf("Expected decoding error, got: %v", err)
	}

	// Verify interactions
	Verify(mockEncoder, Times(1)).DecodeDataPayload(dataPayload)
	Verify(mockCrypto, Times(0)).AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())

	st.Up().Close()
}
