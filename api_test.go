package watercrawl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
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
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
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
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		case "/api/v1/core/crawl-requests/test-uuid/status/":
			// Set SSE headers
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Send state event
			stateData, err := json.Marshal(EventStreamMessage{
				Type: "state",
				Data: map[string]interface{}{
					"uuid":   "test-uuid",
					"url":    "https://example.com",
					"status": "running",
				},
			})
			if err != nil {
				t.Errorf("Failed to marshal state data: %v", err)
			}

			_, err = fmt.Fprintf(w, "data: %s\n\n", stateData)
			if err != nil {
				t.Errorf("Failed to write state event: %v", err)
			}

			// Send result event
			eventData, err := json.Marshal(EventStreamMessage{
				Type: "result",
				Data: map[string]interface{}{
					"content": "test content",
				},
			})
			if err != nil {
				t.Errorf("Failed to marshal result data: %v", err)
			}

			_, err = fmt.Fprintf(w, "data: %s\n\n", eventData)
			if err != nil {
				t.Errorf("Failed to write result event: %v", err)
			}

			// Flush data immediately
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
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

func TestClient_ScrapeURL_StateOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/core/crawl-requests/":
			response := CrawlRequest{
				UUID:   "test-uuid",
				Status: "pending",
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		case "/api/v1/core/crawl-requests/test-uuid/status/":
			// Set SSE headers
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Send state event with completed status
			stateData, err := json.Marshal(EventStreamMessage{
				Type: "state",
				Data: map[string]interface{}{
					"uuid":   "test-uuid",
					"url":    "https://example.com",
					"status": "completed",
				},
			})
			if err != nil {
				t.Errorf("Failed to marshal state data: %v", err)
			}

			_, err = fmt.Fprintf(w, "data: %s\n\n", stateData)
			if err != nil {
				t.Errorf("Failed to write state event: %v", err)
			}

			// No result event, only state

			// Flush data immediately
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	result, err := client.ScrapeURL(ctx, "https://example.com", nil, nil, true, false)
	if err != nil {
		t.Fatalf("ScrapeURL() error = %v", err)
	}

	// Check if we got the state data
	status, ok := result["status"].(string)
	if !ok || status != "completed" {
		t.Errorf("ScrapeURL() status = %v, want completed", status)
	}
}

func TestClient_MonitorCrawlRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/core/crawl-requests/test-uuid/status/" {
			t.Errorf("Expected path /api/v1/core/crawl-requests/test-uuid/status/, got %s", r.URL.Path)
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Send multiple SSE events
		// State event
		stateData, err := json.Marshal(EventStreamMessage{
			Type: "state",
			Data: map[string]interface{}{
				"uuid":   "test-uuid",
				"status": "running",
			},
		})
		if err != nil {
			t.Errorf("Failed to marshal state data: %v", err)
		}

		_, err = fmt.Fprintf(w, "data: %s\n\n", stateData)
		if err != nil {
			t.Errorf("Failed to write state event: %v", err)
		}

		// Progress event
		progressData, err := json.Marshal(EventStreamMessage{
			Type: "progress",
			Data: map[string]interface{}{
				"progress": 50.0,
			},
		})
		if err != nil {
			t.Errorf("Failed to marshal progress data: %v", err)
		}

		_, err = fmt.Fprintf(w, "data: %s\n\n", progressData)
		if err != nil {
			t.Errorf("Failed to write progress event: %v", err)
		}

		// Result event
		resultData, err := json.Marshal(EventStreamMessage{
			Type: "result",
			Data: map[string]interface{}{
				"content": "test content",
			},
		})
		if err != nil {
			t.Errorf("Failed to marshal result data: %v", err)
		}

		_, err = fmt.Fprintf(w, "data: %s\n\n", resultData)
		if err != nil {
			t.Errorf("Failed to write result event: %v", err)
		}

		// Flush data immediately
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, err := client.MonitorCrawlRequest(ctx, "test-uuid", false)
	if err != nil {
		t.Fatalf("MonitorCrawlRequest() error = %v", err)
	}

	// Read the first event (state)
	event1 := <-events
	if event1.Type != "state" {
		t.Errorf("Expected first event type 'state', got %s", event1.Type)
	}

	// Read the second event (progress)
	event2 := <-events
	if event2.Type != "progress" {
		t.Errorf("Expected second event type 'progress', got %s", event2.Type)
	}

	progressData, ok := event2.Data.(map[string]interface{})
	if !ok {
		t.Errorf("Expected progress data to be map, got %T", event2.Data)
	} else {
		progress, ok := progressData["progress"].(float64)
		if !ok || progress != 50.0 {
			t.Errorf("Expected progress value 50.0, got %v", progressData["progress"])
		}
	}

	// Read the third event (result)
	event3 := <-events
	if event3.Type != "result" {
		t.Errorf("Expected third event type 'result', got %s", event3.Type)
	}

	resultData, ok := event3.Data.(map[string]interface{})
	if !ok {
		t.Errorf("Expected result data to be map, got %T", event3.Data)
	} else {
		content, ok := resultData["content"].(string)
		if !ok || content != "test content" {
			t.Errorf("Expected content 'test content', got %v", resultData["content"])
		}
	}
}

func TestClient_DownloadCrawlRequest_Object(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/core/crawl-requests/test-uuid/download/" {
			t.Errorf("Expected path /api/v1/core/crawl-requests/test-uuid/download/, got %s", r.URL.Path)
		}

		// Return object JSON
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"content":"test content","metadata":{"url":"https://example.com"}}`))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	result, err := client.DownloadCrawlRequest(ctx, "test-uuid")
	if err != nil {
		t.Fatalf("DownloadCrawlRequest() error = %v", err)
	}

	content, ok := result["content"].(string)
	if !ok || content != "test content" {
		t.Errorf("DownloadCrawlRequest() content = %v, want %v", content, "test content")
	}
}

func TestClient_DownloadCrawlRequest_Array(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/core/crawl-requests/test-uuid/download/" {
			t.Errorf("Expected path /api/v1/core/crawl-requests/test-uuid/download/, got %s", r.URL.Path)
		}

		// Return array JSON
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`[{"content":"item1"},{"content":"item2"}]`))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	result, err := client.DownloadCrawlRequest(ctx, "test-uuid")
	if err != nil {
		t.Fatalf("DownloadCrawlRequest() error = %v", err)
	}

	// Check if we have results key
	results, ok := result["results"].([]interface{})
	if !ok {
		t.Errorf("DownloadCrawlRequest() results not found or not an array")
		return
	}

	if len(results) != 2 {
		t.Errorf("DownloadCrawlRequest() results length = %v, want %v", len(results), 2)
	}

	// Check first item content
	item1, ok := results[0].(map[string]interface{})
	if !ok {
		t.Errorf("DownloadCrawlRequest() results[0] not a map")
		return
	}

	content1, ok := item1["content"].(string)
	if !ok || content1 != "item1" {
		t.Errorf("DownloadCrawlRequest() results[0].content = %v, want %v", content1, "item1")
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
				URL:     nil,
				Options: CrawlOptions{},
			},
			expectedError: "watercrawl: validation error: url: URL is required",
		},
		{
			name: "empty string URL",
			input: CreateCrawlRequestInput{
				URL:     "",
				Options: CrawlOptions{},
			},
			expectedError: "watercrawl: validation error: url: URL cannot be empty",
		},
		{
			name: "empty string array URL",
			input: CreateCrawlRequestInput{
				URL:     []string{},
				Options: CrawlOptions{},
			},
			expectedError: "watercrawl: validation error: url: URL list cannot be empty",
		},
		{
			name: "array with empty string URL",
			input: CreateCrawlRequestInput{
				URL:     []string{"https://example.com", ""},
				Options: CrawlOptions{},
			},
			expectedError: "watercrawl: validation error: url[1]: URL cannot be empty",
		},
		{
			name: "invalid URL type",
			input: CreateCrawlRequestInput{
				URL:     123,
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
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request parameters",
		}); err != nil {
			t.Errorf("Failed to encode error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	_, err := client.GetCrawlRequests(ctx, 1, 10)
	if err == nil {
		t.Error("Expected API error, got nil")
		return
	}

	var apiErr *APIError
	ok := errors.As(err, &apiErr)
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
