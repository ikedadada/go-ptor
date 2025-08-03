package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
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

func TestHandleExtendUseCase_Extend(t *testing.T) {
	ctrl := NewMockController(t)

	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleExtendUseCase(priv, mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	cid := vo.NewCircuitID()
	extendPayload := []byte("extend-payload")
	clientPub := [32]byte{1, 2, 3, 4}
	relayPriv := []byte("relay-private-key")
	relayPub := []byte("relay-public-key")
	sharedSecret := []byte("shared-secret")
	derivedKey, _ := vo.NewAESKey()
	derivedNonce, _ := vo.NewNonce()
	createdPayload := []byte("created-payload")

	up1, _ := net.Pipe()
	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: extendPayload}

	// Mock extend payload DTO (no next hop - exit node scenario)
	extendDTO := &service.ExtendPayloadDTO{NextHop: "", ClientPub: clientPub}

	// Configure mocks for successful extend operation
	WhenDouble(mockEncoder.DecodeExtendPayload(extendPayload)).ThenReturn(extendDTO, nil)
	When(mockCrypto.X25519Generate()).ThenReturn(relayPriv, relayPub, nil)
	WhenDouble(mockCrypto.X25519Shared(relayPriv, clientPub[:])).ThenReturn(sharedSecret, nil)
	When(mockCrypto.DeriveKeyNonce(sharedSecret)).ThenReturn(derivedKey, derivedNonce, nil)
	WhenSingle(mockRepo.Add(Any[vo.CircuitID](), Any[*entity.ConnState]())).ThenReturn(nil)
	WhenDouble(mockEncoder.EncodeCreatedPayload(Any[*service.CreatedPayloadDTO]())).ThenReturn(createdPayload, nil)
	WhenSingle(mockSender.SendCreated(up1, cid, createdPayload)).ThenReturn(nil)

	// Execute
	err := uc.Extend(up1, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("Extend failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockEncoder, Times(1)).DecodeExtendPayload(extendPayload)
	Verify(mockCrypto, Times(1)).X25519Generate()
	Verify(mockCrypto, Times(1)).X25519Shared(relayPriv, clientPub[:])
	Verify(mockCrypto, Times(1)).DeriveKeyNonce(sharedSecret)
	Verify(mockRepo, Times(1)).Add(Any[vo.CircuitID](), Any[*entity.ConnState]())
	Verify(mockEncoder, Times(1)).EncodeCreatedPayload(Any[*service.CreatedPayloadDTO]())
	Verify(mockSender, Times(1)).SendCreated(up1, cid, createdPayload)

	up1.Close()
}

func TestHandleExtendUseCase_ForwardExtend(t *testing.T) {
	ctrl := NewMockController(t)

	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleExtendUseCase(priv, mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	extendPayload := []byte("extend-payload")
	createdPayload := []byte("created-payload")

	// Create mock connections
	up1, _ := net.Pipe()
	mockDown := Mock[net.Conn](ctrl)
	st := entity.NewConnState(key, nonce, up1, mockDown)

	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: extendPayload}

	// Configure mocks for forward extend operation
	WhenSingle(mockSender.ForwardCell(mockDown, cid, cell)).ThenReturn(nil)

	// Mock reading response header and payload
	responseHeader := make([]byte, 20)
	responseHeader[16] = byte(vo.CmdCreated)
	responseHeader[17] = byte(vo.ProtocolV1)
	responseHeader[18] = 0
	responseHeader[19] = byte(len(createdPayload))

	WhenDouble(mockDown.Read(Any[[]byte]())).ThenAnswer(func(args []any) (int, error) {
		buf := args[0].([]byte)
		copy(buf, responseHeader)
		return len(responseHeader), nil
	}).ThenAnswer(func(args []any) (int, error) {
		buf := args[0].([]byte)
		copy(buf, createdPayload)
		return len(createdPayload), nil
	})

	WhenSingle(mockSender.SendCreated(up1, cid, createdPayload)).ThenReturn(nil)

	// Execute
	err := uc.ForwardExtend(st, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("ForwardExtend failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockSender, Times(1)).ForwardCell(mockDown, cid, cell)
	Verify(mockDown, Times(2)).Read(Any[[]byte]())
	Verify(mockSender, Times(1)).SendCreated(up1, cid, createdPayload)

	st.Up().Close()
}

// Test payload decoding failure
func TestHandleExtendUseCase_PayloadDecodingFailure(t *testing.T) {
	ctrl := NewMockController(t)

	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleExtendUseCase(priv, mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	cid := vo.NewCircuitID()
	invalidPayload := []byte("invalid-payload")

	up1, _ := net.Pipe()
	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: invalidPayload}

	// Configure mocks
	decodingError := errors.New("payload decoding failed")
	WhenDouble(mockEncoder.DecodeExtendPayload(invalidPayload)).ThenReturn(nil, decodingError)

	// Execute
	err := uc.Extend(up1, cid, cell)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, decodingError) {
		t.Fatalf("Expected decoding error, got: %v", err)
	}

	// Verify interactions
	Verify(mockEncoder, Times(1)).DecodeExtendPayload(invalidPayload)
	Verify(mockCrypto, Times(0)).X25519Generate()

	up1.Close()
}

// Test ForwardExtend with no down connection
func TestHandleExtendUseCase_ForwardExtendNoDownConnection(t *testing.T) {
	ctrl := NewMockController(t)

	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockCrypto := Mock[service.CryptoService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockEncoder := Mock[service.PayloadEncodingService](ctrl)

	uc := usecase.NewHandleExtendUseCase(priv, mockRepo, mockCrypto, mockSender, mockEncoder)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	extendPayload := []byte("extend-payload")

	// Create state with no down connection
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: extendPayload}

	// Execute
	err := uc.ForwardExtend(st, cid, cell)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err.Error() != "no downstream connection" {
		t.Fatalf("Expected 'no downstream connection' error, got: %v", err)
	}

	// Verify no mock interactions
	Verify(mockSender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
}
