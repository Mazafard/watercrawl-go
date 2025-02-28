package watercrawl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GetCrawlRequests retrieves a paginated list of crawl requests
func (c *Client) GetCrawlRequests(ctx context.Context, page, pageSize int) (*CrawlRequestList, error) {
	queryParams := url.Values{}
	queryParams.Set("page", strconv.Itoa(page))
	queryParams.Set("page_size", strconv.Itoa(pageSize))

	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/core/crawl-requests/", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var result CrawlRequestList
	if err := c.processResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetCrawlRequest retrieves a specific crawl request by ID
func (c *Client) GetCrawlRequest(ctx context.Context, id string) (*CrawlRequest, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/core/crawl-requests/%s/", id), nil, nil)
	if err != nil {
		return nil, err
	}

	var result CrawlRequest
	if err := c.processResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateCrawlRequest creates a new crawl request
func (c *Client) CreateCrawlRequest(ctx context.Context, input CreateCrawlRequestInput) (*CrawlRequest, error) {
	// Validate input
	if input.URL == nil {
		return nil, &ValidationError{
			Field:   "url",
			Message: "URL is required",
		}
	}

	switch v := input.URL.(type) {
	case string:
		if v == "" {
			return nil, &ValidationError{
				Field:   "url",
				Message: "URL cannot be empty",
			}
		}
	case []string:
		if len(v) == 0 {
			return nil, &ValidationError{
				Field:   "url",
				Message: "URL list cannot be empty",
			}
		}
		for i, u := range v {
			if u == "" {
				return nil, &ValidationError{
					Field:   fmt.Sprintf("url[%d]", i),
					Message: "URL cannot be empty",
				}
			}
		}
	default:
		return nil, &ValidationError{
			Field:   "url",
			Message: "URL must be a string or array of strings",
		}
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/core/crawl-requests/", nil, input)
	if err != nil {
		return nil, err
	}

	var result CrawlRequest
	if err := c.processResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// StopCrawlRequest stops a specific crawl request
func (c *Client) StopCrawlRequest(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/core/crawl-requests/%s/", id), nil, nil)
	if err != nil {
		return err
	}

	return c.processResponse(resp, nil)
}

// DownloadCrawlRequest downloads the results of a crawl request
func (c *Client) DownloadCrawlRequest(ctx context.Context, id string) (map[string]interface{}, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/core/crawl-requests/%s/download/", id), nil, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// First try to unmarshal as object
	var resultObj map[string]interface{}
	if err := json.Unmarshal(body, &resultObj); err != nil {
		// If that fails, try to unmarshal as array
		var resultArray []interface{}
		if err := json.Unmarshal(body, &resultArray); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Convert array to map with "results" key
		resultObj = map[string]interface{}{
			"results": resultArray,
		}
	}

	return resultObj, nil
}

// MonitorCrawlRequest monitors the status of a crawl request and returns a channel of events
func (c *Client) MonitorCrawlRequest(ctx context.Context, id string, download bool) (<-chan *EventStreamMessage, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/core/crawl-requests/%s/status/", id), nil, nil)
	if err != nil {
		return nil, err
	}

	eventChan := make(chan *EventStreamMessage)

	go func() {
		defer close(eventChan)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("Error closing response body: %v\n", err)
			}
		}()

		// Create a reader for the response body
		reader := bufio.NewReader(resp.Body)

		for {
			select {
			case <-ctx.Done():
				fmt.Println("Context done, stopping monitoring")
				return
			default:
				// Read line by line
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						fmt.Printf("Error reading line: %v\n", err)
					} else {
						fmt.Println("End of stream (EOF)")
					}
					return
				}

				// Trim whitespace
				line = strings.TrimSpace(line)

				// Skip empty lines
				if line == "" {
					continue
				}

				fmt.Printf("Received line: %s\n", line)

				// Check if it's an SSE data line
				if strings.HasPrefix(line, "data:") {
					// Extract the JSON payload
					jsonData := strings.TrimPrefix(line, "data:")
					jsonData = strings.TrimSpace(jsonData)

					// Parse the JSON
					var event EventStreamMessage
					if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
						fmt.Printf("Error parsing JSON from SSE: %v\n", err)
						continue
					}

					// Process the event
					if download && event.Type == "result" {
						// Download the result data if requested
						if _, ok := event.Data.(map[string]interface{}); ok {
							// Create a new timeout context for download operation
							downloadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							downloadedData, err := c.DownloadCrawlRequest(downloadCtx, id)
							cancel()

							if err == nil {
								// Replace the entire event data with downloaded data
								event.Data = downloadedData
								fmt.Println("Successfully downloaded result data")
							} else {
								fmt.Printf("Error downloading result data: %v\n", err)
							}
						}
					}

					// Try to send the event, respecting context cancellation
					select {
					case eventChan <- &event:
						// Event sent successfully
					case <-ctx.Done():
						fmt.Println("Context done while sending event")
						return
					}
				} else {
					// Handle other types of SSE lines if needed (like "id:" or "event:")
					fmt.Printf("Non-data SSE line: %s\n", line)
				}
			}
		}
	}()

	return eventChan, nil
}

// GetCrawlRequestResults retrieves the results of a crawl request
func (c *Client) GetCrawlRequestResults(ctx context.Context, id string, page, pageSize int) (*CrawlResultList, error) {
	queryParams := url.Values{}
	queryParams.Set("page", strconv.Itoa(page))
	queryParams.Set("page_size", strconv.Itoa(pageSize))

	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/core/crawl-requests/%s/results/", id), queryParams, nil)
	if err != nil {
		return nil, err
	}

	var result CrawlResultList
	if err := c.processResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ScrapeURL performs a single URL scrape
func (c *Client) ScrapeURL(ctx context.Context, url string, pageOptions, pluginOptions map[string]interface{}, sync, download bool) (map[string]interface{}, error) {
	input := CreateCrawlRequestInput{
		URL: url,
		Options: CrawlOptions{
			SpiderOptions: map[string]interface{}{
				"allowed_domains": []string{"*"},
			},
			PageOptions:   pageOptions,
			PluginOptions: pluginOptions,
		},
	}

	result, err := c.CreateCrawlRequest(ctx, input)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Crawl request created with UUID: %s, Status: %s\n", result.UUID, result.Status)

	if !sync {
		return map[string]interface{}{
			"uuid":   result.UUID,
			"status": result.Status,
		}, nil
	}

	fmt.Println("Monitoring crawl request...")
	events, err := c.MonitorCrawlRequest(ctx, result.UUID, download)
	if err != nil {
		return nil, err
	}

	fmt.Println("Waiting for events...")
	eventCount := 0
	var lastProgress float64
	var lastError interface{}
	var lastStateData map[string]interface{}

	for event := range events {
		eventCount++
		fmt.Printf("Received event #%d of type: %s\n", eventCount, event.Type)

		switch event.Type {
		case "result":
			fmt.Println("Found result event!")
			if data, ok := event.Data.(map[string]interface{}); ok {
				return data, nil
			} else {
				fmt.Printf("Warning: result event has unexpected data type: %T\n", event.Data)
			}
		case "error":
			fmt.Printf("Error event received: %v\n", event.Data)
			lastError = event.Data
		case "progress":
			if progressData, ok := event.Data.(map[string]interface{}); ok {
				if progress, ok := progressData["progress"].(float64); ok {
					lastProgress = progress
					fmt.Printf("Progress: %.2f%%\n", progress)
				}
			}
		case "state":
			// Save state data in case we don't get a result event
			if stateData, ok := event.Data.(map[string]interface{}); ok {
				lastStateData = stateData

				// Check if status is "completed" or "failed"
				if status, ok := stateData["status"].(string); ok {
					if status == "completed" {
						fmt.Println("Crawl completed according to state event")
						if download {
							// Try to download the results
							downloadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							downloadData, err := c.DownloadCrawlRequest(downloadCtx, result.UUID)
							cancel()

							if err == nil {
								return downloadData, nil
							} else {
								fmt.Printf("Error downloading result data: %v\n", err)
							}
						}
						// If download failed or wasn't requested, return the state data
						return stateData, nil
					} else if status == "failed" {
						return nil, fmt.Errorf("crawl failed with status: %s", status)
					}
				}
			}
		case "completed":
			fmt.Println("Crawl completed event received")
			// If we receive a completed event but haven't received a result yet, try to download
			if download {
				fmt.Println("Attempting to download final results...")
				downloadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				downloadedData, err := c.DownloadCrawlRequest(downloadCtx, result.UUID)
				cancel()

				if err == nil && len(downloadedData) > 0 {
					fmt.Println("Successfully downloaded final results")
					return downloadedData, nil
				} else if err != nil {
					fmt.Printf("Error downloading final results: %v\n", err)
				} else {
					fmt.Println("Downloaded results were empty")
				}
			}
		}
	}

	// If we have state data but no result, return the state data
	if lastStateData != nil {
		return lastStateData, nil
	}

	// If we get here, we didn't receive a valid result
	if eventCount == 0 {
		return nil, fmt.Errorf("no events received from crawl request (timeout or connection error)")
	} else if lastError != nil {
		return nil, fmt.Errorf("crawl request failed with error: %v", lastError)
	} else {
		return nil, fmt.Errorf("received %d events (last progress: %.2f%%) but no valid result event",
			eventCount, lastProgress)
	}
}
