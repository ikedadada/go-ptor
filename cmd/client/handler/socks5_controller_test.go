package handler

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/service"
)

// Mock connection for testing
type mockConnection struct {
	readData  []byte
	writeData bytes.Buffer
	closed    bool
}

func (m *mockConnection) Read(b []byte) (int, error) {
	if len(m.readData) == 0 {
		return 0, io.EOF
	}
	n := copy(b, m.readData)
	m.readData = m.readData[n:]
	return n, nil
}

func (m *mockConnection) Write(b []byte) (int, error) {
	return m.writeData.Write(b)
}

func (m *mockConnection) Close() error {
	m.closed = true
	return nil
}

func (m *mockConnection) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1080}
}

func (m *mockConnection) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (m *mockConnection) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

// Mock UseCase implementations
type mockBuildCircuitUseCase struct {
	circuitID string
	err       error
}

func (m *mockBuildCircuitUseCase) Handle(in usecase.BuildCircuitInput) (usecase.BuildCircuitOutput, error) {
	if m.err != nil {
		return usecase.BuildCircuitOutput{}, m.err
	}
	return usecase.BuildCircuitOutput{CircuitID: m.circuitID}, nil
}

type mockResolveTargetAddressUseCase struct {
	dialAddress string
	exitRelayID string
	err         error
}

func (m *mockResolveTargetAddressUseCase) Handle(in usecase.ResolveTargetAddressInput) (usecase.ResolveTargetAddressOutput, error) {
	if m.err != nil {
		return usecase.ResolveTargetAddressOutput{}, m.err
	}
	return usecase.ResolveTargetAddressOutput{
		DialAddress: m.dialAddress,
		ExitRelayID: m.exitRelayID,
	}, nil
}

type mockSendConnectUseCase struct {
	err error
}

func (m *mockSendConnectUseCase) Handle(in usecase.SendConnectInput) (usecase.SendConnectOutput, error) {
	if m.err != nil {
		return usecase.SendConnectOutput{}, m.err
	}
	return usecase.SendConnectOutput{Sent: true}, nil
}

type mockOpenStreamUseCase struct {
	streamID int
	err      error
}

func (m *mockOpenStreamUseCase) Handle(in usecase.OpenStreamInput) (usecase.OpenStreamOutput, error) {
	if m.err != nil {
		return usecase.OpenStreamOutput{}, m.err
	}
	return usecase.OpenStreamOutput{StreamID: uint16(m.streamID)}, nil
}

type mockSendDataUseCase struct {
	err error
}

func (m *mockSendDataUseCase) Handle(in usecase.SendDataInput) (usecase.SendDataOutput, error) {
	if m.err != nil {
		return usecase.SendDataOutput{}, m.err
	}
	return usecase.SendDataOutput{}, nil
}

type mockCloseStreamUseCase struct {
	err error
}

func (m *mockCloseStreamUseCase) Handle(in usecase.CloseStreamInput) (usecase.CloseStreamOutput, error) {
	if m.err != nil {
		return usecase.CloseStreamOutput{}, m.err
	}
	return usecase.CloseStreamOutput{}, nil
}

type mockHandleEndUseCase struct {
	err error
}

func (m *mockHandleEndUseCase) Handle(in usecase.HandleEndInput) (usecase.HandleEndOutput, error) {
	if m.err != nil {
		return usecase.HandleEndOutput{}, m.err
	}
	return usecase.HandleEndOutput{}, nil
}

type mockReceiveCellUseCase struct {
	cell    *entity.Cell
	circuit *entity.Circuit
	isEOF   bool
	err     error
}

func (m *mockReceiveCellUseCase) Handle(in usecase.ReceiveCellInput) (usecase.ReceiveCellOutput, error) {
	if m.err != nil {
		return usecase.ReceiveCellOutput{}, m.err
	}
	return usecase.ReceiveCellOutput{
		Cell:    m.cell,
		Circuit: m.circuit,
		IsEOF:   m.isEOF,
	}, nil
}

type mockDecryptCellDataUseCase struct {
	cellData    *usecase.DecryptedCellData
	shouldClose bool
	err         error
}

func (m *mockDecryptCellDataUseCase) Handle(in usecase.DecryptCellDataInput) (usecase.DecryptCellDataOutput, error) {
	if m.err != nil {
		return usecase.DecryptCellDataOutput{}, m.err
	}
	return usecase.DecryptCellDataOutput{
		CellData:    m.cellData,
		ShouldClose: m.shouldClose,
	}, nil
}

type mockPayloadEncodingService struct {
	beginPayload []byte
	err          error
}

func (m *mockPayloadEncodingService) EncodeExtendPayload(dto *service.ExtendPayloadDTO) ([]byte, error) {
	return nil, nil
}

func (m *mockPayloadEncodingService) DecodeExtendPayload(data []byte) (*service.ExtendPayloadDTO, error) {
	return nil, nil
}

func (m *mockPayloadEncodingService) EncodeCreatedPayload(dto *service.CreatedPayloadDTO) ([]byte, error) {
	return nil, nil
}

func (m *mockPayloadEncodingService) DecodeCreatedPayload(data []byte) (*service.CreatedPayloadDTO, error) {
	return nil, nil
}

func (m *mockPayloadEncodingService) EncodeBeginPayload(dto *service.BeginPayloadDTO) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.beginPayload, nil
}

func (m *mockPayloadEncodingService) DecodeBeginPayload(data []byte) (*service.BeginPayloadDTO, error) {
	return nil, nil
}

func (m *mockPayloadEncodingService) EncodeConnectPayload(dto *service.ConnectPayloadDTO) ([]byte, error) {
	return nil, nil
}

func (m *mockPayloadEncodingService) DecodeConnectPayload(data []byte) (*service.ConnectPayloadDTO, error) {
	return nil, nil
}

func (m *mockPayloadEncodingService) EncodeDataPayload(dto *service.DataPayloadDTO) ([]byte, error) {
	return nil, nil
}

func (m *mockPayloadEncodingService) DecodeDataPayload(data []byte) (*service.DataPayloadDTO, error) {
	return nil, nil
}

type mockStreamManagerService struct {
	streams map[uint16]net.Conn
}

func (m *mockStreamManagerService) Add(id uint16, conn net.Conn) {
	if m.streams == nil {
		m.streams = make(map[uint16]net.Conn)
	}
	m.streams[id] = conn
}

func (m *mockStreamManagerService) Get(id uint16) (net.Conn, bool) {
	conn, ok := m.streams[id]
	return conn, ok
}

func (m *mockStreamManagerService) Remove(id uint16) {
	if m.streams != nil {
		delete(m.streams, id)
	}
}

func (m *mockStreamManagerService) CloseAll() {
	if m.streams != nil {
		for _, conn := range m.streams {
			conn.Close()
		}
		m.streams = make(map[uint16]net.Conn)
	}
}

func TestSOCKS5Controller_HandleConnection_InvalidSOCKS5Request(t *testing.T) {
	// Create mock connection with invalid SOCKS5 data
	conn := &mockConnection{
		readData: []byte{0x04}, // Invalid SOCKS version (should be 0x05)
	}

	// Create controller with minimal mocks
	controller := NewSOCKS5Controller(
		nil, nil, nil, nil, nil, nil, // UseCases won't be called due to early error
		&mockResolveTargetAddressUseCase{},
		nil, nil,
		&mockPayloadEncodingService{},
		&mockStreamManagerService{},
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	if !conn.closed {
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

	conn := &mockConnection{
		readData: socks5Data,
	}

	// Create mocks
	resolveUC := &mockResolveTargetAddressUseCase{
		dialAddress: "google.com:80",
		exitRelayID: "",
	}
	buildUC := &mockBuildCircuitUseCase{
		circuitID: "test-circuit-123",
	}

	controller := NewSOCKS5Controller(
		buildUC,
		&mockSendConnectUseCase{},
		&mockOpenStreamUseCase{streamID: 1},
		&mockCloseStreamUseCase{},
		&mockSendDataUseCase{},
		&mockHandleEndUseCase{},
		resolveUC,
		&mockReceiveCellUseCase{isEOF: true}, // Will cause recvLoop to exit immediately
		&mockDecryptCellDataUseCase{},
		&mockPayloadEncodingService{beginPayload: []byte("begin-payload")},
		&mockStreamManagerService{},
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	if !conn.closed {
		t.Error("Expected connection to be closed after handling")
	}

	// Check that handshake response was written
	writtenData := conn.writeData.Bytes()
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

	conn := &mockConnection{
		readData: socks5Data,
	}

	resolveUC := &mockResolveTargetAddressUseCase{
		dialAddress: "192.168.1.1:8080",
		exitRelayID: "",
	}
	buildUC := &mockBuildCircuitUseCase{
		circuitID: "test-circuit-456",
	}

	controller := NewSOCKS5Controller(
		buildUC,
		&mockSendConnectUseCase{},
		&mockOpenStreamUseCase{streamID: 2},
		&mockCloseStreamUseCase{},
		&mockSendDataUseCase{},
		&mockHandleEndUseCase{},
		resolveUC,
		&mockReceiveCellUseCase{isEOF: true},
		&mockDecryptCellDataUseCase{},
		&mockPayloadEncodingService{beginPayload: []byte("begin-payload")},
		&mockStreamManagerService{},
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	if !conn.closed {
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

	conn := &mockConnection{
		readData: socks5Data,
	}

	resolveUC := &mockResolveTargetAddressUseCase{
		dialAddress: "test.ptor:80",
		exitRelayID: "exit-relay-123", // Hidden service has exit relay ID
	}
	buildUC := &mockBuildCircuitUseCase{
		circuitID: "hidden-circuit-789",
	}

	controller := NewSOCKS5Controller(
		buildUC,
		&mockSendConnectUseCase{}, // This will be called for hidden service
		&mockOpenStreamUseCase{streamID: 3},
		&mockCloseStreamUseCase{},
		&mockSendDataUseCase{},
		&mockHandleEndUseCase{},
		resolveUC,
		&mockReceiveCellUseCase{isEOF: true},
		&mockDecryptCellDataUseCase{},
		&mockPayloadEncodingService{beginPayload: []byte("begin-payload")},
		&mockStreamManagerService{},
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	if !conn.closed {
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

	conn := &mockConnection{
		readData: socks5Data,
	}

	controller := NewSOCKS5Controller(
		nil, nil, nil, nil, nil, nil, // UseCases won't be called due to parsing error
		&mockResolveTargetAddressUseCase{},
		nil, nil,
		&mockPayloadEncodingService{},
		&mockStreamManagerService{},
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	if !conn.closed {
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

	conn := &mockConnection{
		readData: socks5Data,
	}

	controller := NewSOCKS5Controller(
		nil, nil, nil, nil, nil, nil, // UseCases won't be called due to unsupported command
		&mockResolveTargetAddressUseCase{},
		nil, nil,
		&mockPayloadEncodingService{},
		&mockStreamManagerService{},
		3,
	)

	// Test
	controller.HandleConnection(conn)

	// Assertions
	if !conn.closed {
		t.Error("Expected connection to be closed after unsupported command")
	}
}
