package openport

import (
	"flag"
	"fmt"
)

// parseInput parses command-line arguments and returns a Config instance
func parseInput() (*Config, error) {
	var startPort int
	var endPort int
	var protocol string

	flag.IntVar(&startPort, "start", 1024, "Starting port number")
	flag.IntVar(&endPort, "end", 65535, "Ending port number")
	flag.StringVar(&protocol, "protocol", "tcp", "Protocol to use (tcp or udp)")
	flag.Parse()

	config := &Config{
		StartPort: startPort,
		EndPort:   endPort,
		Protocol:  protocol,
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	return config, nil
}
