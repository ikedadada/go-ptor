package usecase_test

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/ovechkin-dm/mockio/v2/matchers"
	. "github.com/ovechkin-dm/mockio/v2/mock"

	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestHandleBeginUseCase_Begin(t *testing.T) {
	type beginTestMocks struct {
		repo    repository.ConnStateRepository
		crypto  service.CryptoService
		sender  service.CellSenderService
		encoder service.PayloadEncodingService
	}

	type beginConnStateSetup struct {
		hasDown   bool
		isHidden  bool
		target    string
		streamID  vo.StreamID
		needsStub bool
	}

	type expectedCalls struct {
		aesOpen       int
		decodePayload int
		forwardCell   int
		sendAck       int
		addStream     int
	}

	type beginTestData struct {
		ctrl             *matchers.MockController
		key              vo.AESKey
		nonce            vo.Nonce
		cid              vo.CircuitID
		encryptedPayload []byte
		decryptedPayload []byte
		beginDTO         *service.BeginPayloadDTO
		stubAddr         string
	}

	tests := []struct {
		name            string
		setupConnState  func(ctrl *matchers.MockController) (*entity.ConnState, beginConnStateSetup)
		setupMocks      func(mocks beginTestMocks, setup beginConnStateSetup, testData beginTestData)
		expectError     bool
		errorCheck      func(error) bool
		expectedCalls   expectedCalls
		expectServeDown bool
		needsDelay      bool
	}{
		{
			name: "Forward to middle relay",
			setupConnState: func(ctrl *matchers.MockController) (*entity.ConnState, beginConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				down1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, down1)
				return st, beginConnStateSetup{hasDown: true, isHidden: false, target: "example.com:80", streamID: vo.StreamID(1), needsStub: false}
			},
			setupMocks: func(mocks beginTestMocks, setup beginConnStateSetup, testData beginTestData) {
				// Use Any[] for AESOpen due to nonce side effects from BeginNonce()
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.decryptedPayload, nil)
				WhenSingle(mocks.sender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
			},
			expectError:     false,
			expectedCalls:   expectedCalls{aesOpen: 1, decodePayload: 0, forwardCell: 1, sendAck: 0, addStream: 0},
			expectServeDown: true,
		},
		{
			name: "Exit node begin",
			setupConnState: func(ctrl *matchers.MockController) (*entity.ConnState, beginConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil)
				return st, beginConnStateSetup{hasDown: false, isHidden: false, target: "example.com:80", streamID: vo.StreamID(1), needsStub: true}
			},
			setupMocks: func(mocks beginTestMocks, setup beginConnStateSetup, testData beginTestData) {
				// Use Any[] for AESOpen due to nonce side effects from BeginNonce()
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.decryptedPayload, nil)
				// Use Any[] for DecodeBeginPayload to match any decrypted payload
				WhenDouble(mocks.encoder.DecodeBeginPayload(Any[[]byte]())).ThenReturn(testData.beginDTO, nil)
				WhenSingle(mocks.repo.AddStream(Any[vo.CircuitID](), Any[vo.StreamID](), Any[net.Conn]())).ThenReturn(nil)
				WhenSingle(mocks.sender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
			},
			expectError:     false,
			expectedCalls:   expectedCalls{aesOpen: 1, decodePayload: 1, forwardCell: 2, sendAck: 0, addStream: 1},
			expectServeDown: false,
		},

		{
			name: "Hidden service begin",
			setupConnState: func(ctrl *matchers.MockController) (*entity.ConnState, beginConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				down1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, down1)
				st.SetHidden(true)
				return st, beginConnStateSetup{hasDown: true, isHidden: true, target: "svc", streamID: vo.StreamID(1), needsStub: false}
			},
			setupMocks: func(mocks beginTestMocks, setup beginConnStateSetup, testData beginTestData) {
				// Use Any[] for AESOpen due to nonce side effects from BeginNonce()
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.decryptedPayload, nil)
				// Use Any[] for DecodeBeginPayload to match any decrypted payload
				WhenDouble(mocks.encoder.DecodeBeginPayload(Any[[]byte]())).ThenReturn(testData.beginDTO, nil)
				WhenSingle(mocks.sender.SendAck(Any[net.Conn](), Any[vo.CircuitID]())).ThenReturn(nil)
			},
			expectError:     false,
			expectedCalls:   expectedCalls{aesOpen: 1, decodePayload: 1, forwardCell: 0, sendAck: 1, addStream: 0},
			expectServeDown: false,
		},

		{
			name: "Payload decoding failure",
			setupConnState: func(ctrl *matchers.MockController) (*entity.ConnState, beginConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil) // Exit node scenario
				return st, beginConnStateSetup{hasDown: false, isHidden: false, target: "", streamID: vo.StreamID(1), needsStub: false}
			},
			setupMocks: func(mocks beginTestMocks, setup beginConnStateSetup, testData beginTestData) {
				decodingError := errors.New("payload decoding failed")
				// Use Any[] for AESOpen due to nonce side effects from BeginNonce()
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.decryptedPayload, nil)
				// Use Any[] for DecodeBeginPayload to match any decrypted payload
				WhenDouble(mocks.encoder.DecodeBeginPayload(Any[[]byte]())).ThenReturn(nil, decodingError)
			},
			expectError: true,
			errorCheck: func(err error) bool {
				return err != nil && err.Error() == "payload decoding failed"
			},
			expectedCalls:   expectedCalls{aesOpen: 1, decodePayload: 1, forwardCell: 0, sendAck: 0, addStream: 0},
			expectServeDown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)

			// Create mocks
			mocks := beginTestMocks{
				repo:    Mock[repository.ConnStateRepository](ctrl),
				crypto:  Mock[service.CryptoService](ctrl),
				sender:  Mock[service.CellSenderService](ctrl),
				encoder: Mock[service.PayloadEncodingService](ctrl),
			}

			// Setup connection state
			st, connSetup := tt.setupConnState(ctrl)
			defer func() {
				if st.Up() != nil {
					st.Up().Close()
				}
				if st.Down() != nil {
					st.Down().Close()
				}
			}()

			// Setup test data
			var stubAddr string
			if connSetup.needsStub {
				var err error
				stubAddr, err = makeTestStabServer()
				if err != nil {
					t.Fatalf("setup stub server: %v", err)
				}
			}

			testData := beginTestData{
				ctrl:             ctrl,
				key:              st.Key(),
				nonce:            vo.Nonce{}, // Not used with Any[] matchers
				cid:              vo.NewCircuitID(),
				encryptedPayload: []byte("encrypted-payload"),
				decryptedPayload: []byte("decrypted-payload"),
				beginDTO:         &service.BeginPayloadDTO{StreamID: connSetup.streamID, Target: connSetup.target},
				stubAddr:         stubAddr,
			}

			// Use stub address if needed
			if connSetup.needsStub {
				testData.beginDTO.Target = stubAddr
			}

			cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: testData.encryptedPayload}

			// Setup mock behaviors after stub address is assigned
			tt.setupMocks(mocks, connSetup, testData)

			// Track ensureServeDown calls
			serveDownCalled := false
			ensureServeDown := func(st *entity.ConnState) {
				serveDownCalled = true
			}

			// Create usecase and execute
			uc := usecase.NewHandleBeginUseCase(mocks.repo, mocks.crypto, mocks.sender, mocks.encoder)
			err := uc.Begin(st, testData.cid, cell, ensureServeDown)

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

			// Give goroutine time to complete if needed
			if tt.needsDelay {
				time.Sleep(50 * time.Millisecond)
			}

			// Verify ensureServeDown call
			if tt.expectServeDown && !serveDownCalled {
				t.Errorf("Expected ensureServeDown to be called")
			}
			if !tt.expectServeDown && serveDownCalled {
				t.Errorf("Unexpected ensureServeDown call")
			}

			// Verify mock interactions using parameterized call counts
			// Focus on business logic verification with expected call counts
			// Note: Use Any[] matchers to handle nonce side effects from BeginNonce()/DataNonce()

			// Verify all expected call counts
			Verify(mocks.encoder, Times(tt.expectedCalls.decodePayload)).DecodeBeginPayload(Any[[]byte]())
			Verify(mocks.sender, Times(tt.expectedCalls.forwardCell)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
			Verify(mocks.sender, Times(tt.expectedCalls.sendAck)).SendAck(Any[net.Conn](), Any[vo.CircuitID]())
			Verify(mocks.repo, Times(tt.expectedCalls.addStream)).AddStream(Any[vo.CircuitID](), Any[vo.StreamID](), Any[net.Conn]())
		})
	}
}
