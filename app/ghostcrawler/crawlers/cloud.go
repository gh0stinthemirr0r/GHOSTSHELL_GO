package ghostcrawler

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CloudCrawler is a crawler for discovering cloud resource configurations.
type CloudCrawler struct {
	name   string
	mutex  sync.Mutex
	output []Result
}

// NewCloudCrawler initializes a new instance of CloudCrawler.
func NewCloudCrawler() *CloudCrawler {
	return &CloudCrawler{
		name:   "cloud",
		output: []Result{},
	}
}

// Name returns the name of the crawler.
func (c *CloudCrawler) Name() string {
	return c.name
}

// Start begins the cloud resource discovery process.
func (c *CloudCrawler) Start(ctx context.Context, input []string, output chan<- Result) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var wg sync.WaitGroup
	client := &http.Client{Timeout: 10 * time.Second}

	for _, endpoint := range input {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()
			info, err := probeCloudEndpoint(client, endpoint)
			if err != nil {
				output <- Result{CrawlerName: c.name, Error: fmt.Errorf("failed to probe cloud endpoint %s: %w", endpoint, err)}
				return
			}
			result := Result{
				CrawlerName: c.name,
				Data: map[string]string{
					"endpoint": endpoint,
					"info":     info,
				},
			}
			c.output = append(c.output, result)
			output <- result
		}(endpoint)
	}

	wg.Wait()
	return nil
}

// probeCloudEndpoint performs a mock probe of a cloud endpoint.
func probeCloudEndpoint(client *http.Client, endpoint string) (string, error) {
	// Simulate an HTTP GET request to the cloud endpoint
	resp, err := client.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Mock logic: determine cloud provider based on headers or status code
	switch resp.StatusCode {
	case 200:
		if resp.Header.Get("Server") == "AmazonS3" {
			return "AWS S3", nil
		} else if resp.Header.Get("X-Google-Metadata") != "" {
			return "Google Cloud", nil
		} else if resp.Header.Get("X-Azure-Resource") != "" {
			return "Azure", nil
		}
	case 403:
		return "Access Denied - Possible Cloud Resource", nil
	}

	return "Unknown Cloud Provider", nil
}
