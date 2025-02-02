package ghostcrawler

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
)

// CDNCrawler is a crawler for detecting CDN configurations.
type CDNCrawler struct {
	name   string
	mutex  sync.Mutex
	output []Result
}

// NewCDNCrawler initializes a new instance of CDNCrawler.
func NewCDNCrawler() *CDNCrawler {
	return &CDNCrawler{
		name:   "cdn",
		output: []Result{},
	}
}

// Name returns the name of the crawler.
func (c *CDNCrawler) Name() string {
	return c.name
}

// Start begins the CDN detection process.
func (c *CDNCrawler) Start(ctx context.Context, input []string, output chan<- Result) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var wg sync.WaitGroup

	for _, domain := range input {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			cdn, err := detectCDN(domain)
			if err != nil {
				output <- Result{CrawlerName: c.name, Error: fmt.Errorf("failed to detect CDN for %s: %w", domain, err)}
				return
			}
			result := Result{
				CrawlerName: c.name,
				Data: map[string]string{
					"domain": domain,
					"cdn":    cdn,
				},
			}
			c.output = append(c.output, result)
			output <- result
		}(domain)
	}

	wg.Wait()
	return nil
}

// detectCDN performs a mock detection of a CDN for a domain.
func detectCDN(domain string) (string, error) {
	// Simulate a DNS lookup for a domain
	ns, err := net.LookupNS(domain)
	if err != nil {
		return "", fmt.Errorf("DNS lookup failed: %w", err)
	}

	// Mock logic: check for common CDN-related names in the NS records
	for _, nsRecord := range ns {
		if strings.Contains(nsRecord.Host, "cloudflare") {
			return "Cloudflare", nil
		} else if strings.Contains(nsRecord.Host, "akamai") {
			return "Akamai", nil
		} else if strings.Contains(nsRecord.Host, "aws") {
			return "AWS CloudFront", nil
		}
	}

	return "Unknown CDN", nil
}
