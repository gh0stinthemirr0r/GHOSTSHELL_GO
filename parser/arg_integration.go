package parser

import (
	"fmt"
	"ghostshell/app/asnscanner"
	"os"
)

// ParseCommandLine parses global CLI arguments and delegates to the appropriate tool.
func ParseCommandLine() {
	if len(os.Args) < 2 {
		fmt.Println("Please specify a tool to run (e.g., asnscanner, tldfinder).")
		os.Exit(1)
	}

	tool := os.Args[1]
	switch tool {
	case "asnscanner":
		options, err := asnscanner.ParseOptions()
		if err != nil {
			fmt.Println("Error:", err)
			asnscanner.PrintHelp()
			os.Exit(1)
		}
		runASNScanner(options)

	default:
		fmt.Printf("Unknown tool: %s\n", tool)
		os.Exit(1)
	}
}

// runASNScanner executes the asnscanner with the provided options.
func runASNScanner(options *asnscanner.Options) {
	// Call asnscanner logic
	fmt.Println("Running asnscanner with the following options:")
	fmt.Printf("Targets: %v\n", options.Targets)
	fmt.Printf("Output File: %s\n", options.OutputFile)
	fmt.Printf("Concurrency: %d\n", options.Concurrency)
	fmt.Printf("Verbose: %t\n", options.Verbose)
	// Pass options to asnscanner logic
}
