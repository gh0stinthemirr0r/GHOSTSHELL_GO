package burp

import (
	"flag"
	"fmt"
)

// parseInput parses command-line arguments and returns the options
func parseInput() (*Options, error) {
	var options Options

	flag.StringVar(&options.ConfigFile, "config", "", "Path to the configuration file")
	flag.IntVar(&options.InterceptTimeout, "timeout", 600, "Timeout for BIID interception in seconds")
	flag.Parse()

	if options.ConfigFile == "" {
		return nil, fmt.Errorf("config file path is required")
	}

	return &options, nil
}
