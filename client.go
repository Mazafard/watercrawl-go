package watercrawl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client represents the WaterCrawl API client
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	version    string
}

// NewClient creates a new WaterCrawl API client
func NewClient(apiKey string, baseURL string) *Client {

	if baseURL == "" {
		baseURL = "https://app.watercrawl.dev/"
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
		version:    Version,
	}
}

// doRequest performs an HTTP request and returns the response
func (c *Client) doRequest(ctx context.Context, method, endpoint string, queryParams url.Values, body interface{}) (*http.Response, error) {
	// Construct the full URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = endpoint
	if queryParams != nil {
		u.RawQuery = queryParams.Encode()
	}

	// Prepare request body if provided
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "WaterCrawl-Go-SDK")
	req.Header.Set("Accept-Language", "en-US")

	// For debugging
	fmt.Printf("Making request to: %s %s\n", method, u.String())

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Log response status for debugging
	fmt.Printf("Received response: %d %s\n", resp.StatusCode, resp.Status)

	return resp, nil
}

// processResponse processes the HTTP response and unmarshals the response body
func (c *Client) processResponse(resp *http.Response, v interface{}) error {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to parse error response as JSON
		var apiErr struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    apiErr.Error,
			}
		}

		// If JSON parsing fails or no error message, use raw body or default message
		errorMsg := string(body)
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("HTTP error %d", resp.StatusCode)
		}

		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    errorMsg,
		}
	}

	if v != nil {
		if err := json.Unmarshal(body, v); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
