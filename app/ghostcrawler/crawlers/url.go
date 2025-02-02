package ghostcrawler

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// URLCrawler is a crawler for probing and analyzing URLs.
type URLCrawler struct {
	name   string
	mutex  sync.Mutex
	output []Result
}

// NewURLCrawler initializes a new instance of URLCrawler.
func NewURLCrawler() *URLCrawler {
	return &URLCrawler{
		name:   "url",
		output: []Result{},
	}
}

// Name returns the name of the crawler.
func (c *URLCrawler) Name() string {
	return c.name
}

// Start begins the URL probing process.
func (c *URLCrawler) Start(ctx context.Context, input []string, output chan<- Result) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var wg sync.WaitGroup
	client := &http.Client{Timeout: 10 * time.Second}

	for _, url := range input {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			status, links, err := probeURL(client, url)
			if err != nil {
				output <- Result{CrawlerName: c.name, Error: fmt.Errorf("failed to probe URL %s: %w", url, err)}
				return
			}
			result := Result{
				CrawlerName: c.name,
				Data: map[string]interface{}{
					"url":    url,
					"status": status,
					"links":  links,
				},
			}
			c.output = append(c.output, result)
			output <- result
		}(url)
	}

	wg.Wait()
	return nil
}

// probeURL probes a URL for its HTTP status and extracts links from the response.
func probeURL(client *http.Client, url string) (int, []string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Extract links from the response body (mock implementation for now)
	links := extractLinksFromBody(resp.Body)
	return resp.StatusCode, links, nil
}

// extractLinksFromBody parses and extracts links from the HTTP response body.
func extractLinksFromBody(body interface{}) []string {
	// Mock implementation: replace with actual HTML parsing logic.
	return []string{
		"https://example.com/about",
		"https://example.com/contact",
		"https://example.com/products",
	}
}
