// Package cdnscanner handles the input processing for CDN scanning tasks.
package cdncrawler

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

// InputProcessor handles reading and validating input data for scanning.
type InputProcessor struct{}

// NewInputProcessor initializes and returns a new InputProcessor.
func NewInputProcessor() *InputProcessor {
	return &InputProcessor{}
}

// ReadInput reads input targets (IP/hostname) from a file or command-line arguments.
func (ip *InputProcessor) ReadInput(filePath string, args []string) ([]string, error) {
	var targets []string

	// If a file is provided, read input from the file.
	if filePath != "" {
		fileTargets, err := ip.readFromFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read input from file: %w", err)
		}
		targets = append(targets, fileTargets...)
	}

	// Append command-line arguments as additional input targets.
	if len(args) > 0 {
		for _, arg := range args {
			if strings.TrimSpace(arg) != "" {
				targets = append(targets, strings.TrimSpace(arg))
			}
		}
	}

	// Validate and sanitize the collected input targets.
	validTargets, err := ip.validateTargets(targets)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return validTargets, nil
}

// readFromFile reads input targets line by line from the provided file path.
func (ip *InputProcessor) readFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var targets []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			targets = append(targets, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return targets, nil
}

// validateTargets checks the validity of input targets (e.g., valid IPs or hostnames).
func (ip *InputProcessor) validateTargets(targets []string) ([]string, error) {
	var validTargets []string
	for _, target := range targets {
		if isValidTarget(target) {
			validTargets = append(validTargets, target)
		} else {
			fmt.Printf("Invalid target skipped: %s\n", target)
		}
	}

	if len(validTargets) == 0 {
		return nil, errors.New("no valid targets provided")
	}

	return validTargets, nil
}

// isValidTarget checks whether the input is a valid IP address or hostname.
func isValidTarget(target string) bool {
	// Check if the target is a valid IP address.
	if net.ParseIP(target) != nil {
		return true
	}

	// Check if the target is a valid hostname.
	if len(target) > 0 && len(target) <= 255 && strings.Contains(target, ".") && !strings.HasPrefix(target, "-") {
		return true
	}

	return false
}
