package ghostcrawler

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// DNSCrawler is a crawler for performing DNS lookups and resolving records.
type DNSCrawler struct {
	name   string
	mutex  sync.Mutex
	output []Result
}

// NewDNSCrawler initializes a new instance of DNSCrawler.
func NewDNSCrawler() *DNSCrawler {
	return &DNSCrawler{
		name:   "dns",
		output: []Result{},
	}
}

// Name returns the name of the crawler.
func (c *DNSCrawler) Name() string {
	return c.name
}

// Start begins the DNS lookup process.
func (c *DNSCrawler) Start(ctx context.Context, input []string, output chan<- Result) error {
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
			records, err := resolveDNS(domain)
			if err != nil {
				output <- Result{CrawlerName: c.name, Error: fmt.Errorf("failed to resolve DNS for %s: %w", domain, err)}
				return
			}
			result := Result{
				CrawlerName: c.name,
				Data: map[string]interface{}{
					"domain":  domain,
					"records": records,
				},
			}
			c.output = append(c.output, result)
			output <- result
		}(domain)
	}

	wg.Wait()
	return nil
}

// resolveDNS performs a DNS lookup for the given domain and returns its records.
func resolveDNS(domain string) (map[string][]string, error) {
	results := make(map[string][]string)

	// Perform A record lookup
	aRecords, err := net.LookupHost(domain)
	if err == nil {
		results["A"] = aRecords
	}

	// Perform MX record lookup
	mxRecords, err := net.LookupMX(domain)
	if err == nil {
		var mxList []string
		for _, mx := range mxRecords {
			mxList = append(mxList, fmt.Sprintf("%s %d", mx.Host, mx.Pref))
		}
		results["MX"] = mxList
	}

	// Perform NS record lookup
	nsRecords, err := net.LookupNS(domain)
	if err == nil {
		var nsList []string
		for _, ns := range nsRecords {
			nsList = append(nsList, ns.Host)
		}
		results["NS"] = nsList
	}

	// Perform TXT record lookup
	txtRecords, err := net.LookupTXT(domain)
	if err == nil {
		results["TXT"] = txtRecords
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no DNS records found for %s", domain)
	}

	return results, nil
}
