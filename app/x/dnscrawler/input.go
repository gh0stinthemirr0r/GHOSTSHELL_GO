package dnscrawler

import (
	"flag"
	"fmt"
	"strings"
)

type Options struct {
	Domains     []string
	Resolvers   []string
	Concurrency int
	OutputFile  string
}

// parseInput parses command-line arguments and validates input
func parseInput() (*Options, error) {
	var domainList, resolvers, outputFile string
	var concurrency int

	flag.StringVar(&domainList, "domains", "", "Comma-separated list of domains to query")
	flag.StringVar(&resolvers, "resolvers", "", "Comma-separated list of DNS resolvers")
	flag.StringVar(&outputFile, "output", "results.txt", "File to write results")
	flag.IntVar(&concurrency, "concurrency", 10, "Number of concurrent queries")
	flag.Parse()

	if domainList == "" {
		return nil, fmt.Errorf("no domains provided")
	}

	options := &Options{
		Domains:     strings.Split(domainList, ","),
		Resolvers:   strings.Split(resolvers, ","),
		Concurrency: concurrency,
		OutputFile:  outputFile,
	}

	return options, nil
}
