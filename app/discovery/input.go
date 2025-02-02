package discovery

import (
	"flag"
	"fmt"
	"strings"
)

type Options struct {
	Query         []string
	Engines       []string
	Output        string
	RetryMax      int
	Timeout       int
	RateLimit     int
	RateLimitUnit int
}

// parseInput parses command-line arguments into options
func parseInput() (*Options, error) {
	var queries string
	var engines string
	var output string
	var retryMax int
	var timeout int
	var rateLimit int
	var rateLimitUnit int

	// Define flags
	flag.StringVar(&queries, "queries", "", "Comma-separated list of queries")
	flag.StringVar(&engines, "engines", "shodan", "Comma-separated list of search engines (default: shodan)")
	flag.StringVar(&output, "output", "results.txt", "Output file for results")
	flag.IntVar(&retryMax, "retry", 3, "Maximum number of retries for requests")
	flag.IntVar(&timeout, "timeout", 30, "Request timeout in seconds")
	flag.IntVar(&rateLimit, "rate-limit", 30, "Rate limit for requests per second")
	flag.IntVar(&rateLimitUnit, "rate-limit-unit", 60, "Rate limit unit in seconds")

	flag.Parse()

	// Validate required arguments
	if queries == "" {
		return nil, fmt.Errorf("queries are required")
	}

	return &Options{
		Query:         strings.Split(queries, ","),
		Engines:       strings.Split(engines, ","),
		Output:        output,
		RetryMax:      retryMax,
		Timeout:       timeout,
		RateLimit:     rateLimit,
		RateLimitUnit: rateLimitUnit,
	}, nil
}
