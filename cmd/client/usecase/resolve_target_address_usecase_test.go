package usecase

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

var ErrHiddenServiceNotFound = errors.New("hidden service not found")

func TestResolveTargetAddressUseCase_Handle(t *testing.T) {
	tests := []struct {
		name        string
		input       ResolveTargetAddressInput
		setupMock   func(repository.HiddenServiceRepository)
		expected    ResolveTargetAddressOutput
		expectError bool
	}{
		{
			name: "Regular IPv4 address",
			input: ResolveTargetAddressInput{
				Host: "192.168.1.1",
				Port: 80,
			},
			setupMock: func(m repository.HiddenServiceRepository) {},
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
			setupMock: func(m repository.HiddenServiceRepository) {},
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
			setupMock: func(m repository.HiddenServiceRepository) {},
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
			setupMock: func(m repository.HiddenServiceRepository) {
				if mockRepo, ok := m.(interface {
					FindByAddressString(string) (*entity.HiddenService, error)
				}); ok {
					// Create a test public key
					pub, _, _ := ed25519.GenerateKey(rand.Reader)

					addr := vo.HiddenAddrFromString("test.ptor")
					relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
					pubKey := vo.Ed25519PubKey{PublicKey: pub}
					hs := entity.NewHiddenService(addr, relayID, pubKey)

					// Setup mock behavior for FindByAddressString
					WhenDouble(mockRepo.FindByAddressString(Any[string]())).ThenReturn(hs, nil)
				}
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
			setupMock: func(m repository.HiddenServiceRepository) {
				if mockRepo, ok := m.(interface {
					FindByAddressString(string) (*entity.HiddenService, error)
				}); ok {
					// Setup mock behavior for FindByAddressString to return error
					WhenDouble(mockRepo.FindByAddressString(Any[string]())).ThenReturn((*entity.HiddenService)(nil), repository.ErrNotFound)
				}
			},
			expected:    ResolveTargetAddressOutput{},
			expectError: true,
		},
		{
			name: "Case insensitive hidden service",
			input: ResolveTargetAddressInput{
				Host: "TEST.PTOR",
				Port: 443,
			},
			setupMock: func(m repository.HiddenServiceRepository) {
				if mockRepo, ok := m.(interface {
					FindByAddressString(string) (*entity.HiddenService, error)
				}); ok {
					// Create a test public key
					pub, _, _ := ed25519.GenerateKey(rand.Reader)

					addr := vo.HiddenAddrFromString("test.ptor")
					relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440001")
					pubKey := vo.Ed25519PubKey{PublicKey: pub}
					hs := entity.NewHiddenService(addr, relayID, pubKey)

					// Setup mock behavior for FindByAddressString with case insensitive match
					WhenDouble(mockRepo.FindByAddressString(Any[string]())).ThenReturn(hs, nil)
				}
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
			ctrl := NewMockController(t)
			mockRepo := Mock[repository.HiddenServiceRepository](ctrl)
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
