package asnscanner

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Options holds the configuration for the ASN scanner
// extracted from command-line arguments.
type Options struct {
	Targets     []string // List of IP addresses, domains, or ASNs to scan
	OutputFile  string   // Path to the output file
	Format      string   // Output format (json, csv)
	Verbose     bool     // Enable verbose logging
	Concurrency int      // Number of concurrent scanning threads
	ConfigFile  string   // Path to the configuration file
}

// ParseOptions parses command-line arguments and returns an Options struct.
func ParseOptions() (*Options, error) {
	options := &Options{}

	// Define command-line flags
	targets := flag.String("targets", "", "Comma-separated list of targets (IPs, domains, or ASNs)")
	flag.StringVar(&options.OutputFile, "output", "output.json", "Path to the output file")
	flag.StringVar(&options.Format, "format", "json", "Output format: json or csv")
	flag.BoolVar(&options.Verbose, "verbose", false, "Enable verbose logging")
	flag.IntVar(&options.Concurrency, "concurrency", 4, "Number of concurrent scanning threads")
	flag.StringVar(&options.ConfigFile, "config", "config.yaml", "Path to the configuration file")

	// Parse flags
	flag.Parse()

	// Handle targets
	if *targets == "" {
		return nil, fmt.Errorf("no targets specified; use the --targets flag")
	}
	options.Targets = splitTargets(*targets)

	return options, nil
}

// splitTargets splits a comma-separated list of targets into a slice.
func splitTargets(input string) []string {
	var targets []string
	for _, target := range strings.Split(input, ",") {
		trimmed := strings.TrimSpace(target)
		if trimmed != "" {
			targets = append(targets, trimmed)
		}
	}
	return targets
}

// PrintUsage prints the usage message for the command-line tool.
func PrintUsage() {
	fmt.Fprintf(os.Stderr, `Usage of %s:

ASN Scanner CLI

Flags:
  --targets	Comma-separated list of targets (IPs, domains, or ASNs) (required)
  --output	Path to the output file (default "output.json")
  --format	Output format: json or csv (default "json")
  --verbose	Enable verbose logging (default false)
  --concurrency	Number of concurrent scanning threads (default 4)
  --config	Path to the configuration file (default "config.yaml")

Example:
  asnscanner --targets 8.8.8.8,example.com,AS12345 --output results.json --format json --verbose
`, os.Args[0])
}
