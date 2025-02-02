package main

import (
	"fmt"
)

type Config struct {
	StartPort int
	EndPort   int
	Protocol  string
}

// DefaultConfig provides a default configuration for open port scanning
var DefaultConfig = Config{
	StartPort: 1024,
	EndPort:   65535,
	Protocol:  "tcp",
}

// Validate ensures the configuration is valid
func (c *Config) Validate() error {
	if c.StartPort < 1 || c.StartPort > 65535 {
		return fmt.Errorf("invalid start port: %d", c.StartPort)
	}
	if c.EndPort < 1 || c.EndPort > 65535 || c.EndPort < c.StartPort {
		return fmt.Errorf("invalid end port: %d", c.EndPort)
	}
	if c.Protocol != "tcp" && c.Protocol != "udp" {
		return fmt.Errorf("invalid protocol: %s", c.Protocol)
	}
	return nil
}
