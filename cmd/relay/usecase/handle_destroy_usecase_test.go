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

func TestHandleDestroyUseCase_Destroy(t *testing.T) {
	type destroyTestMocks struct {
		repo   repository.ConnStateRepository
		sender service.CellSenderService
	}

	type destroyConnStateSetup struct {
		hasDown bool
	}

	type destroyExpectedCalls struct {
		delete      int
		forwardCell int
	}
	tests := []struct {
		name          string
		setupConn     func() (*entity.ConnState, destroyConnStateSetup)
		setupMocks    func(mocks destroyTestMocks, cid vo.CircuitID, setup destroyConnStateSetup)
		expectError   bool
		expectedCalls destroyExpectedCalls
	}{
		{
			name: "Exit node scenario",
			setupConn: func() (*entity.ConnState, destroyConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil)
				return st, destroyConnStateSetup{hasDown: false}
			},
			setupMocks: func(mocks destroyTestMocks, cid vo.CircuitID, setup destroyConnStateSetup) {
				WhenSingle(mocks.repo.Delete(cid)).ThenReturn(nil)
			},
			expectError:   false,
			expectedCalls: destroyExpectedCalls{delete: 1, forwardCell: 0},
		},
		{
			name: "Middle relay scenario",
			setupConn: func() (*entity.ConnState, destroyConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				down1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, down1)
				return st, destroyConnStateSetup{hasDown: true}
			},
			setupMocks: func(mocks destroyTestMocks, cid vo.CircuitID, setup destroyConnStateSetup) {
				WhenSingle(mocks.sender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
				WhenSingle(mocks.repo.Delete(cid)).ThenReturn(nil)
			},
			expectError:   false,
			expectedCalls: destroyExpectedCalls{delete: 1, forwardCell: 1},
		},
		{
			name: "Repository delete failure",
			setupConn: func() (*entity.ConnState, destroyConnStateSetup) {
				key, _ := vo.NewAESKey()
				nonce, _ := vo.NewNonce()
				up1, _ := net.Pipe()
				st := entity.NewConnState(key, nonce, up1, nil)
				return st, destroyConnStateSetup{hasDown: false}
			},
			setupMocks: func(mocks destroyTestMocks, cid vo.CircuitID, setup destroyConnStateSetup) {
				// Destroy usecase ignores delete errors and always returns nil
				WhenSingle(mocks.repo.Delete(cid)).ThenReturn(errors.New("delete failed"))
			},
			expectError:   false,
			expectedCalls: destroyExpectedCalls{delete: 1, forwardCell: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)

			// Create mocks
			mocks := destroyTestMocks{
				repo:   Mock[repository.ConnStateRepository](ctrl),
				sender: Mock[service.CellSenderService](ctrl),
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

			cid := vo.NewCircuitID()

			// Setup mock behaviors
			tt.setupMocks(mocks, cid, connSetup)

			// Create usecase and execute
			uc := usecase.NewHandleDestroyUseCase(mocks.repo, mocks.sender)
			err := uc.Destroy(st, cid)

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

			// Verify mock interactions using parameterized call counts
			Verify(mocks.repo, Times(tt.expectedCalls.delete)).Delete(Any[vo.CircuitID]())
			Verify(mocks.sender, Times(tt.expectedCalls.forwardCell)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
		})
	}
}
