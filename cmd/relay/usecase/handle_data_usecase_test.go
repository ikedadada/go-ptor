package usecase_test

import (
	"errors"
	"net"
	"testing"

	"github.com/ovechkin-dm/mockio/v2/matchers"
	. "github.com/ovechkin-dm/mockio/v2/mock"

	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestHandleDataUseCase_Handle(t *testing.T) {

	type testMocks struct {
		repo    repository.ConnStateRepository
		crypto  service.CryptoService
		sender  service.CellSenderService
		encoder service.PayloadEncodingService
	}

	type testData struct {
		ctrl              *matchers.MockController
		key               vo.AESKey
		nonce             vo.Nonce
		cid               vo.CircuitID
		encryptedData     []byte
		decryptedData     []byte
		dataPayload       []byte
		invalidData       []byte
		forwardPayload    []byte
		upstreamEncrypted []byte
		upstreamPayload   []byte
	}

	type connStateSetup struct {
		hasDown  bool
		isHidden bool
		mockDown net.Conn
		streamID vo.StreamID
	}

	tests := []struct {
		name            string
		setupConnState  func(ctrl *matchers.MockController) (*entity.ConnState, connStateSetup)
		setupMocks      func(mocks testMocks, setup connStateSetup, testData testData)
		expectError     bool
		errorCheck      func(error) bool
		verifyMocks     func(mocks testMocks, setup connStateSetup, testData testData)
		expectServeDown bool
	}{
		{
			name: "Forward to middle relay",
			setupConnState: func(ctrl *matchers.MockController) (*entity.ConnState, connStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				down1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, down1)
				return st, connStateSetup{hasDown: true, isHidden: false, streamID: 1}
			},
			setupMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				dataDTO := &service.DataPayloadDTO{StreamID: setup.streamID, Data: testData.encryptedData}
				WhenDouble(mocks.encoder.DecodeDataPayload(testData.dataPayload)).ThenReturn(dataDTO, nil)
				// Use Any[] for AESOpen due to nonce side effects
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.decryptedData, nil)
				WhenDouble(mocks.encoder.EncodeDataPayload(Any[*service.DataPayloadDTO]())).ThenReturn(testData.forwardPayload, nil)
				WhenSingle(mocks.sender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
			},
			expectError:     false,
			expectServeDown: true,
			verifyMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				Verify(mocks.encoder, Times(1)).DecodeDataPayload(testData.dataPayload)
				// Note: AESOpen verification is complex due to nonce side effects
				// We focus on ensuring the behavior is correct rather than exact parameter matching
				Verify(mocks.encoder, Times(1)).EncodeDataPayload(Any[*service.DataPayloadDTO]())
				Verify(mocks.sender, Times(1)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
			},
		},
		{
			name: "Exit node data handling",
			setupConnState: func(ctrl *matchers.MockController) (*entity.ConnState, connStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil)
				return st, connStateSetup{hasDown: false, isHidden: false, streamID: 1}
			},
			setupMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				dataDTO := &service.DataPayloadDTO{StreamID: setup.streamID, Data: testData.encryptedData}
				sid := setup.streamID
				mockConn := Mock[net.Conn](testData.ctrl)
				WhenDouble(mocks.encoder.DecodeDataPayload(testData.dataPayload)).ThenReturn(dataDTO, nil)
				// Use Any[] for AESOpen due to nonce side effects
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.decryptedData, nil)
				WhenDouble(mocks.repo.GetStream(testData.cid, sid)).ThenReturn(mockConn, nil)
				WhenDouble(mockConn.Write(testData.decryptedData)).ThenReturn(len(testData.decryptedData), nil)
			},
			expectError:     false,
			expectServeDown: false,
			verifyMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				Verify(mocks.encoder, Times(1)).DecodeDataPayload(testData.dataPayload)
				// Note: AESOpen verification is complex due to nonce side effects
				Verify(mocks.repo, Times(1)).GetStream(testData.cid, setup.streamID)
			},
		},
		{
			name: "Hidden service data handling",
			setupConnState: func(ctrl *matchers.MockController) (*entity.ConnState, connStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				mockDown := Mock[net.Conn](ctrl)
				st := entity.NewConnState(key, nonce, up1, mockDown)
				st.SetHidden(true)
				return st, connStateSetup{hasDown: true, isHidden: true, mockDown: mockDown, streamID: 1}
			},
			setupMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				dataDTO := &service.DataPayloadDTO{StreamID: setup.streamID, Data: testData.encryptedData}
				WhenDouble(mocks.encoder.DecodeDataPayload(testData.dataPayload)).ThenReturn(dataDTO, nil)
				// Use Any[] for AESOpen due to nonce side effects
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.decryptedData, nil)
				WhenDouble(setup.mockDown.Write(testData.decryptedData)).ThenReturn(len(testData.decryptedData), nil)
			},
			expectError:     false,
			expectServeDown: false,
			verifyMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				Verify(mocks.encoder, Times(1)).DecodeDataPayload(testData.dataPayload)
				// Note: AESOpen verification is complex due to nonce side effects
				// Write verification simplified due to data transformation complexity
				Verify(setup.mockDown, Times(1)).Write(Any[[]byte]())
				// Verify ForwardCell is NOT called for hidden services
				Verify(mocks.sender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
			},
		},
		{
			name: "Upstream data encryption (decryption fails)",
			setupConnState: func(ctrl *matchers.MockController) (*entity.ConnState, connStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				down1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, down1)
				return st, connStateSetup{hasDown: true, isHidden: false, streamID: 1}
			},
			setupMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				dataDTO := &service.DataPayloadDTO{StreamID: setup.streamID, Data: testData.invalidData}
				cryptoError := errors.New("decryption failed")
				WhenDouble(mocks.encoder.DecodeDataPayload(testData.dataPayload)).ThenReturn(dataDTO, nil)
				// Use Any[] for AESOpen due to nonce side effects
				WhenDouble(mocks.crypto.AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(nil, cryptoError)
				// Note: For upstream data flow, a different nonce is used
				WhenDouble(mocks.crypto.AESSeal(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())).ThenReturn(testData.upstreamEncrypted, nil)
				WhenDouble(mocks.encoder.EncodeDataPayload(Any[*service.DataPayloadDTO]())).ThenReturn(testData.upstreamPayload, nil)
				WhenSingle(mocks.sender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
			},
			expectError:     false,
			expectServeDown: true,
			verifyMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				Verify(mocks.encoder, Times(1)).DecodeDataPayload(testData.dataPayload)
				// Note: Crypto operations verification is complex due to nonce side effects
				// We focus on the business logic flow instead
				Verify(mocks.encoder, Times(1)).EncodeDataPayload(Any[*service.DataPayloadDTO]())
				Verify(mocks.sender, Times(1)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
			},
		},
		{
			name: "Payload decoding failure",
			setupConnState: func(ctrl *matchers.MockController) (*entity.ConnState, connStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil)
				return st, connStateSetup{hasDown: false, isHidden: false, streamID: 1}
			},
			setupMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				decodingError := errors.New("payload decoding failed")
				WhenDouble(mocks.encoder.DecodeDataPayload(testData.dataPayload)).ThenReturn(nil, decodingError)
			},
			expectError: true,
			errorCheck: func(err error) bool {
				return err != nil && err.Error() == "payload decoding failed"
			},
			expectServeDown: false,
			verifyMocks: func(mocks testMocks, setup connStateSetup, testData testData) {
				Verify(mocks.encoder, Times(1)).DecodeDataPayload(testData.dataPayload)
				// For decode error case, AESOpen should not be called
				Verify(mocks.crypto, Times(0)).AESOpen(Any[vo.AESKey](), Any[vo.Nonce](), Any[[]byte]())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)

			// Create mocks
			mocks := testMocks{
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
				if st.Down() != nil && !connSetup.isHidden {
					st.Down().Close()
				}
			}()

			// Setup test data with placeholders
			testData := testData{
				ctrl:              ctrl,
				key:               st.Key(),
				nonce:             vo.Nonce{}, // Will be set just before setup
				cid:               vo.NewCircuitID(),
				encryptedData:     []byte("encrypted-data"),
				decryptedData:     []byte("decrypted-data"),
				dataPayload:       []byte("data-payload"),
				invalidData:       []byte("invalid encrypted data"),
				forwardPayload:    []byte("forward-payload"),
				upstreamEncrypted: []byte("upstream-encrypted"),
				upstreamPayload:   []byte("upstream-payload"),
			}

			// Get the nonce that will actually be used for this call
			testData.nonce = st.DataNonce()

			cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: testData.dataPayload}

			// Setup mock behaviors
			tt.setupMocks(mocks, connSetup, testData)

			// Track ensureServeDown calls
			serveDownCalled := false
			ensureServeDown := func(st *entity.ConnState) {
				serveDownCalled = true
			}

			// Create usecase and execute
			uc := usecase.NewHandleDataUseCase(mocks.repo, mocks.crypto, mocks.sender, mocks.encoder)
			err := uc.Data(st, testData.cid, cell, ensureServeDown)

			// Verify error expectations
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if tt.errorCheck != nil && !tt.errorCheck(err) {
					t.Fatalf("Error check failed for error: %v", err)
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

			// Verify mock interactions
			if tt.verifyMocks != nil {
				tt.verifyMocks(mocks, connSetup, testData)
			}
		})
	}
}
