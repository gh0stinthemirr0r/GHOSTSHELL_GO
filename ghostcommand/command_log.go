// File: main.go
package ghostcommand

import (
	"fmt"
	"os"
	"strings"

	"ghostshell/ghostbrowse"

	"go.uber.org/zap"
)

// SimpleErrorHandler is a basic implementation of the ErrorHandler interface.
// It logs errors using the zap logger.
type SimpleErrorHandler struct {
	logger *zap.Logger
}

// NewSimpleErrorHandler initializes a new SimpleErrorHandler with a zap logger.
func NewSimpleErrorHandler(logger *zap.Logger) *SimpleErrorHandler {
	return &SimpleErrorHandler{
		logger: logger,
	}
}

// HandleError logs errors with a specific context and message.
func (eh *SimpleErrorHandler) HandleError(context, message string) {
	eh.logger.Error("Error in "+context, zap.String("message", message))
}

func main() {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize zap logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize ErrorHandler
	errorHandler := NewSimpleErrorHandler(logger)

	// Initialize CommandHandler
	commandHandler := ghostbrowse.NewCommandHandler(errorHandler, logger)

	// Sample commands to process
	commands := []string{"help", "ping", "unknown", "exit"}

	for _, cmd := range commands {
		fmt.Printf("\nExecuting command: %s\n", cmd)
		success := commandHandler.ProcessCommand(cmd)
		if success {
			fmt.Printf("Command '%s' executed successfully.\n", cmd)
		} else {
			fmt.Printf("Command '%s' failed to execute.\n", cmd)
		}
	}

	// Interactive command processing (optional)
	fmt.Println("\nEnter commands (type 'exit' to quit):")
	for {
		fmt.Print("> ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println("Error reading input. Please try again.")
			continue
		}

		// Convert input to lowercase to handle case-insensitive commands
		command := strings.ToLower(input)

		success := commandHandler.ProcessCommand(command)
		if success {
			fmt.Printf("Command '%s' executed successfully.\n", command)
		} else {
			fmt.Printf("Command '%s' failed to execute.\n", command)
		}

		if command == "exit" {
			fmt.Println("Exiting command processor.")
			break
		}
	}
}
