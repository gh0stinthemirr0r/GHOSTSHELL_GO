// parser.go

// Package commands provides functionalities to execute and parse shell commands securely.
package parser

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type Command struct {
	Name      string
	Arguments []string
	Flags     map[string]string
	Options   map[string]string
}

type CommandParser struct {
	logger  *zap.Logger
	mu      sync.RWMutex
	aliases map[string]string // New field for command aliases
}

func NewCommandParser(logger *zap.Logger) *CommandParser {
	return &CommandParser{
		logger: logger,
		aliases: map[string]string{
			"test": "oqs-test", // Example alias
			"ai":   "run-ai",   // Example alias
		},
	}
}

func (cp *CommandParser) ParseCommand(input string) (*Command, error) {
	cp.logger.Sugar().Infof("Parsing command input: %s", input)

	if strings.TrimSpace(input) == "" {
		return nil, errors.New("command input is empty")
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil, errors.New("no command detected")
	}

	// Check for command alias
	if alias, exists := cp.aliases[parts[0]]; exists {
		parts[0] = alias
	}

	command := &Command{
		Name:      parts[0],
		Arguments: []string{},
		Flags:     make(map[string]string),
		Options:   make(map[string]string),
	}

	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.HasPrefix(part, "--") {
			optionParts := strings.SplitN(part[2:], "=", 2)
			key := sanitizeInput(optionParts[0])
			if key == "" {
				return nil, fmt.Errorf("invalid option key: %s", optionParts[0])
			}
			if len(optionParts) == 2 {
				command.Options[key] = sanitizeInput(optionParts[1])
			} else {
				command.Options[key] = ""
			}
		} else if strings.HasPrefix(part, "-") {
			flags := strings.TrimPrefix(part, "-")
			for _, char := range flags {
				flag := string(char)
				if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
					command.Flags[flag] = sanitizeInput(parts[i+1])
					i++ // Skip the next part as it's the value for the flag
				} else {
					command.Flags[flag] = ""
				}
			}
		} else {
			command.Arguments = append(command.Arguments, sanitizeInput(part))
		}
	}

	return command, nil
}

func sanitizeInput(input string) string {
	// Improved sanitization logic
	cleanInput := strings.ReplaceAll(input, ";", "")
	cleanInput = strings.ReplaceAll(cleanInput, "&", "")
	cleanInput = strings.ReplaceAll(cleanInput, "|", "")
	cleanInput = strings.ReplaceAll(cleanInput, "`", "")
	cleanInput = strings.ReplaceAll(cleanInput, "$", "")
	cleanInput = strings.ReplaceAll(cleanInput, ">", "")
	cleanInput = strings.ReplaceAll(cleanInput, "<", "")
	cleanInput = strings.ReplaceAll(cleanInput, "\"", "")
	cleanInput = strings.ReplaceAll(cleanInput, "'", "")
	return cleanInput
}

func (cp *CommandParser) PrintCommandDetails(cmd *Command) {
	fmt.Println("Parsed Command:")
	fmt.Printf("  Name: %s\n", cmd.Name)
	if len(cmd.Arguments) > 0 {
		fmt.Printf("  Arguments: %v\n", cmd.Arguments)
	}
	if len(cmd.Flags) > 0 {
		fmt.Printf("  Flags: %v\n", cmd.Flags)
	}
	if len(cmd.Options) > 0 {
		fmt.Printf("  Options: %v\n", cmd.Options)
	}
}
