package ghostcrawler

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// ASNCrawler is a crawler for performing ASN lookups.
type ASNCrawler struct {
	name   string
	mutex  sync.Mutex
	output []Result
}

// NewASNCrawler initializes a new instance of ASNCrawler.
func NewASNCrawler() *ASNCrawler {
	return &ASNCrawler{
		name:   "asn",
		output: []Result{},
	}
}

// Name returns the name of the crawler.
func (c *ASNCrawler) Name() string {
	return c.name
}

// Start begins the ASN lookup process.
func (c *ASNCrawler) Start(ctx context.Context, input []string, output chan<- Result) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var wg sync.WaitGroup

	for _, ip := range input {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			asn, err := lookupASN(ip)
			if err != nil {
				output <- Result{CrawlerName: c.name, Error: fmt.Errorf("failed to lookup ASN for %s: %w", ip, err)}
				return
			}
			result := Result{
				CrawlerName: c.name,
				Data: map[string]string{
					"ip":  ip,
					"asn": asn,
				},
			}
			c.output = append(c.output, result)
			output <- result
		}(ip)
	}

	wg.Wait()
	return nil
}

// lookupASN performs a mock ASN lookup for an IP address.
func lookupASN(ip string) (string, error) {
	// In a real implementation, you would query a database or API (e.g., Team Cymru, MaxMind).
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return "", fmt.Errorf("invalid IP address")
	}

	// Simulate lookup result (replace with real logic)
	return fmt.Sprintf("AS%d", len(netIP)), nil
}
