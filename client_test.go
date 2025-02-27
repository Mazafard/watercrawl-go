package watercrawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		baseURL  string
		wantBase string
	}{
		{
			name:     "with default base URL",
			apiKey:   "test-key",
			baseURL:  "",
			wantBase: "https://app.watercrawl.dev/",
		},
		{
			name:     "with custom base URL",
			apiKey:   "test-key",
			baseURL:  "https://custom.example.com/",
			wantBase: "https://custom.example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey, tt.baseURL)
			if client.apiKey != tt.apiKey {
				t.Errorf("NewClient().apiKey = %v, want %v", client.apiKey, tt.apiKey)
			}
			if client.baseURL != tt.wantBase {
				t.Errorf("NewClient().baseURL = %v, want %v", client.baseURL, tt.wantBase)
			}
			if client.httpClient == nil {
				t.Error("NewClient().httpClient is nil")
			}
		})
	}
}

func TestClient_doRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "test-key" {
			t.Errorf("Expected X-API-Key header to be 'test-key', got %v", apiKey)
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Expected Content-Type header to be 'application/json', got %v", contentType)
		}
		if userAgent := r.Header.Get("User-Agent"); userAgent != "WaterCrawl-Go-SDK" {
			t.Errorf("Expected User-Agent header to be 'WaterCrawl-Go-SDK', got %v", userAgent)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/")
	ctx := context.Background()

	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("doRequest() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}
}

func TestClient_processResponse(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		contentType string
		body        string
		wantErr     bool
	}{
		{
			name:        "successful JSON response",
			statusCode:  http.StatusOK,
			contentType: "application/json",
			body:        `{"status":"ok"}`,
			wantErr:     false,
		},
		{
			name:        "no content response",
			statusCode:  http.StatusNoContent,
			contentType: "",
			body:        "",
			wantErr:     false,
		},
		{
			name:        "error response",
			statusCode:  http.StatusBadRequest,
			contentType: "application/json",
			body:        `{"error":"bad request"}`,
			wantErr:     true,
		},
		{
			name:        "invalid JSON response",
			statusCode:  http.StatusOK,
			contentType: "application/json",
			body:        `invalid json`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient("test-key", server.URL+"/")
			resp, err := http.Get(server.URL)
			if err != nil {
				t.Fatalf("Failed to make test request: %v", err)
			}

			var result map[string]interface{}
			err = client.processResponse(resp, &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("processResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
} 