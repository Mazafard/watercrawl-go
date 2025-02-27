package watercrawl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

	var result map[string]interface{}
	if err := c.processResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
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
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var event EventStreamMessage
				if err := decoder.Decode(&event); err != nil {
					if err != io.EOF {
						// Log error if needed
					}
					return
				}

				if download && event.Type == "result" {
					// Download the result data if requested
					if resultData, ok := event.Data.(map[string]interface{}); ok {
						downloadedData, err := c.DownloadCrawlRequest(ctx, id)
						if err == nil {
							resultData = downloadedData
							event.Data = resultData
						}
					}
				}

				eventChan <- &event
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

	if !sync {
		return map[string]interface{}{
			"uuid":   result.UUID,
			"status": result.Status,
		}, nil
	}

	events, err := c.MonitorCrawlRequest(ctx, result.UUID, download)
	if err != nil {
		return nil, err
	}

	for event := range events {
		if event.Type == "result" {
			if data, ok := event.Data.(map[string]interface{}); ok {
				return data, nil
			}
		}
	}

	return nil, fmt.Errorf("no result received from crawl request")
} 