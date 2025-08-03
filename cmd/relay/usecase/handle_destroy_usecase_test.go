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
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)

	uc := usecase.NewHandleDestroyUseCase(mockRepo, mockSender)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()

	// Create mock connection (exit node - no down connection)
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	// Configure mocks for exit node scenario (st.Down() == nil)
	WhenSingle(mockRepo.Delete(cid)).ThenReturn(nil)

	// Execute
	err := uc.Destroy(st, cid)

	// Verify
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockRepo, Times(1)).Delete(Any[vo.CircuitID]())
	// Verify ForwardCell is not called for exit nodes
	Verify(mockSender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
}

func TestHandleDestroyUseCase_DestroyWithDownstream(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)

	uc := usecase.NewHandleDestroyUseCase(mockRepo, mockSender)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()

	// Create mock connections (middle relay - has down connection)
	up1, _ := net.Pipe()
	down1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)

	// Configure mocks for middle relay scenario (st.Down() != nil)
	WhenSingle(mockSender.ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())).ThenReturn(nil)
	WhenSingle(mockRepo.Delete(cid)).ThenReturn(nil)

	// Execute
	err := uc.Destroy(st, cid)

	// Verify
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	// Verify mock interactions
	Verify(mockSender, Times(1)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
	Verify(mockRepo, Times(1)).Delete(Any[vo.CircuitID]())

	st.Up().Close()
	st.Down().Close()
}

// Test repository delete failure
func TestHandleDestroyUseCase_RepositoryDeleteFailure(t *testing.T) {
	ctrl := NewMockController(t)

	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)

	uc := usecase.NewHandleDestroyUseCase(mockRepo, mockSender)

	// Setup test data
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()

	// Create mock connection (exit node)
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)

	// Configure mocks - the destroy usecase ignores delete errors and always returns nil
	WhenSingle(mockRepo.Delete(cid)).ThenReturn(errors.New("delete failed"))

	// Execute
	err := uc.Destroy(st, cid)

	// Verify - destroy always returns nil, even if delete fails
	if err != nil {
		t.Fatalf("Destroy should not fail even if delete fails: %v", err)
	}

	// Verify mock interactions
	Verify(mockRepo, Times(1)).Delete(Any[vo.CircuitID]())
	Verify(mockSender, Times(0)).ForwardCell(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())

	st.Up().Close()
}
