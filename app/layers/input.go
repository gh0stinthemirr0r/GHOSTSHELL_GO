package layers

import (
	"flag"
	"fmt"
	"os"
)

// InputArgs holds the parsed command-line arguments.
type InputArgs struct {
	OutputFormat string // Desired output format: csv, pdf, or json
	OutputPath   string // Path to save the output report
	ConfigPath   string // Path to the configuration file
}

// ParseInput parses and validates command-line arguments.
func ParseInput() (*InputArgs, error) {
	// Define command-line flags
	outputFormat := flag.String("format", "csv", "Output format for the report (csv, pdf, or json)")
	outputPath := flag.String("output", "report.csv", "Path to save the output report")
	configPath := flag.String("config", "config.json", "Path to the configuration file")

	// Parse flags
	flag.Parse()

	// Validate output format
	if *outputFormat != "csv" && *outputFormat != "pdf" && *outputFormat != "json" {
		return nil, fmt.Errorf("invalid output format: %s. Allowed values are: csv, pdf, json", *outputFormat)
	}

	// Validate output path
	if *outputPath == "" {
		return nil, fmt.Errorf("output path cannot be empty")
	}

	// Create and return the InputArgs struct
	return &InputArgs{
		OutputFormat: *outputFormat,
		OutputPath:   *outputPath,
		ConfigPath:   *configPath,
	}, nil
}

// PrintUsage displays the application usage instructions.
func PrintUsage() {
	fmt.Println("Usage: osi-tester [options]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

// ValidateArgs ensures that the provided arguments meet the application's requirements.
func ValidateArgs(args *InputArgs) error {
	if _, err := os.Stat(args.ConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist at path: %s", args.ConfigPath)
	}
	return nil
}
