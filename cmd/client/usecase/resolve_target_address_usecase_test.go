package usecase

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

var ErrHiddenServiceNotFound = errors.New("hidden service not found")

// mockHiddenServiceRepository for testing
type mockHiddenServiceRepository struct {
	services map[string]*entity.HiddenService
}

func newMockHiddenServiceRepository() *mockHiddenServiceRepository {
	return &mockHiddenServiceRepository{
		services: make(map[string]*entity.HiddenService),
	}
}

func (m *mockHiddenServiceRepository) FindByAddress(address vo.HiddenAddr) (*entity.HiddenService, error) {
	hs, exists := m.services[address.String()]
	if !exists {
		return nil, ErrHiddenServiceNotFound
	}
	return hs, nil
}

func (m *mockHiddenServiceRepository) FindByAddressString(address string) (*entity.HiddenService, error) {
	hs, exists := m.services[address]
	if !exists {
		return nil, ErrHiddenServiceNotFound
	}
	return hs, nil
}

func (m *mockHiddenServiceRepository) Save(hs *entity.HiddenService) error {
	m.services[hs.Address().String()] = hs
	return nil
}

func (m *mockHiddenServiceRepository) All() ([]*entity.HiddenService, error) {
	var result []*entity.HiddenService
	for _, hs := range m.services {
		result = append(result, hs)
	}
	return result, nil
}

func TestResolveTargetAddressUseCase_Handle(t *testing.T) {
	tests := []struct {
		name        string
		input       ResolveTargetAddressInput
		setupMock   func(*mockHiddenServiceRepository)
		expected    ResolveTargetAddressOutput
		expectError bool
	}{
		{
			name: "Regular IPv4 address",
			input: ResolveTargetAddressInput{
				Host: "192.168.1.1",
				Port: 80,
			},
			setupMock: func(m *mockHiddenServiceRepository) {},
			expected: ResolveTargetAddressOutput{
				DialAddress: "192.168.1.1:80",
				ExitRelayID: "",
			},
			expectError: false,
		},
		{
			name: "Regular hostname",
			input: ResolveTargetAddressInput{
				Host: "example.com",
				Port: 443,
			},
			setupMock: func(m *mockHiddenServiceRepository) {},
			expected: ResolveTargetAddressOutput{
				DialAddress: "example.com:443",
				ExitRelayID: "",
			},
			expectError: false,
		},
		{
			name: "IPv6 address",
			input: ResolveTargetAddressInput{
				Host: "2001:db8::1",
				Port: 80,
			},
			setupMock: func(m *mockHiddenServiceRepository) {},
			expected: ResolveTargetAddressOutput{
				DialAddress: "[2001:db8::1]:80",
				ExitRelayID: "",
			},
			expectError: false,
		},
		{
			name: "Hidden service address - found",
			input: ResolveTargetAddressInput{
				Host: "test.ptor",
				Port: 80,
			},
			setupMock: func(m *mockHiddenServiceRepository) {
				// Create a test public key
				pub, _, _ := ed25519.GenerateKey(rand.Reader)

				addr := vo.HiddenAddrFromString("test.ptor")
				relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
				pubKey := vo.Ed25519PubKey{PublicKey: pub}
				hs := entity.NewHiddenService(addr, relayID, pubKey)
				m.Save(hs)
			},
			expected: ResolveTargetAddressOutput{
				DialAddress: "test.ptor:80",
				ExitRelayID: "550e8400-e29b-41d4-a716-446655440000",
			},
			expectError: false,
		},
		{
			name: "Hidden service address - not found",
			input: ResolveTargetAddressInput{
				Host: "notfound.ptor",
				Port: 80,
			},
			setupMock:   func(m *mockHiddenServiceRepository) {},
			expected:    ResolveTargetAddressOutput{},
			expectError: true,
		},
		{
			name: "Case insensitive hidden service",
			input: ResolveTargetAddressInput{
				Host: "TEST.PTOR",
				Port: 443,
			},
			setupMock: func(m *mockHiddenServiceRepository) {
				// Create a test public key
				pub, _, _ := ed25519.GenerateKey(rand.Reader)

				addr := vo.HiddenAddrFromString("test.ptor")
				relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440001")
				pubKey := vo.Ed25519PubKey{PublicKey: pub}
				hs := entity.NewHiddenService(addr, relayID, pubKey)
				m.Save(hs)
			},
			expected: ResolveTargetAddressOutput{
				DialAddress: "test.ptor:443",
				ExitRelayID: "550e8400-e29b-41d4-a716-446655440001",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := newMockHiddenServiceRepository()
			tt.setupMock(mockRepo)

			uc := NewResolveTargetAddressUseCase(mockRepo)
			result, err := uc.Handle(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.DialAddress != tt.expected.DialAddress {
				t.Errorf("DialAddress = %v, want %v", result.DialAddress, tt.expected.DialAddress)
			}

			if result.ExitRelayID != tt.expected.ExitRelayID {
				t.Errorf("ExitRelayID = %v, want %v", result.ExitRelayID, tt.expected.ExitRelayID)
			}
		})
	}
}
