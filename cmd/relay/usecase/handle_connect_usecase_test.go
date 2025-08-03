package usecase_test

import (
	"net"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"

	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestHandleConnectUseCase_Connect(t *testing.T) {
	type connectTestMocks struct {
		repo    repository.ConnStateRepository
		crypto  service.CryptoService
		sender  service.CellSenderService
		encoder service.PayloadEncodingService
	}

	type connectConnStateSetup struct {
		hasDown bool
	}

	type connectExpectedCalls struct {
		decodeConnectPayload int
		repoAdd              int
		sendAck              int
		forwardCell          int
	}

	type connectTestData struct {
		encryptedPayload []byte
		decryptedPayload []byte
		target           string
		needsStub        bool
		simulateError    bool
	}

	tests := []struct {
		name            string
		setupConn       func() (*entity.ConnState, connectConnStateSetup)
		setupTestData   func() connectTestData
		setupMocks      func(mocks connectTestMocks, cid vo.CircuitID, testData connectTestData, setup connectConnStateSetup)
		expectError     bool
		expectedCalls   connectExpectedCalls
		expectServeDown bool
	}{
		{
			name: "Connect middle relay - forward cell",
			setupConn: func() (*entity.ConnState, connectConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				down1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, down1)
				return st, connectConnStateSetup{hasDown: true}
			},
			setupTestData: func() connectTestData {
				return connectTestData{
					encryptedPayload: []byte("encrypted-payload"),
					decryptedPayload: []byte("decrypted-payload"),
					target:           "",
					needsStub:        false,
					simulateError:    false,
				}
			},
			setupMocks: func(mocks connectTestMocks, cid vo.CircuitID, testData connectTestData, setup connectConnStateSetup) {
				// Mock AESOpen with all Any matchers
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.decryptedPayload, nil)
				WhenSingle(mocks.sender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
			},
			expectError:     false,
			expectedCalls:   connectExpectedCalls{decodeConnectPayload: 0, repoAdd: 0, sendAck: 0, forwardCell: 1},
			expectServeDown: true,
		},
		{
			name: "Connect exit node - establish connection with empty payload",
			setupConn: func() (*entity.ConnState, connectConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil)
				return st, connectConnStateSetup{hasDown: false}
			},
			setupTestData: func() connectTestData {
				return connectTestData{
					encryptedPayload: []byte("encrypted-payload"),
					decryptedPayload: []byte(""), // Empty payload - uses environment variable
					target:           "",
					needsStub:        true,
					simulateError:    false,
				}
			},
			setupMocks: func(mocks connectTestMocks, cid vo.CircuitID, testData connectTestData, setup connectConnStateSetup) {
				// For exit node with empty payload, no DecodeConnectPayload call
				// Mock AESOpen with all Any matchers to return empty payload
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.decryptedPayload, nil)
				WhenSingle(mocks.repo.Add(Any[vo.CircuitID](), Any[*entity.ConnState]())).ThenReturn(nil)
				WhenSingle(mocks.sender.SendAck(Any[net.Conn](), Any[vo.CircuitID]())).ThenReturn(nil)
			},
			expectError:     false,
			expectedCalls:   connectExpectedCalls{decodeConnectPayload: 0, repoAdd: 1, sendAck: 1, forwardCell: 0},
			expectServeDown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctrl := NewMockController(t)

			stubAddr, err := makeTestStabServer()
			if err != nil {
				t.Fatalf("setup stub server: %v", err)
			}

			// env variable for stub server address
			t.Setenv("PTOR_HIDDEN_ADDR", stubAddr)

			// Create mocks
			mocks := connectTestMocks{
				repo:    Mock[repository.ConnStateRepository](ctrl),
				crypto:  Mock[service.CryptoService](ctrl),
				sender:  Mock[service.CellSenderService](ctrl),
				encoder: Mock[service.PayloadEncodingService](ctrl),
			}

			// Setup connection state
			st, connSetup := tt.setupConn()
			defer func() {
				if st.Up() != nil {
					st.Up().Close()
				}
				if st.Down() != nil {
					st.Down().Close()
				}
			}()

			// Setup test data
			testData := tt.setupTestData()
			cid := vo.NewCircuitID()
			cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: testData.encryptedPayload}

			// Setup mock behaviors
			tt.setupMocks(mocks, cid, testData, connSetup)

			// Track ensureServeDown calls
			serveDownCalled := false
			ensureServeDown := func(st *entity.ConnState) {
				serveDownCalled = true
			}

			// Create usecase and execute
			uc := usecase.NewHandleConnectUseCase(mocks.repo, mocks.crypto, mocks.sender, mocks.encoder)
			err = uc.Connect(st, cid, cell, ensureServeDown)

			// Verify error expectations
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			// Verify ensureServeDown call
			if tt.expectServeDown && !serveDownCalled {
				t.Errorf("Expected ensureServeDown to be called")
			}
			if !tt.expectServeDown && serveDownCalled {
				t.Errorf("Unexpected ensureServeDown call")
			}

			// Verify mock interactions using parameterized call counts
			// Note: Skip AESOpen verification due to nonce side effects - focus on business logic
			if tt.expectedCalls.decodeConnectPayload > 0 {
				Verify(mocks.encoder, Times(tt.expectedCalls.decodeConnectPayload)).DecodeConnectPayload(testData.decryptedPayload)
			}
			Verify(mocks.repo, Times(tt.expectedCalls.repoAdd)).Add(Any[vo.CircuitID](), Any[*entity.ConnState]())
			Verify(mocks.sender, Times(tt.expectedCalls.sendAck)).SendAck(Any[net.Conn](), Any[vo.CircuitID]())
			Verify(mocks.sender, Times(tt.expectedCalls.forwardCell)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
		})
	}
}
