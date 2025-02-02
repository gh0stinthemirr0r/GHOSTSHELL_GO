// Package input handles user input and validation for the application.
package input

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// InputHandler manages user input and validation.
type InputHandler struct{}

// NewInputHandler creates a new instance of InputHandler.
func NewInputHandler() *InputHandler {
	return &InputHandler{}
}

// GetDestination prompts the user for a destination and validates the input.
func (ih *InputHandler) GetDestination() ([]string, error) {
	fmt.Println("Enter destination IP(s) or network(s) (comma-separated):")
	var input string
	fmt.Scanln(&input)

	// Split the input into multiple destinations.
	destinations := strings.Split(input, ",")
	var validDestinations []string

	for _, dest := range destinations {
		dest = strings.TrimSpace(dest)
		if isValidIP(dest) || isValidCIDR(dest) {
			validDestinations = append(validDestinations, dest)
		} else {
			return nil, fmt.Errorf("invalid destination: %s", dest)
		}
	}

	if len(validDestinations) == 0 {
		return nil, errors.New("no valid destinations provided")
	}

	return validDestinations, nil
}

// isValidIP checks if the provided input is a valid IP address.
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// isValidCIDR checks if the provided input is a valid CIDR notation.
func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// PromptUser prompts the user for input and returns their response.
func (ih *InputHandler) PromptUser(prompt string) (string, error) {
	fmt.Println(prompt)
	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(response)

	if response == "" {
		return "", errors.New("input cannot be empty")
	}

	return response, nil
}
