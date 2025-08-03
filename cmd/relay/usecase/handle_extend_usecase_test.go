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

type extendTestMocks struct {
	repo    repository.ConnStateRepository
	crypto  service.CryptoService
	sender  service.CellSenderService
	encoder service.PayloadEncodingService
}

type extendExpectedCalls struct {
	decodeExtendPayload  int
	x25519Generate       int
	x25519Shared         int
	deriveKeyNonce       int
	repoAdd              int
	encodeCreatedPayload int
	sendCreated          int
	forwardCell          int
	mockDownRead         int
}

type extendTestData struct {
	payload       []byte
	nextHop       string
	isExitNode    bool
	hasDownConn   bool
	simulateError bool
	errorType     string
}

func TestHandleExtendUseCase(t *testing.T) {
	tests := []struct {
		name           string
		testMethod     string // "Extend" or "ForwardExtend"
		testData       extendTestData
		setupMocks     func(mocks extendTestMocks, testData extendTestData)
		setupConnState func() (*entity.ConnState, net.Conn) // returns state and mock down connection if needed
		expectError    bool
		errorCheck     func(error) bool
		expectedCalls  extendExpectedCalls
	}{
		{
			name:       "Extend - Exit node scenario",
			testMethod: "Extend",
			testData: extendTestData{
				payload:       []byte("extend-payload"),
				nextHop:       "", // Empty means exit node
				isExitNode:    true,
				hasDownConn:   false,
				simulateError: false,
			},
			setupMocks: func(mocks extendTestMocks, testData extendTestData) {
				clientPub := [32]byte{1, 2, 3, 4}
				relayPriv := []byte("relay-private-key")
				relayPub := []byte("relay-public-key")
				sharedSecret := []byte("shared-secret")
				derivedKey, _ := vo.NewAESKey()
				derivedNonce, _ := vo.NewNonce()
				createdPayload := []byte("created-payload")

				extendDTO := &service.ExtendPayloadDTO{NextHop: testData.nextHop, ClientPub: clientPub}
				WhenDouble(mocks.encoder.DecodeExtendPayload(testData.payload)).ThenReturn(extendDTO, nil)
				When(mocks.crypto.X25519Generate()).ThenReturn(relayPriv, relayPub, nil)
				WhenDouble(mocks.crypto.X25519Shared(relayPriv, clientPub[:])).ThenReturn(sharedSecret, nil)
				When(mocks.crypto.DeriveKeyNonce(sharedSecret)).ThenReturn(derivedKey, derivedNonce, nil)
				WhenSingle(mocks.repo.Add(Any[vo.CircuitID](), Any[*entity.ConnState]())).ThenReturn(nil)
				WhenDouble(mocks.encoder.EncodeCreatedPayload(Any[*service.CreatedPayloadDTO]())).ThenReturn(createdPayload, nil)
				WhenSingle(mocks.sender.SendCreated(Any[net.Conn](), Any[vo.CircuitID](), Any[[]byte]())).ThenReturn(nil)
			},
			setupConnState: func() (*entity.ConnState, net.Conn) {
				return nil, nil // Not needed for Extend method
			},
			expectError:   false,
			expectedCalls: extendExpectedCalls{decodeExtendPayload: 1, x25519Generate: 1, x25519Shared: 1, deriveKeyNonce: 1, repoAdd: 1, encodeCreatedPayload: 1, sendCreated: 1, forwardCell: 0, mockDownRead: 0},
		},
		{
			name:       "ForwardExtend - Middle relay scenario",
			testMethod: "ForwardExtend",
			testData: extendTestData{
				payload:       []byte("extend-payload"),
				nextHop:       "next-relay:443",
				isExitNode:    false,
				hasDownConn:   true,
				simulateError: false,
			},
			setupMocks: func(mocks extendTestMocks, testData extendTestData) {
				createdPayload := []byte("created-payload")

				// Mock reading response header and payload
				responseHeader := make([]byte, 20)
				responseHeader[16] = byte(vo.CmdCreated)
				responseHeader[17] = byte(vo.ProtocolV1)
				responseHeader[18] = 0
				responseHeader[19] = byte(len(createdPayload))

				WhenSingle(mocks.sender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
				WhenSingle(mocks.sender.SendCreated(Any[net.Conn](), Any[vo.CircuitID](), Any[[]byte]())).ThenReturn(nil)
			},
			setupConnState: func() (*entity.ConnState, net.Conn) {
				return nil, nil // Mock setup moved to setupMocks for simplicity
			},
			expectError:   false,
			expectedCalls: extendExpectedCalls{decodeExtendPayload: 0, x25519Generate: 0, x25519Shared: 0, deriveKeyNonce: 0, repoAdd: 0, encodeCreatedPayload: 0, sendCreated: 1, forwardCell: 1, mockDownRead: 2},
		},
		{
			name:       "Payload decoding failure",
			testMethod: "Extend",
			testData: extendTestData{
				payload:       []byte("invalid-payload"),
				simulateError: true,
				errorType:     "decode",
			},
			setupMocks: func(mocks extendTestMocks, testData extendTestData) {
				decodingError := errors.New("payload decoding failed")
				WhenDouble(mocks.encoder.DecodeExtendPayload(testData.payload)).ThenReturn(nil, decodingError)
			},
			setupConnState: func() (*entity.ConnState, net.Conn) {
				return nil, nil
			},
			expectError: true,
			errorCheck: func(err error) bool {
				return err != nil && err.Error() == "payload decoding failed"
			},
			expectedCalls: extendExpectedCalls{decodeExtendPayload: 1, x25519Generate: 0, x25519Shared: 0, deriveKeyNonce: 0, repoAdd: 0, encodeCreatedPayload: 0, sendCreated: 0, forwardCell: 0, mockDownRead: 0},
		},
		{
			name:       "ForwardExtend - No downstream connection",
			testMethod: "ForwardExtend",
			testData: extendTestData{
				payload:       []byte("extend-payload"),
				hasDownConn:   false,
				simulateError: true,
				errorType:     "no_downstream",
			},
			setupMocks: func(mocks extendTestMocks, testData extendTestData) {
				// No mocks needed as error occurs before mock calls
			},
			setupConnState: func() (*entity.ConnState, net.Conn) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil) // No down connection
				return st, nil
			},
			expectError: true,
			errorCheck: func(err error) bool {
				return err != nil && err.Error() == "no downstream connection"
			},
			expectedCalls: extendExpectedCalls{decodeExtendPayload: 0, x25519Generate: 0, x25519Shared: 0, deriveKeyNonce: 0, repoAdd: 0, encodeCreatedPayload: 0, sendCreated: 0, forwardCell: 0, mockDownRead: 0},
		},
	}

	// Generate RSA key once for all tests
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)

			// Create mocks
			mocks := extendTestMocks{
				repo:    Mock[repository.ConnStateRepository](ctrl),
				crypto:  Mock[service.CryptoService](ctrl),
				sender:  Mock[service.CellSenderService](ctrl),
				encoder: Mock[service.PayloadEncodingService](ctrl),
			}

			// Setup test data
			cid := vo.NewCircuitID()
			up1, _ := net.Pipe()
			defer up1.Close()

			cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: tt.testData.payload}

			// Setup mock behaviors
			tt.setupMocks(mocks, tt.testData)

			// Setup connection state for ForwardExtend if needed
			var st *entity.ConnState
			var mockDown net.Conn
			if tt.testMethod == "ForwardExtend" {
				st, mockDown = tt.setupConnState()
				defer func() {
					if st != nil && st.Up() != nil {
						st.Up().Close()
					}
				}()

				if tt.testData.hasDownConn {
					key, _ := vo.NewAESKey()
					nonce, _ := vo.NewNonce()
					up2, _ := net.Pipe()
					mockDown = Mock[net.Conn](ctrl)
					st = entity.NewConnState(key, nonce, up2, mockDown)

					// Setup mock down connection responses for ForwardExtend
					createdPayload := []byte("created-payload")
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
				} else {
					// No down connection case
					key, _ := vo.NewAESKey()
					nonce, _ := vo.NewNonce()
					up2, _ := net.Pipe()
					st = entity.NewConnState(key, nonce, up2, nil)
				}
			}

			// Create usecase and execute
			uc := usecase.NewHandleExtendUseCase(priv, mocks.repo, mocks.crypto, mocks.sender, mocks.encoder)

			var err error
			switch tt.testMethod {
			case "Extend":
				err = uc.Extend(up1, cid, cell)
			case "ForwardExtend":
				err = uc.ForwardExtend(st, cid, cell)
			}

			// Verify error expectations
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tt.errorCheck != nil && !tt.errorCheck(err) {
					t.Fatalf("Error check failed for error: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			// Verify mock interactions using parameterized call counts
			Verify(mocks.encoder, Times(tt.expectedCalls.decodeExtendPayload)).DecodeExtendPayload(Any[[]byte]())
			Verify(mocks.crypto, Times(tt.expectedCalls.x25519Generate)).X25519Generate()
			Verify(mocks.crypto, Times(tt.expectedCalls.x25519Shared)).X25519Shared(Any[[]byte](), Any[[]byte]())
			Verify(mocks.crypto, Times(tt.expectedCalls.deriveKeyNonce)).DeriveKeyNonce(Any[[]byte]())
			Verify(mocks.repo, Times(tt.expectedCalls.repoAdd)).Add(Any[vo.CircuitID](), Any[*entity.ConnState]())
			Verify(mocks.encoder, Times(tt.expectedCalls.encodeCreatedPayload)).EncodeCreatedPayload(Any[*service.CreatedPayloadDTO]())
			Verify(mocks.sender, Times(tt.expectedCalls.sendCreated)).SendCreated(Any[net.Conn](), Any[vo.CircuitID](), Any[[]byte]())
			Verify(mocks.sender, Times(tt.expectedCalls.forwardCell)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

			// Verify mockDown interactions if applicable
			if mockDown != nil && tt.expectedCalls.mockDownRead > 0 {
				Verify(mockDown, Times(tt.expectedCalls.mockDownRead)).Read(Any[[]byte]())
			}
		})
	}
}
