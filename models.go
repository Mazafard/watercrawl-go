package watercrawl

// CrawlRequest represents a crawl request
type CrawlRequest struct {
	UUID      string         `json:"uuid"`
	URL       interface{}    `json:"url"` // Can be string or []string
	Status    string         `json:"status"`
	Progress  float64        `json:"progress"`
	Options   CrawlOptions   `json:"options"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

// CrawlOptions represents the options for a crawl request
type CrawlOptions struct {
	SpiderOptions  map[string]interface{} `json:"spider_options"`
	PageOptions    map[string]interface{} `json:"page_options"`
	PluginOptions  map[string]interface{} `json:"plugin_options"`
}

// CrawlRequestList represents a paginated list of crawl requests
type CrawlRequestList struct {
	Count    int            `json:"count"`
	Next     *string        `json:"next"`
	Previous *string        `json:"previous"`
	Results  []CrawlRequest `json:"results"`
}

// CrawlResult represents a crawl result
type CrawlResult struct {
	UUID       string                 `json:"uuid"`
	URL        string                 `json:"url"`
	Status     string                 `json:"status"`
	Data       map[string]interface{} `json:"data"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// CrawlResultList represents a paginated list of crawl results
type CrawlResultList struct {
	Count    int           `json:"count"`
	Next     *string       `json:"next"`
	Previous *string       `json:"previous"`
	Results  []CrawlResult `json:"results"`
}

// EventStreamMessage represents a message from the event stream
type EventStreamMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// CreateCrawlRequestInput represents the input for creating a crawl request
type CreateCrawlRequestInput struct {
	URL     interface{} `json:"url"` // Can be string or []string
	Options CrawlOptions `json:"options"`
} 