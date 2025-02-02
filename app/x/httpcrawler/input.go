package httpcrawler

import (
	"flag"
	"fmt"
	"strings"
)

type Options struct {
	Targets     []string
	OutputFile  string
	Concurrency int
}

// parseInput parses command-line arguments and validates input
func parseInput() (*Options, error) {
	var targets string
	var outputFile string
	var concurrency int

	flag.StringVar(&targets, "targets", "", "Comma-separated list of targets to probe")
	flag.StringVar(&outputFile, "output", "results.txt", "File to write results")
	flag.IntVar(&concurrency, "concurrency", 10, "Number of concurrent probes")
	flag.Parse()

	if targets == "" {
		return nil, fmt.Errorf("no targets provided")
	}

	options := &Options{
		Targets:     strings.Split(targets, ","),
		OutputFile:  outputFile,
		Concurrency: concurrency,
	}

	return options, nil
}
