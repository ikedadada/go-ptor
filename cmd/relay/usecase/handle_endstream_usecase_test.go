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

func TestHandleEndStreamUseCase_EndStream(t *testing.T) {
	type endstreamTestMocks struct {
		repo    repository.ConnStateRepository
		sender  service.CellSenderService
		encoder service.PayloadEncodingService
	}

	type endstreamConnStateSetup struct {
		hasDown bool
	}

	type endstreamExpectedCalls struct {
		decodeDataPayload int
		removeStream      int
		delete            int
		forwardCell       int
	}

	type endstreamTestData struct {
		payload     []byte
		isAllStream bool // true for empty payload (end all streams)
		streamID    vo.StreamID
	}

	tests := []struct {
		name            string
		setupConn       func() (*entity.ConnState, endstreamConnStateSetup)
		setupTestData   func() endstreamTestData
		setupMocks      func(mocks endstreamTestMocks, cid vo.CircuitID, testData endstreamTestData, setup endstreamConnStateSetup)
		expectError     bool
		errorCheck      func(error) bool
		expectedCalls   endstreamExpectedCalls
		expectServeDown bool
	}{
		{
			name: "End specific stream - exit node",
			setupConn: func() (*entity.ConnState, endstreamConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil)
				return st, endstreamConnStateSetup{hasDown: false}
			},
			setupTestData: func() endstreamTestData {
				return endstreamTestData{
					payload:     []byte("end-payload"),
					isAllStream: false,
					streamID:    vo.StreamID(1),
				}
			},
			setupMocks: func(mocks endstreamTestMocks, cid vo.CircuitID, testData endstreamTestData, setup endstreamConnStateSetup) {
				dataDTO := &service.DataPayloadDTO{StreamID: testData.streamID}
				WhenDouble(mocks.encoder.DecodeDataPayload(testData.payload)).ThenReturn(dataDTO, nil)
				sid, _ := vo.StreamIDFrom(uint16(testData.streamID))
				WhenSingle(mocks.repo.RemoveStream(cid, sid)).ThenReturn(nil)
			},
			expectError:     false,
			expectedCalls:   endstreamExpectedCalls{decodeDataPayload: 1, removeStream: 1, delete: 0, forwardCell: 0},
			expectServeDown: false,
		},
		{
			name: "End all streams - exit node",
			setupConn: func() (*entity.ConnState, endstreamConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil)
				return st, endstreamConnStateSetup{hasDown: false}
			},
			setupTestData: func() endstreamTestData {
				return endstreamTestData{
					payload:     []byte{}, // Empty payload means end all streams
					isAllStream: true,
					streamID:    vo.StreamID(0),
				}
			},
			setupMocks: func(mocks endstreamTestMocks, cid vo.CircuitID, testData endstreamTestData, setup endstreamConnStateSetup) {
				WhenSingle(mocks.repo.Delete(cid)).ThenReturn(nil)
			},
			expectError:     false,
			expectedCalls:   endstreamExpectedCalls{decodeDataPayload: 0, removeStream: 0, delete: 1, forwardCell: 0},
			expectServeDown: false,
		},
		{
			name: "End specific stream with forward - middle relay",
			setupConn: func() (*entity.ConnState, endstreamConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				down1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, down1)
				return st, endstreamConnStateSetup{hasDown: true}
			},
			setupTestData: func() endstreamTestData {
				return endstreamTestData{
					payload:     []byte("end-payload"),
					isAllStream: false,
					streamID:    vo.StreamID(1),
				}
			},
			setupMocks: func(mocks endstreamTestMocks, cid vo.CircuitID, testData endstreamTestData, setup endstreamConnStateSetup) {
				dataDTO := &service.DataPayloadDTO{StreamID: testData.streamID}
				WhenDouble(mocks.encoder.DecodeDataPayload(testData.payload)).ThenReturn(dataDTO, nil)
				sid, _ := vo.StreamIDFrom(uint16(testData.streamID))
				WhenSingle(mocks.repo.RemoveStream(cid, sid)).ThenReturn(nil)
				WhenSingle(mocks.sender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
			},
			expectError:     false,
			expectedCalls:   endstreamExpectedCalls{decodeDataPayload: 1, removeStream: 1, delete: 0, forwardCell: 1},
			expectServeDown: true,
		},
		{
			name: "Payload decoding failure",
			setupConn: func() (*entity.ConnState, endstreamConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil)
				return st, endstreamConnStateSetup{hasDown: false}
			},
			setupTestData: func() endstreamTestData {
				return endstreamTestData{
					payload:     []byte("invalid-payload"),
					isAllStream: false,
					streamID:    vo.StreamID(1),
				}
			},
			setupMocks: func(mocks endstreamTestMocks, cid vo.CircuitID, testData endstreamTestData, setup endstreamConnStateSetup) {
				decodingError := errors.New("payload decoding failed")
				WhenDouble(mocks.encoder.DecodeDataPayload(testData.payload)).ThenReturn(nil, decodingError)
			},
			expectError: true,
			errorCheck: func(err error) bool {
				return err != nil && err.Error() == "payload decoding failed"
			},
			expectedCalls:   endstreamExpectedCalls{decodeDataPayload: 1, removeStream: 0, delete: 0, forwardCell: 0},
			expectServeDown: false,
		},
		{
			name: "End all streams with forward - middle relay",
			setupConn: func() (*entity.ConnState, endstreamConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				down1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, down1)
				return st, endstreamConnStateSetup{hasDown: true}
			},
			setupTestData: func() endstreamTestData {
				return endstreamTestData{
					payload:     []byte{}, // Empty payload means end all streams
					isAllStream: true,
					streamID:    vo.StreamID(0),
				}
			},
			setupMocks: func(mocks endstreamTestMocks, cid vo.CircuitID, testData endstreamTestData, setup endstreamConnStateSetup) {
				WhenSingle(mocks.sender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
				WhenSingle(mocks.repo.Delete(cid)).ThenReturn(nil)
			},
			expectError:     false,
			expectedCalls:   endstreamExpectedCalls{decodeDataPayload: 0, removeStream: 0, delete: 1, forwardCell: 1},
			expectServeDown: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)

			// Create mocks
			mocks := endstreamTestMocks{
				repo:    Mock[repository.ConnStateRepository](ctrl),
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
			cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: testData.payload}

			// Setup mock behaviors
			tt.setupMocks(mocks, cid, testData, connSetup)

			// Track ensureServeDown calls
			serveDownCalled := false
			ensureServeDown := func(st *entity.ConnState) {
				serveDownCalled = true
			}

			// Create usecase and execute
			uc := usecase.NewHandleEndStreamUseCase(mocks.repo, mocks.sender, mocks.encoder)
			err := uc.EndStream(st, cid, cell, ensureServeDown)

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

			// Verify ensureServeDown call
			if tt.expectServeDown && !serveDownCalled {
				t.Errorf("Expected ensureServeDown to be called")
			}
			if !tt.expectServeDown && serveDownCalled {
				t.Errorf("Unexpected ensureServeDown call")
			}

			// Verify mock interactions using parameterized call counts
			Verify(mocks.encoder, Times(tt.expectedCalls.decodeDataPayload)).DecodeDataPayload(Any[[]byte]())
			Verify(mocks.repo, Times(tt.expectedCalls.removeStream)).RemoveStream(Any[vo.CircuitID](), Any[vo.StreamID]())
			Verify(mocks.repo, Times(tt.expectedCalls.delete)).Delete(Any[vo.CircuitID]())
			Verify(mocks.sender, Times(tt.expectedCalls.forwardCell)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
		})
	}
}
