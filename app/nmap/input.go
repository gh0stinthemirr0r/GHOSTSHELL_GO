package nmap

import (
	"flag"
	"fmt"
	"strings"
)

// Options represents the user-provided input for the Nmap scanner
type Options struct {
	Targets        []string
	Ports          string
	OutputFile     string
	TimingTemplate int
	Verbose        bool
}

// parseInput parses command-line arguments and returns Options
func parseInput() (*Options, error) {
	var targets string
	var ports string
	var outputFile string
	var timingTemplate int
	var verbose bool

	flag.StringVar(&targets, "targets", "", "Comma-separated list of targets to scan")
	flag.StringVar(&ports, "ports", "1-1000", "Ports to scan (e.g., 80,443 or 1-1000)")
	flag.StringVar(&outputFile, "output", "results.xml", "File to write scan results")
	flag.IntVar(&timingTemplate, "timing", 3, "Timing template for the scan (1-5)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.Parse()

	if targets == "" {
		return nil, fmt.Errorf("no targets provided")
	}

	options := &Options{
		Targets:        strings.Split(targets, ","),
		Ports:          ports,
		OutputFile:     outputFile,
		TimingTemplate: timingTemplate,
		Verbose:        verbose,
	}

	return options, nil
}
