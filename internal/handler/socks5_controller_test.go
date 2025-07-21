package handler

import (
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
)

// Mock implementations for testing

type mockHiddenServiceRepo struct {
	services map[string]*entity.HiddenService
	err      error
}

func (m *mockHiddenServiceRepo) FindByAddressString(address string) (*entity.HiddenService, error) {
	if m.err != nil {
		return nil, m.err
	}
	if hs, found := m.services[address]; found {
		return hs, nil
	}
	return nil, errors.New("hidden service not found")
}

func (m *mockHiddenServiceRepo) FindByAddress(addr value_object.HiddenAddr) (*entity.HiddenService, error) {
	return m.FindByAddressString(addr.String())
}

func (m *mockHiddenServiceRepo) Save(hs *entity.HiddenService) error {
	if m.services == nil {
		m.services = make(map[string]*entity.HiddenService)
	}
	m.services[hs.Address().String()] = hs
	return m.err
}

func (m *mockHiddenServiceRepo) All() ([]*entity.HiddenService, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*entity.HiddenService
	for _, hs := range m.services {
		result = append(result, hs)
	}
	return result, nil
}

type mockCircuitRepo struct {
	circuits map[value_object.CircuitID]*entity.Circuit
	err      error
}

func (m *mockCircuitRepo) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	if m.err != nil {
		return nil, m.err
	}
	if circuit, found := m.circuits[id]; found {
		return circuit, nil
	}
	return nil, errors.New("circuit not found")
}

func (m *mockCircuitRepo) Save(circuit *entity.Circuit) error {
	if m.circuits == nil {
		m.circuits = make(map[value_object.CircuitID]*entity.Circuit)
	}
	m.circuits[circuit.ID()] = circuit
	return m.err
}

func (m *mockCircuitRepo) Delete(id value_object.CircuitID) error {
	if m.err != nil {
		return m.err
	}
	delete(m.circuits, id)
	return nil
}

func (m *mockCircuitRepo) ListActive() ([]*entity.Circuit, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*entity.Circuit
	for _, circuit := range m.circuits {
		result = append(result, circuit)
	}
	return result, nil
}

type mockCryptoService struct{}

func (m *mockCryptoService) RSAEncrypt(pub *rsa.PublicKey, in []byte) ([]byte, error) {
	return in, nil
}

func (m *mockCryptoService) RSADecrypt(priv *rsa.PrivateKey, in []byte) ([]byte, error) {
	return in, nil
}

func (m *mockCryptoService) AESSeal(key [32]byte, nonce [12]byte, plain []byte) ([]byte, error) {
	return plain, nil
}

func (m *mockCryptoService) AESOpen(key [32]byte, nonce [12]byte, enc []byte) ([]byte, error) {
	return enc, nil
}

func (m *mockCryptoService) AESMultiSeal(keys [][32]byte, nonces [][12]byte, plain []byte) ([]byte, error) {
	return plain, nil
}

func (m *mockCryptoService) AESMultiOpen(keys [][32]byte, nonces [][12]byte, enc []byte) ([]byte, error) {
	return enc, nil
}

func (m *mockCryptoService) X25519Generate() (priv, pub []byte, err error) {
	return make([]byte, 32), make([]byte, 32), nil
}

func (m *mockCryptoService) X25519Shared(priv, pub []byte) ([]byte, error) {
	return make([]byte, 32), nil
}

func (m *mockCryptoService) DeriveKeyNonce(secret []byte) ([32]byte, [12]byte, error) {
	var key [32]byte
	var nonce [12]byte
	return key, nonce, nil
}

func (m *mockCryptoService) ModifyNonceWithSequence(baseNonce [12]byte, sequence uint64) [12]byte {
	return baseNonce
}

type mockCellReaderService struct{}

func (m *mockCellReaderService) ReadCell(r io.Reader) (value_object.CircuitID, *entity.Cell, error) {
	return value_object.NewCircuitID(), nil, errors.New("mock read cell")
}

type mockStreamManagerService struct{}

func (m *mockStreamManagerService) Add(id uint16, conn net.Conn)       {}
func (m *mockStreamManagerService) Get(id uint16) (net.Conn, bool)     { return nil, false }
func (m *mockStreamManagerService) Remove(id uint16)                    {}
func (m *mockStreamManagerService) CloseAll()                          {}

// Simple mock implementations for use cases
type mockBuildCircuitUseCase struct{}
func (m *mockBuildCircuitUseCase) Handle(input usecase.BuildCircuitInput) (usecase.BuildCircuitOutput, error) {
	return usecase.BuildCircuitOutput{}, nil
}

type mockConnectUseCase struct{}
func (m *mockConnectUseCase) Handle(input usecase.ConnectInput) (usecase.ConnectOutput, error) {
	return usecase.ConnectOutput{}, nil
}

type mockOpenStreamUseCase struct{}
func (m *mockOpenStreamUseCase) Handle(input usecase.OpenStreamInput) (usecase.OpenStreamOutput, error) {
	return usecase.OpenStreamOutput{}, nil
}

type mockCloseStreamUseCase struct{}
func (m *mockCloseStreamUseCase) Handle(input usecase.CloseStreamInput) (usecase.CloseStreamOutput, error) {
	return usecase.CloseStreamOutput{}, nil
}

type mockSendDataUseCase struct{}
func (m *mockSendDataUseCase) Handle(input usecase.SendDataInput) (usecase.SendDataOutput, error) {
	return usecase.SendDataOutput{}, nil
}

type mockHandleEndUseCase struct{}
func (m *mockHandleEndUseCase) Handle(input usecase.HandleEndInput) (usecase.HandleEndOutput, error) {
	return usecase.HandleEndOutput{}, nil
}

func createTestController() *SOCKS5Controller {
	return NewSOCKS5Controller(
		&mockHiddenServiceRepo{},
		&mockCircuitRepo{},
		&mockCryptoService{},
		&mockCellReaderService{},
		&mockBuildCircuitUseCase{},
		&mockConnectUseCase{},
		&mockOpenStreamUseCase{},
		&mockCloseStreamUseCase{},
		&mockSendDataUseCase{},
		&mockHandleEndUseCase{},
		3, // hops
	)
}

func TestNewSOCKS5Controller(t *testing.T) {
	controller := createTestController()
	
	if controller == nil {
		t.Fatal("SOCKS5Controller should not be nil")
	}
	
	if controller.hops != 3 {
		t.Errorf("Expected hops to be 3, got %d", controller.hops)
	}
}

func TestSOCKS5Controller_ResolveAddress_BasicCases(t *testing.T) {
	controller := createTestController()
	
	tests := []struct {
		name         string
		host         string
		port         int
		expectedAddr string
		expectedExit string
	}{
		{
			name:         "IPv4 address",
			host:         "192.168.1.1",
			port:         80,
			expectedAddr: "192.168.1.1:80",
			expectedExit: "",
		},
		{
			name:         "Domain name",
			host:         "example.com",
			port:         443,
			expectedAddr: "example.com:443",
			expectedExit: "",
		},
		{
			name:         "Localhost",
			host:         "localhost",
			port:         8080,
			expectedAddr: "localhost:8080",
			expectedExit: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, exit, err := controller.ResolveAddress(tt.host, tt.port)
			if err != nil {
				t.Fatalf("ResolveAddress failed: %v", err)
			}
			
			if addr != tt.expectedAddr {
				t.Errorf("Expected address %s, got %s", tt.expectedAddr, addr)
			}
			
			if exit != tt.expectedExit {
				t.Errorf("Expected exit %s, got %s", tt.expectedExit, exit)
			}
		})
	}
}

func TestSOCKS5Controller_ResolveAddress_IPv6(t *testing.T) {
	controller := createTestController()
	
	tests := []struct {
		name         string
		host         string
		port         int
		expectedAddr string
		expectedExit string
	}{
		{
			name:         "IPv6 address",
			host:         "2001:db8::1",
			port:         80,
			expectedAddr: "[2001:db8::1]:80",
			expectedExit: "",
		},
		{
			name:         "IPv6 loopback",
			host:         "::1",
			port:         443,
			expectedAddr: "[::1]:443",
			expectedExit: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, exit, err := controller.ResolveAddress(tt.host, tt.port)
			if err != nil {
				t.Fatalf("ResolveAddress failed: %v", err)
			}
			
			if addr != tt.expectedAddr {
				t.Errorf("Expected address %s, got %s", tt.expectedAddr, addr)
			}
			
			if exit != tt.expectedExit {
				t.Errorf("Expected exit %s, got %s", tt.expectedExit, exit)
			}
		})
	}
}

func TestSOCKS5Controller_ResolveAddress_HiddenService_NotFound(t *testing.T) {
	controller := createTestController()
	
	_, _, err := controller.ResolveAddress("nonexistent.ptor", 80)
	if err == nil {
		t.Error("Expected ResolveAddress to fail for non-existent hidden service")
	}
	
	expectedError := "hidden service not found"
	if err != nil && !stringContains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedError, err.Error())
	}
}

func TestSOCKS5Controller_ResolveAddress_CaseInsensitive(t *testing.T) {
	controller := createTestController()
	
	tests := []struct {
		name         string
		host         string
		port         int
		expectedAddr string
	}{
		{
			name:         "Uppercase domain",
			host:         "EXAMPLE.COM",
			port:         80,
			expectedAddr: "example.com:80",
		},
		{
			name:         "Mixed case domain",
			host:         "Example.Com",
			port:         443,
			expectedAddr: "example.com:443",
		},
		{
			name:         "Uppercase IPv6",
			host:         "2001:DB8::1",
			port:         80,
			expectedAddr: "[2001:db8::1]:80",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, _, err := controller.ResolveAddress(tt.host, tt.port)
			if err != nil {
				t.Fatalf("ResolveAddress failed: %v", err)
			}
			
			if addr != tt.expectedAddr {
				t.Errorf("Expected address %s, got %s", tt.expectedAddr, addr)
			}
		})
	}
}

func TestSOCKS5Controller_ResolveAddress_PortHandling(t *testing.T) {
	controller := createTestController()
	
	tests := []struct {
		name         string
		host         string
		port         int
		expectedAddr string
	}{
		{
			name:         "Standard HTTP port",
			host:         "example.com",
			port:         80,
			expectedAddr: "example.com:80",
		},
		{
			name:         "Standard HTTPS port",
			host:         "example.com",
			port:         443,
			expectedAddr: "example.com:443",
		},
		{
			name:         "High port number",
			host:         "example.com",
			port:         65535,
			expectedAddr: "example.com:65535",
		},
		{
			name:         "Low port number",
			host:         "example.com",
			port:         1,
			expectedAddr: "example.com:1",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, _, err := controller.ResolveAddress(tt.host, tt.port)
			if err != nil {
				t.Fatalf("ResolveAddress failed: %v", err)
			}
			
			if addr != tt.expectedAddr {
				t.Errorf("Expected address %s, got %s", tt.expectedAddr, addr)
			}
		})
	}
}

func TestSOCKS5Controller_ResolveAddress_HiddenService_Success(t *testing.T) {
	// Create a test hidden service
	relayID, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	
	// Use deterministic key for consistent testing
	testSeed := make([]byte, ed25519.SeedSize)
	for i := range testSeed {
		testSeed[i] = byte(i)
	}
	priv := ed25519.NewKeyFromSeed(testSeed)
	pub := priv.Public().(ed25519.PublicKey)
	
	// Create hidden address and public key
	hiddenAddr := value_object.NewHiddenAddr(pub)
	pubKey := value_object.Ed25519PubKey{PublicKey: pub}
	
	hiddenService := entity.NewHiddenService(hiddenAddr, relayID, pubKey)
	
	// Setup mock repository with the hidden service
	// Use lowercase key since the controller converts to lowercase
	hsRepo := &mockHiddenServiceRepo{
		services: map[string]*entity.HiddenService{
			strings.ToLower(hiddenAddr.String()): hiddenService,
		},
	}
	
	controller := NewSOCKS5Controller(
		hsRepo,
		&mockCircuitRepo{},
		&mockCryptoService{},
		&mockCellReaderService{},
		&mockBuildCircuitUseCase{},
		&mockConnectUseCase{},
		&mockOpenStreamUseCase{},
		&mockCloseStreamUseCase{},
		&mockSendDataUseCase{},
		&mockHandleEndUseCase{},
		3,
	)
	
	addr, exit, err := controller.ResolveAddress(hiddenAddr.String(), 80)
	if err != nil {
		t.Fatalf("ResolveAddress failed for hidden service: %v", err)
	}
	
	expectedAddr := strings.ToLower(hiddenAddr.String()) + ":80"
	if addr != expectedAddr {
		t.Errorf("Expected address %s, got %s", expectedAddr, addr)
	}
	
	expectedExit := relayID.String()
	if exit != expectedExit {
		t.Errorf("Expected exit %s, got %s", expectedExit, exit)
	}
}

// Helper function to check if a string contains a substring
func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}