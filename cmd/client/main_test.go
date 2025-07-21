package main

import (
	"encoding/json"
	"testing"

	"ikedadada/go-ptor/cmd/client/handler"
	"ikedadada/go-ptor/cmd/client/infrastructure/repository"
)

// Mock HTTPClient for testing
type mockHTTPClient struct {
	response interface{}
	err      error
}

func (m *mockHTTPClient) FetchJSON(url string, result interface{}) error {
	if m.err != nil {
		return m.err
	}
	// Simulate JSON unmarshaling
	data, _ := json.Marshal(m.response)
	return json.Unmarshal(data, result)
}

func TestResolveAddress_CaseInsensitive(t *testing.T) {
	// Create mock HTTP client with test data in new array format
	type hiddenServiceDTO struct {
		Address string `json:"address"`
		Relay   string `json:"relay"`
		PubKey  string `json:"pubkey"`
	}

	mockClient := &mockHTTPClient{
		response: []hiddenServiceDTO{
			{
				Address: "lower.ptor",
				Relay:   "550e8400-e29b-41d4-a716-446655440000",
				PubKey:  "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41\nfGnJm6gOdrj8ym3rFkEjWT2btf0QYfIXS7n/K/5gUneqQzpYwXRzP5xYrxRHLZUK\n/HPNdmqwSKKmRAx5bqLDEP/TrE/fgJ2KBJHEpLv4T/ZrIZz4VZkj7mwwF6VLzCz2\nIRFsVVmF4v5XF5sGLa4y8a/q/8jDYhpRqW7JK1QLpTVHJr8GqIa6QA1GcwWqh5rV\n6U8oLzQm8vOhQvhM3v4sVYd6q7Dl+QHe5Nm5q1u2gZO/0UG1ZgA2p0Qq0LGhQVvQ\nqwZO/xXvE8xF0hc8UEBxhwp1v1m9A1k3J1kA5mZ6t8X5W7jv7sZrCQ9J5H7m4aZK\nLQIDAQAB\n-----END PUBLIC KEY-----",
			},
		},
	}

	hsRepo, err := repository.NewHiddenServiceRepository(mockClient, "http://test.com")
	if err != nil {
		t.Fatalf("NewHiddenServiceRepository: %v", err)
	}

	// Create a minimal SOCKS5Controller for testing
	controller := handler.NewSOCKS5Controller(
		hsRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, 0,
	)

	addr, exit, err := controller.ResolveAddress("LOWER.PTOR", 80)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if exit != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("unexpected exit: %s", exit)
	}
	if addr != "lower.ptor:80" {
		t.Fatalf("unexpected addr: %s", addr)
	}
}
