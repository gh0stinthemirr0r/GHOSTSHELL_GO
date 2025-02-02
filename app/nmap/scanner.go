package nmap

import (
	"fmt"

	"github.com/Ullaakut/nmap/v3"
)

// NmapScanner implements the Scanner interface using Nmap
type NmapScanner struct {
	options *Options
	results *nmap.Run
}

// NewNmapScanner creates a new instance of NmapScanner
func NewNmapScanner(options *Options) *NmapScanner {
	return &NmapScanner{
		options: options,
	}
}

// Run executes the Nmap scan
func (s *NmapScanner) Run() error {
	scanner, err := nmap.NewScanner(
		nmap.WithTargets(s.options.Targets...),
		nmap.WithPorts(s.options.Ports),
		nmap.WithTimingTemplate(nmap.Timing(s.options.TimingTemplate)),
		nmap.WithServiceInfo(),
		nmap.WithOSDetection(),
		nmap.WithXMLResults(s.options.OutputFile),
	)
	if err != nil {
		return fmt.Errorf("failed to create Nmap scanner: %w", err)
	}

	results, warnings, err := scanner.Run()
	if err != nil {
		return fmt.Errorf("Nmap scan failed: %w", err)
	}

	if len(*warnings) > 0 {
		fmt.Printf("Warnings during scan: %v\n", *warnings)
	}

	s.results = results
	return nil
}

// SetOptions applies the provided options to the scanner
func (s *NmapScanner) SetOptions(options *Options) {
	s.options = options
}

// GetResults retrieves the results of the scan
func (s *NmapScanner) GetResults() ([]byte, error) {
	if s.results == nil {
		return nil, fmt.Errorf("no results available, please run the scan first")
	}

	xmlResults, err := s.results.MarshalXML()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results to XML: %w", err)
	}

	return xmlResults, nil
}
