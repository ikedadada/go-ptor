package handler_test

import (
	"errors"
	"net"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"

	"ikedadada/go-ptor/cmd/relay/handler"
	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestRelayHandler_HandleCellExtend(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: []byte("extend-payload")}
	conn := &net.TCPConn{}

	// Mock circuit not found (new circuit scenario)
	WhenDouble(mockRepo.Find(cid)).ThenReturn(nil, repository.ErrNotFound)

	// Mock successful extend operation
	WhenSingle(mockExtendUC.Extend(conn, cid, cell)).ThenReturn(nil)

	// Execute
	err := h.HandleCell(conn, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	Verify(mockExtendUC, Times(1)).Extend(conn, cid, cell)
}

func TestRelayHandler_HandleCellBeginAck(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdBeginAck, Version: vo.ProtocolV1}
	conn := &net.TCPConn{}

	// Create mock connection state
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := &net.TCPConn{}
	st := entity.NewConnState(key, nonce, upConn, nil)

	// Mock circuit found
	WhenDouble(mockRepo.Find(cid)).ThenReturn(st, nil)

	// Mock forwarding cell upstream
	WhenSingle(mockSender.ForwardCell(st.Up(), cid, cell)).ThenReturn(nil)

	// Execute
	err := h.HandleCell(conn, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	Verify(mockSender, Times(1)).ForwardCell(st.Up(), cid, cell)
}

func TestRelayHandler_HandleCellDestroy(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
	conn := &net.TCPConn{}

	// Create mock connection state
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := &net.TCPConn{}
	downConn := &net.TCPConn{}
	st := entity.NewConnState(key, nonce, upConn, downConn)

	// Mock circuit found
	WhenDouble(mockRepo.Find(cid)).ThenReturn(st, nil)

	// Mock destroy operation
	WhenSingle(mockDestroyUC.Destroy(st, cid)).ThenReturn(nil)

	// Execute
	err := h.HandleCell(conn, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	Verify(mockDestroyUC, Times(1)).Destroy(st, cid)
}

func TestRelayHandler_HandleCellEndUnknown(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: nil}
	conn := &net.TCPConn{}

	// Mock circuit not found (unknown circuit scenario)
	WhenDouble(mockRepo.Find(cid)).ThenReturn(nil, repository.ErrNotFound)

	// Execute - should ignore end for unknown circuit
	err := h.HandleCell(conn, cid, cell)

	// Verify - no error should be returned, and no usecases should be called
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	// Verify that no usecases were called since the circuit is unknown and command is End
	Verify(mockExtendUC, Never()).Extend(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
	Verify(mockEndStreamUC, Never()).EndStream(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())
}

func TestRelayHandler_ServeConn(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	conn := &net.TCPConn{}
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: []byte("extend-payload")}

	// Mock reading cells - first return a cell, then EOF to terminate the loop
	When(mockReader.ReadCell(conn)).ThenReturn(cid, cell, nil).ThenReturn(vo.CircuitID{}, nil, errors.New("EOF"))

	// Mock circuit not found (new extend scenario)
	WhenDouble(mockRepo.Find(cid)).ThenReturn(nil, repository.ErrNotFound)

	// Mock successful extend operation
	WhenSingle(mockExtendUC.Extend(conn, cid, cell)).ThenReturn(nil)

	// Execute ServeConn - it should read one cell, handle it, then exit on EOF
	h.ServeConn(conn)

	// Verify
	Verify(mockReader, Times(2)).ReadCell(conn)
	Verify(mockRepo, Times(1)).Find(cid)
	Verify(mockExtendUC, Times(1)).Extend(conn, cid, cell)
}

// Additional test cases for different cell commands

func TestRelayHandler_HandleCellBegin(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1}
	conn := &net.TCPConn{}

	// Create mock connection state
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := &net.TCPConn{}
	st := entity.NewConnState(key, nonce, upConn, nil)

	// Mock circuit found
	WhenDouble(mockRepo.Find(cid)).ThenReturn(st, nil)

	// Mock begin operation
	WhenSingle(mockBeginUC.Begin(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())).ThenReturn(nil)

	// Execute
	err := h.HandleCell(conn, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	Verify(mockBeginUC, Times(1)).Begin(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())
}

func TestRelayHandler_HandleCellData(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: []byte("data-payload")}
	conn := &net.TCPConn{}

	// Create mock connection state
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := &net.TCPConn{}
	st := entity.NewConnState(key, nonce, upConn, nil)

	// Mock circuit found
	WhenDouble(mockRepo.Find(cid)).ThenReturn(st, nil)

	// Mock data operation
	WhenSingle(mockDataUC.Data(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())).ThenReturn(nil)

	// Execute
	err := h.HandleCell(conn, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	Verify(mockDataUC, Times(1)).Data(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())
}

// Additional test cases for comprehensive coverage

func TestRelayHandler_HandleCellConnect(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1}
	conn := &net.TCPConn{}

	// Create mock connection state
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := &net.TCPConn{}
	st := entity.NewConnState(key, nonce, upConn, nil)

	// Mock circuit found
	WhenDouble(mockRepo.Find(cid)).ThenReturn(st, nil)

	// Mock connect operation
	WhenSingle(mockConnectUC.Connect(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())).ThenReturn(nil)

	// Execute
	err := h.HandleCell(conn, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	Verify(mockConnectUC, Times(1)).Connect(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())
}

func TestRelayHandler_HandleCellExtendForwarding(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: []byte("extend-payload")}
	conn := &net.TCPConn{}

	// Create mock connection state (existing circuit)
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := &net.TCPConn{}
	st := entity.NewConnState(key, nonce, upConn, nil)

	// Mock circuit found (existing circuit - should forward)
	WhenDouble(mockRepo.Find(cid)).ThenReturn(st, nil)

	// Mock forward extend operation
	WhenSingle(mockExtendUC.ForwardExtend(st, cid, cell)).ThenReturn(nil)

	// Execute
	err := h.HandleCell(conn, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	Verify(mockExtendUC, Times(1)).ForwardExtend(st, cid, cell)
}

func TestRelayHandler_HandleCellEnd(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1}
	conn := &net.TCPConn{}

	// Create mock connection state (existing circuit)
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := &net.TCPConn{}
	downConn := &net.TCPConn{}
	st := entity.NewConnState(key, nonce, upConn, downConn)

	// Mock circuit found
	WhenDouble(mockRepo.Find(cid)).ThenReturn(st, nil)

	// Mock end stream operation
	WhenSingle(mockEndStreamUC.EndStream(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())).ThenReturn(nil)

	// Execute
	err := h.HandleCell(conn, cid, cell)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	Verify(mockEndStreamUC, Times(1)).EndStream(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())
}

func TestRelayHandler_HandleCellUnknownCommand(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data with unknown command
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CellCommand(255), Version: vo.ProtocolV1} // Unknown command (255 is within byte range but not a defined command)
	conn := &net.TCPConn{}

	// Create mock connection state
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := &net.TCPConn{}
	st := entity.NewConnState(key, nonce, upConn, nil)

	// Mock circuit found
	WhenDouble(mockRepo.Find(cid)).ThenReturn(st, nil)

	// Execute - should handle unknown command gracefully
	err := h.HandleCell(conn, cid, cell)

	// Verify - no error should be returned for unknown commands
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	// Verify that no usecases were called for unknown command
	Verify(mockExtendUC, Never()).Extend(Any[net.Conn](), Any[vo.CircuitID](), Any[*entity.Cell]())
	Verify(mockExtendUC, Never()).ForwardExtend(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell]())
}

func TestRelayHandler_HandleCellRepositoryError(t *testing.T) {
	ctrl := NewMockController(t)

	// Create mocks
	mockRepo := Mock[repository.ConnStateRepository](ctrl)
	mockReader := Mock[service.CellReaderService](ctrl)
	mockSender := Mock[service.CellSenderService](ctrl)
	mockExtendUC := Mock[usecase.HandleExtendUseCase](ctrl)
	mockBeginUC := Mock[usecase.HandleBeginUseCase](ctrl)
	mockDataUC := Mock[usecase.HandleDataUseCase](ctrl)
	mockEndStreamUC := Mock[usecase.HandleEndStreamUseCase](ctrl)
	mockDestroyUC := Mock[usecase.HandleDestroyUseCase](ctrl)
	mockConnectUC := Mock[usecase.HandleConnectUseCase](ctrl)

	h := handler.NewRelayHandler(mockRepo, mockReader, mockSender, mockExtendUC, mockBeginUC, mockDataUC, mockEndStreamUC, mockDestroyUC, mockConnectUC)

	// Test data
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1}
	conn := &net.TCPConn{}

	// Mock repository error (not ErrNotFound)
	repoErr := errors.New("database connection error")
	WhenDouble(mockRepo.Find(cid)).ThenReturn(nil, repoErr)

	// Execute
	err := h.HandleCell(conn, cid, cell)

	// Verify - should return the repository error
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repository error but got: %v", err)
	}
	Verify(mockRepo, Times(1)).Find(cid)
	// Verify that no usecases were called due to repository error
	Verify(mockDataUC, Never()).Data(Any[*entity.ConnState](), Any[vo.CircuitID](), Any[*entity.Cell](), Any[func(*entity.ConnState)]())
}
