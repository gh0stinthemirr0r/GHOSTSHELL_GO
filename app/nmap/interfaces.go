package nmap

// Scanner defines the interface for any type of scanner
// This can be implemented by various scanning tools (e.g., Nmap, custom scanners)
type Scanner interface {
	// Run executes the scan and returns an error if the scan fails
	Run() error

	// SetOptions applies the provided options to the scanner
	SetOptions(options *Options)

	// GetResults retrieves the results of the scan
	GetResults() ([]byte, error)
}
