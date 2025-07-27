package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestResponse represents a test JSON response structure
type TestResponse struct {
	Message string `json:"message"`
	Value   int    `json:"value"`
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Error("NewHTTPClient should return a non-nil client")
	}

	// Verify it implements HTTPClient interface
	var _ HTTPClient = client
}

func TestHTTPClientImpl_FetchJSON_Success(t *testing.T) {
	// Create test data
	expectedResponse := TestResponse{
		Message: "test message",
		Value:   42,
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	// Test FetchJSON
	client := NewHTTPClient()
	var result TestResponse
	err := client.FetchJSON(server.URL, &result)

	if err != nil {
		t.Fatalf("FetchJSON failed: %v", err)
	}

	if result.Message != expectedResponse.Message {
		t.Errorf("Message mismatch: got %s, want %s", result.Message, expectedResponse.Message)
	}
	if result.Value != expectedResponse.Value {
		t.Errorf("Value mismatch: got %d, want %d", result.Value, expectedResponse.Value)
	}
}

func TestHTTPClientImpl_FetchJSON_HTTPError(t *testing.T) {
	// Create test server that returns HTTP error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	// Test FetchJSON with HTTP error
	client := NewHTTPClient()
	var result TestResponse
	err := client.FetchJSON(server.URL, &result)

	if err == nil {
		t.Error("Expected error for HTTP 500 status")
	}

	expectedErrorPrefix := "unexpected status 500"
	if !strings.Contains(err.Error(), expectedErrorPrefix) {
		t.Errorf("Error should contain '%s', got: %v", expectedErrorPrefix, err)
	}
}

func TestHTTPClientImpl_FetchJSON_InvalidJSON(t *testing.T) {
	// Create test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Test FetchJSON with invalid JSON
	client := NewHTTPClient()
	var result TestResponse
	err := client.FetchJSON(server.URL, &result)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	expectedErrorPrefix := "decode JSON failed"
	if !strings.Contains(err.Error(), expectedErrorPrefix) {
		t.Errorf("Error should contain '%s', got: %v", expectedErrorPrefix, err)
	}
}

func TestHTTPClientImpl_FetchJSON_NetworkError(t *testing.T) {
	// Test with invalid URL to trigger network error
	client := NewHTTPClient()
	var result TestResponse
	err := client.FetchJSON("http://invalid-host:99999", &result)

	if err == nil {
		t.Error("Expected error for network failure")
	}

	expectedErrorPrefix := "HTTP request failed"
	if !strings.Contains(err.Error(), expectedErrorPrefix) {
		t.Errorf("Error should contain '%s', got: %v", expectedErrorPrefix, err)
	}
}

func TestHTTPClientImpl_FetchJSON_EmptyResponse(t *testing.T) {
	// Create test server that returns empty JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	// Test FetchJSON with empty JSON
	client := NewHTTPClient()
	var result TestResponse
	err := client.FetchJSON(server.URL, &result)

	if err != nil {
		t.Fatalf("FetchJSON should handle empty JSON: %v", err)
	}

	// Verify default values
	if result.Message != "" {
		t.Errorf("Expected empty message, got: %s", result.Message)
	}
	if result.Value != 0 {
		t.Errorf("Expected zero value, got: %d", result.Value)
	}
}

func TestHTTPClientImpl_FetchJSON_DifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		statusCode  int
		expectError bool
	}{
		{http.StatusOK, false},
		{http.StatusCreated, true}, // Only 200 OK is accepted
		{http.StatusBadRequest, true},
		{http.StatusNotFound, true},
		{http.StatusInternalServerError, true},
	}

	for _, tc := range testCases {
		t.Run(http.StatusText(tc.statusCode), func(t *testing.T) {
			// Create test server with specific status code
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.statusCode == http.StatusOK {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(TestResponse{Message: "ok", Value: 1})
				} else {
					http.Error(w, "Error response", tc.statusCode)
				}
			}))
			defer server.Close()

			// Test FetchJSON
			client := NewHTTPClient()
			var result TestResponse
			err := client.FetchJSON(server.URL, &result)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for status %d", tc.statusCode)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for status %d: %v", tc.statusCode, err)
			}
		})
	}
}

func TestHTTPClientImpl_FetchJSON_ContentTypes(t *testing.T) {
	testData := TestResponse{Message: "test", Value: 123}

	tests := []struct {
		name        string
		contentType string
	}{
		{"Standard JSON", "application/json"},
		{"JSON with charset", "application/json; charset=utf-8"},
		{"Plain text (flexible)", "text/plain"}, // Server can return any content type, client just tries to decode
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", test.contentType)
				json.NewEncoder(w).Encode(testData)
			}))
			defer server.Close()

			client := NewHTTPClient()
			var result TestResponse
			err := client.FetchJSON(server.URL, &result)

			if err != nil {
				t.Errorf("FetchJSON failed with content type %s: %v", test.contentType, err)
			}
			if result.Message != testData.Message {
				t.Errorf("Data mismatch with content type %s", test.contentType)
			}
		})
	}
}

func TestHTTPClientImpl_FetchJSON_LargeResponse(t *testing.T) {
	// Create a large JSON response
	largeData := make([]TestResponse, 1000)
	for i := range largeData {
		largeData[i] = TestResponse{
			Message: strings.Repeat("x", 100),
			Value:   i,
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(largeData)
	}))
	defer server.Close()

	client := NewHTTPClient()
	var result []TestResponse
	err := client.FetchJSON(server.URL, &result)

	if err != nil {
		t.Fatalf("FetchJSON failed with large response: %v", err)
	}

	if len(result) != len(largeData) {
		t.Errorf("Response length mismatch: got %d, want %d", len(result), len(largeData))
	}
}

func TestHTTPClientImpl_FetchJSON_NilResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	// Pass nil as result to test error handling
	err := client.FetchJSON(server.URL, nil)

	if err == nil {
		t.Error("Expected error when passing nil result")
	}
}

func TestHTTPClientImpl_FetchJSON_HTTPMethods(t *testing.T) {
	// Test that FetchJSON uses GET method
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TestResponse{Message: "ok", Value: 1})
	}))
	defer server.Close()

	client := NewHTTPClient()
	var result TestResponse
	err := client.FetchJSON(server.URL, &result)

	if err != nil {
		t.Fatalf("FetchJSON failed: %v", err)
	}

	if receivedMethod != "GET" {
		t.Errorf("Expected GET method, got: %s", receivedMethod)
	}
}
