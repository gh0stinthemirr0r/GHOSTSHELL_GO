package proxi

import (
	"flag"
	"fmt"
)

// InputOptions represents the user-provided input for Proxi
type InputOptions struct {
	ListenAddress string
	Port          int
	Verbose       bool
	OutputFile    string
}

// parseInput parses command-line arguments and returns InputOptions
func parseInput() (*InputOptions, error) {
	var listenAddress string
	var port int
	var verbose bool
	var outputFile string

	flag.StringVar(&listenAddress, "listen", "127.0.0.1", "Address to listen on")
	flag.IntVar(&port, "port", 8080, "Port to listen on")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&outputFile, "output", "", "File to write captured data")
	flag.Parse()

	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}

	options := &InputOptions{
		ListenAddress: listenAddress,
		Port:          port,
		Verbose:       verbose,
		OutputFile:    outputFile,
	}

	return options, nil
}
