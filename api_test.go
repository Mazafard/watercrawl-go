package watercrawl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetCrawlRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/core/crawl-requests/" {
			t.Errorf("Expected path /api/v1/core/crawl-requests/, got %s", r.URL.Path)
		}

		page := r.URL.Query().Get("page")
		if page != "1" {
			t.Errorf("Expected page=1, got %s", page)
		}
		pageSize := r.URL.Query().Get("page_size")
		if pageSize != "10" {
			t.Errorf("Expected page_size=10, got %s", pageSize)
		}

		response := CrawlRequestList{
			Count: 1,
			Results: []CrawlRequest{
				{
					UUID:   "test-uuid",
					Status: "completed",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	result, err := client.GetCrawlRequests(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetCrawlRequests() error = %v", err)
	}

	if result.Count != 1 {
		t.Errorf("GetCrawlRequests().Count = %v, want %v", result.Count, 1)
	}
	if len(result.Results) != 1 {
		t.Errorf("GetCrawlRequests().Results length = %v, want %v", len(result.Results), 1)
	}
	if result.Results[0].UUID != "test-uuid" {
		t.Errorf("GetCrawlRequests().Results[0].UUID = %v, want %v", result.Results[0].UUID, "test-uuid")
	}
}

func TestClient_CreateCrawlRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/core/crawl-requests/" {
			t.Errorf("Expected path /api/v1/core/crawl-requests/, got %s", r.URL.Path)
		}

		var input CreateCrawlRequestInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		response := CrawlRequest{
			UUID:   "test-uuid",
			URL:    input.URL,
			Status: "pending",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	input := CreateCrawlRequestInput{
		URL: "https://example.com",
		Options: CrawlOptions{
			SpiderOptions: map[string]interface{}{
				"allowed_domains": []string{"example.com"},
			},
		},
	}

	result, err := client.CreateCrawlRequest(ctx, input)
	if err != nil {
		t.Fatalf("CreateCrawlRequest() error = %v", err)
	}

	if result.UUID != "test-uuid" {
		t.Errorf("CreateCrawlRequest().UUID = %v, want %v", result.UUID, "test-uuid")
	}
	if result.Status != "pending" {
		t.Errorf("CreateCrawlRequest().Status = %v, want %v", result.Status, "pending")
	}
}

func TestClient_StopCrawlRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/core/crawl-requests/test-uuid/" {
			t.Errorf("Expected path /api/v1/core/crawl-requests/test-uuid/, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	err := client.StopCrawlRequest(ctx, "test-uuid")
	if err != nil {
		t.Fatalf("StopCrawlRequest() error = %v", err)
	}
}

func TestClient_ScrapeURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/core/crawl-requests/":
			response := CrawlRequest{
				UUID:   "test-uuid",
				Status: "pending",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		case "/api/v1/core/crawl-requests/test-uuid/status/":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(EventStreamMessage{
				Type: "result",
				Data: map[string]interface{}{
					"content": "test content",
				},
			})
		}
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	result, err := client.ScrapeURL(ctx, "https://example.com", nil, nil, true, false)
	if err != nil {
		t.Fatalf("ScrapeURL() error = %v", err)
	}

	content, ok := result["content"].(string)
	if !ok || content != "test content" {
		t.Errorf("ScrapeURL() content = %v, want %v", content, "test content")
	}
}

func TestClient_CreateCrawlRequest_Validation(t *testing.T) {
	client := NewClient("test-key", "")
	ctx := context.Background()

	tests := []struct {
		name          string
		input         CreateCrawlRequestInput
		expectedError string
	}{
		{
			name: "nil URL",
			input: CreateCrawlRequestInput{
				URL: nil,
				Options: CrawlOptions{},
			},
			expectedError: "watercrawl: validation error: url: URL is required",
		},
		{
			name: "empty string URL",
			input: CreateCrawlRequestInput{
				URL: "",
				Options: CrawlOptions{},
			},
			expectedError: "watercrawl: validation error: url: URL cannot be empty",
		},
		{
			name: "empty string array URL",
			input: CreateCrawlRequestInput{
				URL: []string{},
				Options: CrawlOptions{},
			},
			expectedError: "watercrawl: validation error: url: URL list cannot be empty",
		},
		{
			name: "array with empty string URL",
			input: CreateCrawlRequestInput{
				URL: []string{"https://example.com", ""},
				Options: CrawlOptions{},
			},
			expectedError: "watercrawl: validation error: url[1]: URL cannot be empty",
		},
		{
			name: "invalid URL type",
			input: CreateCrawlRequestInput{
				URL: 123,
				Options: CrawlOptions{},
			},
			expectedError: "watercrawl: validation error: url: URL must be a string or array of strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateCrawlRequest(ctx, tt.input)
			if err == nil {
				t.Error("Expected validation error, got nil")
				return
			}
			if err.Error() != tt.expectedError {
				t.Errorf("Expected error %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request parameters",
		})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	_, err := client.GetCrawlRequests(ctx, 1, 10)
	if err == nil {
		t.Error("Expected API error, got nil")
		return
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
		return
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, apiErr.StatusCode)
	}

	expectedMessage := "Invalid request parameters"
	if apiErr.Message != expectedMessage {
		t.Errorf("Expected message %q, got %q", expectedMessage, apiErr.Message)
	}
} 