package webcrawler

import (
	"flag"
	"fmt"
	"strings"
)

type Options struct {
	ConfigFile   string
	Targets      []string
	Concurrency  int
	OutputFormat string
}

// parseInput parses command-line arguments and returns the options
func parseInput() (*Options, error) {
	var options Options
	var targets string

	flag.StringVar(&options.ConfigFile, "config", "", "Path to the configuration file")
	flag.StringVar(&targets, "targets", "", "Comma-separated list of targets to crawl")
	flag.IntVar(&options.Concurrency, "concurrency", 5, "Number of concurrent crawls")
	flag.StringVar(&options.OutputFormat, "output-format", "json", "Output format (e.g., json, text)")
	flag.Parse()

	if options.ConfigFile == "" && targets == "" {
		return nil, fmt.Errorf("either a config file or targets must be provided")
	}

	if targets != "" {
		options.Targets = strings.Split(targets, ",")
	}

	return &options, nil
}
