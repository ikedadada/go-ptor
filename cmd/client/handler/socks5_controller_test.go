package handler

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ovechkin-dm/mockio/v2/matchers"
	. "github.com/ovechkin-dm/mockio/v2/mock"
	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/service"
)

// Helper struct to track connection state for tests
type connState struct {
	readData  []byte
	writeData bytes.Buffer
	mu        sync.Mutex
	closed    bool
}

// Helper function to create connection mock with stateful behavior
func createMockConnection(ctrl *matchers.MockController, readData []byte) (net.Conn, *connState) {
	mockConn := Mock[net.Conn](ctrl)
	state := &connState{readData: make([]byte, len(readData))}
	copy(state.readData, readData) // Make a copy to avoid modifying original

	// Set up Read behavior to consume data progressively
	WhenDouble(mockConn.Read(Any[[]byte]())).ThenAnswer(func(args []any) (int, error) {
		b := args[0].([]byte)
		state.mu.Lock()
		defer state.mu.Unlock()
		if len(state.readData) == 0 {
			return 0, io.EOF
		}
		n := copy(b, state.readData)
		state.readData = state.readData[n:]
		return n, nil
	})

	// Set up Write behavior to capture data
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenAnswer(func(args []any) (int, error) {
		b := args[0].([]byte)
		state.mu.Lock()
		defer state.mu.Unlock()
		return state.writeData.Write(b)
	})

	// Set up Close behavior
	WhenSingle(mockConn.Close()).ThenAnswer(func(args []any) error {
		state.mu.Lock()
		defer state.mu.Unlock()
		state.closed = true
		return nil
	})

	// Set up address methods
	WhenSingle(mockConn.LocalAddr()).ThenReturn(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1080})
	WhenSingle(mockConn.RemoteAddr()).ThenReturn(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345})

	// Set up deadline methods
	WhenSingle(mockConn.SetDeadline(Any[time.Time]())).ThenReturn(nil)
	WhenSingle(mockConn.SetReadDeadline(Any[time.Time]())).ThenReturn(nil)
	WhenSingle(mockConn.SetWriteDeadline(Any[time.Time]())).ThenReturn(nil)

	return mockConn, state
}

func TestSOCKS5Controller_HandleConnection_InvalidSOCKS5Request(t *testing.T) {
	// Create controller with Mockio v2 mocks
	ctrl := NewMockController(t)
	conn, state := createMockConnection(ctrl, []byte{0x04}) // Invalid SOCKS version (should be 0x05)
	resolveUC := Mock[usecase.ResolveTargetAddressUseCase](ctrl)
	payloadService := Mock[service.PayloadEncodingService](ctrl)
	streamManager := Mock[service.StreamManagerService](ctrl)

	// Create controller with minimal mocks (UseCases won't be called due to early error)
	controller := NewSOCKS5Controller(
		nil, nil, nil, nil, nil, nil, // UseCases won't be called due to early error
		resolveUC,
		nil, nil,
		payloadService,
		streamManager,
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	state.mu.Lock()
	closed := state.closed
	state.mu.Unlock()

	if !closed {
		t.Error("Expected connection to be closed after invalid SOCKS5 request")
	}
}

func TestSOCKS5Controller_HandleConnection_SOCKS5ProtocolParsing(t *testing.T) {
	// Create valid SOCKS5 handshake and connect request for google.com:80
	socks5Data := []byte{
		// Handshake
		0x05, 0x01, 0x00, // Version 5, 1 method, no auth
		// Connect request
		0x05, 0x01, 0x00, 0x03, // Version 5, CONNECT, reserved, domain name
		0x0a,                                             // Domain length (10)
		'g', 'o', 'o', 'g', 'l', 'e', '.', 'c', 'o', 'm', // google.com
		0x00, 0x50, // Port 80
	}

	// Create Mockio v2 mocks
	ctrl := NewMockController(t)
	conn, state := createMockConnection(ctrl, socks5Data)
	resolveUC := Mock[usecase.ResolveTargetAddressUseCase](ctrl)
	buildUC := Mock[usecase.BuildCircuitUseCase](ctrl)
	sendConnectUC := Mock[usecase.SendConnectUseCase](ctrl)
	openStreamUC := Mock[usecase.OpenStreamUseCase](ctrl)
	closeStreamUC := Mock[usecase.CloseStreamUseCase](ctrl)
	sendDataUC := Mock[usecase.SendDataUseCase](ctrl)
	handleEndUC := Mock[usecase.HandleEndUseCase](ctrl)
	receiveCellUC := Mock[usecase.ReceiveCellUseCase](ctrl)
	decryptCellUC := Mock[usecase.DecryptCellDataUseCase](ctrl)
	payloadService := Mock[service.PayloadEncodingService](ctrl)
	streamManager := Mock[service.StreamManagerService](ctrl)

	// Setup mock behaviors
	WhenDouble(resolveUC.Handle(Any[usecase.ResolveTargetAddressInput]())).ThenReturn(usecase.ResolveTargetAddressOutput{
		DialAddress: "google.com:80",
		ExitRelayID: "",
	}, nil)
	WhenDouble(buildUC.Handle(Any[usecase.BuildCircuitInput]())).ThenReturn(usecase.BuildCircuitOutput{
		CircuitID: "test-circuit-123",
	}, nil)
	WhenDouble(sendConnectUC.Handle(Any[usecase.SendConnectInput]())).ThenReturn(usecase.SendConnectOutput{Sent: true}, nil)
	WhenDouble(openStreamUC.Handle(Any[usecase.OpenStreamInput]())).ThenReturn(usecase.OpenStreamOutput{StreamID: uint16(1)}, nil)
	WhenDouble(receiveCellUC.Handle(Any[usecase.ReceiveCellInput]())).ThenReturn(usecase.ReceiveCellOutput{IsEOF: true}, nil)
	WhenDouble(payloadService.EncodeBeginPayload(Any[*service.BeginPayloadDTO]())).ThenReturn([]byte("begin-payload"), nil)

	controller := NewSOCKS5Controller(
		buildUC,
		sendConnectUC,
		openStreamUC,
		closeStreamUC,
		sendDataUC,
		handleEndUC,
		resolveUC,
		receiveCellUC, // Will cause recvLoop to exit immediately
		decryptCellUC,
		payloadService,
		streamManager,
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	state.mu.Lock()
	closed := state.closed
	writtenData := state.writeData.Bytes()
	state.mu.Unlock()

	if !closed {
		t.Error("Expected connection to be closed after handling")
	}

	// Check that handshake response was written
	if len(writtenData) < 2 {
		t.Error("Expected handshake response to be written")
	}
	if writtenData[0] != 0x05 || writtenData[1] != 0x00 {
		t.Error("Expected valid SOCKS5 handshake response")
	}
}

func TestSOCKS5Controller_HandleConnection_IPv4Address(t *testing.T) {
	// Create SOCKS5 request with IPv4 address (192.168.1.1:8080)
	socks5Data := []byte{
		// Handshake
		0x05, 0x01, 0x00,
		// Connect request with IPv4
		0x05, 0x01, 0x00, 0x01, // Version 5, CONNECT, reserved, IPv4
		192, 168, 1, 1, // IP address
		0x1f, 0x90, // Port 8080
	}

	// Create Mockio v2 mocks
	ctrl := NewMockController(t)
	conn, state := createMockConnection(ctrl, socks5Data)
	resolveUC := Mock[usecase.ResolveTargetAddressUseCase](ctrl)
	buildUC := Mock[usecase.BuildCircuitUseCase](ctrl)
	sendConnectUC := Mock[usecase.SendConnectUseCase](ctrl)
	openStreamUC := Mock[usecase.OpenStreamUseCase](ctrl)
	closeStreamUC := Mock[usecase.CloseStreamUseCase](ctrl)
	sendDataUC := Mock[usecase.SendDataUseCase](ctrl)
	handleEndUC := Mock[usecase.HandleEndUseCase](ctrl)
	receiveCellUC := Mock[usecase.ReceiveCellUseCase](ctrl)
	decryptCellUC := Mock[usecase.DecryptCellDataUseCase](ctrl)
	payloadService := Mock[service.PayloadEncodingService](ctrl)
	streamManager := Mock[service.StreamManagerService](ctrl)

	// Setup mock behaviors
	WhenDouble(resolveUC.Handle(Any[usecase.ResolveTargetAddressInput]())).ThenReturn(usecase.ResolveTargetAddressOutput{
		DialAddress: "192.168.1.1:8080",
		ExitRelayID: "",
	}, nil)
	WhenDouble(buildUC.Handle(Any[usecase.BuildCircuitInput]())).ThenReturn(usecase.BuildCircuitOutput{
		CircuitID: "test-circuit-456",
	}, nil)
	WhenDouble(sendConnectUC.Handle(Any[usecase.SendConnectInput]())).ThenReturn(usecase.SendConnectOutput{Sent: true}, nil)
	WhenDouble(openStreamUC.Handle(Any[usecase.OpenStreamInput]())).ThenReturn(usecase.OpenStreamOutput{StreamID: uint16(2)}, nil)
	WhenDouble(receiveCellUC.Handle(Any[usecase.ReceiveCellInput]())).ThenReturn(usecase.ReceiveCellOutput{IsEOF: true}, nil)
	WhenDouble(payloadService.EncodeBeginPayload(Any[*service.BeginPayloadDTO]())).ThenReturn([]byte("begin-payload"), nil)

	controller := NewSOCKS5Controller(
		buildUC,
		sendConnectUC,
		openStreamUC,
		closeStreamUC,
		sendDataUC,
		handleEndUC,
		resolveUC,
		receiveCellUC,
		decryptCellUC,
		payloadService,
		streamManager,
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	state.mu.Lock()
	closed := state.closed
	state.mu.Unlock()

	if !closed {
		t.Error("Expected connection to be closed after handling")
	}
}

func TestSOCKS5Controller_HandleConnection_HiddenService(t *testing.T) {
	// Create SOCKS5 request for hidden service (.ptor domain)
	socks5Data := []byte{
		// Handshake
		0x05, 0x01, 0x00,
		// Connect request
		0x05, 0x01, 0x00, 0x03, // Version 5, CONNECT, reserved, domain name
		0x0c,                                                            // Domain length (12)
		't', 'e', 's', 't', '.', 'p', 't', 'o', 'r', '.', 'o', 'r', 'g', // test.ptor.org (13 chars, but length says 12 - will be truncated)
		0x00, 0x50, // Port 80
	}

	// Create Mockio v2 mocks
	ctrl := NewMockController(t)
	conn, state := createMockConnection(ctrl, socks5Data)
	resolveUC := Mock[usecase.ResolveTargetAddressUseCase](ctrl)
	buildUC := Mock[usecase.BuildCircuitUseCase](ctrl)
	sendConnectUC := Mock[usecase.SendConnectUseCase](ctrl)
	openStreamUC := Mock[usecase.OpenStreamUseCase](ctrl)
	closeStreamUC := Mock[usecase.CloseStreamUseCase](ctrl)
	sendDataUC := Mock[usecase.SendDataUseCase](ctrl)
	handleEndUC := Mock[usecase.HandleEndUseCase](ctrl)
	receiveCellUC := Mock[usecase.ReceiveCellUseCase](ctrl)
	decryptCellUC := Mock[usecase.DecryptCellDataUseCase](ctrl)
	payloadService := Mock[service.PayloadEncodingService](ctrl)
	streamManager := Mock[service.StreamManagerService](ctrl)

	// Setup mock behaviors
	WhenDouble(resolveUC.Handle(Any[usecase.ResolveTargetAddressInput]())).ThenReturn(usecase.ResolveTargetAddressOutput{
		DialAddress: "test.ptor:80",
		ExitRelayID: "exit-relay-123", // Hidden service has exit relay ID
	}, nil)
	WhenDouble(buildUC.Handle(Any[usecase.BuildCircuitInput]())).ThenReturn(usecase.BuildCircuitOutput{
		CircuitID: "hidden-circuit-789",
	}, nil)
	WhenDouble(sendConnectUC.Handle(Any[usecase.SendConnectInput]())).ThenReturn(usecase.SendConnectOutput{Sent: true}, nil)
	WhenDouble(openStreamUC.Handle(Any[usecase.OpenStreamInput]())).ThenReturn(usecase.OpenStreamOutput{StreamID: uint16(3)}, nil)
	WhenDouble(receiveCellUC.Handle(Any[usecase.ReceiveCellInput]())).ThenReturn(usecase.ReceiveCellOutput{IsEOF: true}, nil)
	WhenDouble(payloadService.EncodeBeginPayload(Any[*service.BeginPayloadDTO]())).ThenReturn([]byte("begin-payload"), nil)

	controller := NewSOCKS5Controller(
		buildUC,
		sendConnectUC, // This will be called for hidden service
		openStreamUC,
		closeStreamUC,
		sendDataUC,
		handleEndUC,
		resolveUC,
		receiveCellUC,
		decryptCellUC,
		payloadService,
		streamManager,
		3,
	)

	// Test
	done := make(chan struct{})
	go func() {
		controller.HandleConnection(conn)
		close(done)
	}()

	// Wait for completion
	<-done

	// Small delay to ensure all goroutines finish
	time.Sleep(10 * time.Millisecond)

	// Assertions
	state.mu.Lock()
	closed := state.closed
	state.mu.Unlock()

	if !closed {
		t.Error("Expected connection to be closed after handling")
	}
}

func TestSOCKS5Controller_HandleConnection_UnsupportedAddressType(t *testing.T) {
	// Create SOCKS5 request with unsupported address type
	socks5Data := []byte{
		// Handshake
		0x05, 0x01, 0x00,
		// Connect request with unsupported address type
		0x05, 0x01, 0x00, 0x05, // Version 5, CONNECT, reserved, unsupported type (0x05)
	}

	// Create controller with Mockio v2 mocks
	ctrl := NewMockController(t)
	conn, state := createMockConnection(ctrl, socks5Data)
	resolveUC := Mock[usecase.ResolveTargetAddressUseCase](ctrl)
	payloadService := Mock[service.PayloadEncodingService](ctrl)
	streamManager := Mock[service.StreamManagerService](ctrl)

	controller := NewSOCKS5Controller(
		nil, nil, nil, nil, nil, nil, // UseCases won't be called due to parsing error
		resolveUC,
		nil, nil,
		payloadService,
		streamManager,
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	state.mu.Lock()
	closed := state.closed
	state.mu.Unlock()

	if !closed {
		t.Error("Expected connection to be closed after unsupported address type")
	}
}

func TestSOCKS5Controller_HandleConnection_UnsupportedCommand(t *testing.T) {
	// Create SOCKS5 request with unsupported command (BIND instead of CONNECT)
	socks5Data := []byte{
		// Handshake
		0x05, 0x01, 0x00,
		// Request with BIND command
		0x05, 0x02, 0x00, 0x01, // Version 5, BIND (0x02), reserved, IPv4
		127, 0, 0, 1, // 127.0.0.1
		0x00, 0x50, // Port 80
	}

	// Create controller with Mockio v2 mocks
	ctrl := NewMockController(t)
	conn, state := createMockConnection(ctrl, socks5Data)
	resolveUC := Mock[usecase.ResolveTargetAddressUseCase](ctrl)
	payloadService := Mock[service.PayloadEncodingService](ctrl)
	streamManager := Mock[service.StreamManagerService](ctrl)

	controller := NewSOCKS5Controller(
		nil, nil, nil, nil, nil, nil, // UseCases won't be called due to unsupported command
		resolveUC,
		nil, nil,
		payloadService,
		streamManager,
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	state.mu.Lock()
	closed := state.closed
	state.mu.Unlock()

	if !closed {
		t.Error("Expected connection to be closed after unsupported command")
	}
}
