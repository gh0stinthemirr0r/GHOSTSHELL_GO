package subcrawler

import (
	"flag"
	"fmt"
)

// Options represents the user-provided options for Subcrawler
// Includes domain, output file, and other settings
type Options struct {
	Domain     string
	OutputFile string
	Verbose    bool
}

// ParseOptions parses command-line arguments and returns an Options instance
func ParseOptions() (*Options, error) {
	var domain string
	var outputFile string
	var verbose bool

	flag.StringVar(&domain, "domain", "", "Domain to enumerate subdomains for")
	flag.StringVar(&outputFile, "output", "", "File to save results")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	return &Options{
		Domain:     domain,
		OutputFile: outputFile,
		Verbose:    verbose,
	}, nil
}
