package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPClient defines the interface for HTTP operations
type HTTPClient interface {
	FetchJSON(url string, result interface{}) error
}

// HTTPClientImpl is the standard HTTP client implementation
type HTTPClientImpl struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTP client
func NewHTTPClient() HTTPClient {
	return &HTTPClientImpl{
		client: &http.Client{},
	}
}

func (d *HTTPClientImpl) FetchJSON(url string, result interface{}) error {
	resp, err := d.client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %s: %s", resp.Status, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode JSON failed: %w", err)
	}

	return nil
}
