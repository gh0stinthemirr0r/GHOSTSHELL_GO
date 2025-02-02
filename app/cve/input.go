package cve

import (
	"flag"
	"fmt"
	"strings"
)

type Options struct {
	Domain    string
	Output    string
	Providers []string
	Verbose   bool
}

// ParseArgs parses command-line arguments for CloudCrawler
func ParseArgs() (*Options, error) {
	var domain, output, providers string
	var verbose bool

	// Define command-line flags
	flag.StringVar(&domain, "domain", "", "Domain to crawl (required)")
	flag.StringVar(&output, "output", "results.txt", "Output file for results")
	flag.StringVar(&providers, "providers", "", "Comma-separated list of providers (e.g., aws,azure)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	// Validate required arguments
	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	// Parse providers into a slice
	providerList := parseProviders(providers)

	return &Options{
		Domain:    domain,
		Output:    output,
		Providers: providerList,
		Verbose:   verbose,
	}, nil
}

// parseProviders converts a comma-separated string into a slice of providers
func parseProviders(input string) []string {
	if input == "" {
		return []string{}
	}
	return strings.Split(input, ",")
}
