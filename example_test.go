package watercrawl_test

import (
	"context"
	"fmt"
	"log"

	"github.com/watercrawl/watercrawl-go"
)

func Example() {
	// Initialize the client
	client := watercrawl.NewClient("your-api-key", "")

	// Create a context
	ctx := context.Background()

	// Example 1: Simple URL scraping
	result, err := client.ScrapeURL(ctx, "https://example.com", nil, nil, true, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Scraped content: %v\n", result)

	// Example 2: Create a crawl request with options
	input := watercrawl.CreateCrawlRequestInput{
		URL: "https://example.com",
		Options: watercrawl.CrawlOptions{
			SpiderOptions: map[string]interface{}{
				"allowed_domains": []string{"example.com"},
				"max_depth":      2,
			},
			PageOptions: map[string]interface{}{
				"wait_for": "#content",
				"timeout":  30,
			},
			PluginOptions: map[string]interface{}{
				"extract_links": true,
				"extract_text": true,
			},
		},
	}

	request, err := client.CreateCrawlRequest(ctx, input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created crawl request: %s\n", request.UUID)

	// Example 3: Monitor crawl progress
	events, err := client.MonitorCrawlRequest(ctx, request.UUID, true)
	if err != nil {
		log.Fatal(err)
	}

	for event := range events {
		switch event.Type {
		case "progress":
			fmt.Printf("Progress update: %v\n", event.Data)
		case "result":
			fmt.Printf("Got result: %v\n", event.Data)
		}
	}

	// Example 4: List crawl requests
	list, err := client.GetCrawlRequests(ctx, 1, 10)
	if err != nil {
		log.Fatal(err)
	}

	for _, req := range list.Results {
		fmt.Printf("Request %s: %s (Progress: %.2f%%)\n", req.UUID, req.Status, req.Progress)
	}

	// Example 5: Stop a crawl request
	err = client.StopCrawlRequest(ctx, request.UUID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Stopped crawl request: %s\n", request.UUID)
}

func ExampleClient_ScrapeURL() {
	client := watercrawl.NewClient("your-api-key", "")
	ctx := context.Background()

	// Define page and plugin options
	pageOptions := map[string]interface{}{
		"wait_for": "#main-content",
		"timeout":  30,
	}

	pluginOptions := map[string]interface{}{
		"extract_links": true,
		"extract_text":  true,
	}

	// Scrape URL synchronously with automatic result download
	result, err := client.ScrapeURL(ctx, "https://example.com", pageOptions, pluginOptions, true, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Scraped content: %v\n", result)
}

func ExampleClient_MonitorCrawlRequest() {
	client := watercrawl.NewClient("your-api-key", "")
	ctx := context.Background()

	// Create a crawl request first
	input := watercrawl.CreateCrawlRequestInput{
		URL: "https://example.com",
		Options: watercrawl.CrawlOptions{
			SpiderOptions: map[string]interface{}{
				"allowed_domains": []string{"example.com"},
			},
		},
	}

	request, err := client.CreateCrawlRequest(ctx, input)
	if err != nil {
		log.Fatal(err)
	}

	// Monitor the crawl progress
	events, err := client.MonitorCrawlRequest(ctx, request.UUID, true)
	if err != nil {
		log.Fatal(err)
	}

	for event := range events {
		switch event.Type {
		case "progress":
			fmt.Printf("Progress: %v\n", event.Data)
		case "result":
			fmt.Printf("Result: %v\n", event.Data)
		case "error":
			fmt.Printf("Error: %v\n", event.Data)
		}
	}
} 