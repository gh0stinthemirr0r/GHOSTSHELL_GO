package nmap

import (
	"fmt"
	"os"

	"github.com/Ullaakut/nmap/v3"
	"gopkg.in/yaml.v3"
)

// ParseYAMLResults parses the YAML results from an Nmap scan
func ParseYAMLResults(filePath string) (*nmap.Run, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open YAML file: %w", err)
	}
	defer file.Close()

	var results nmap.Run
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return &results, nil
}

// DisplayResults processes and prints the parsed Nmap results
func DisplayResults(results *nmap.Run) {
	for _, host := range results.Hosts {
		if len(host.Ports) == 0 || len(host.Addresses) == 0 {
			continue
		}

		fmt.Printf("Host: %s\n", host.Addresses[0].Addr)
		for _, port := range host.Ports {
			fmt.Printf("  Port %d/%s is %s (%s)\n",
				port.ID, port.Protocol, port.State, port.Service.Name)
		}
	}
}

// SaveResultsAsYAML saves the Nmap results as a YAML file
func SaveResultsAsYAML(results *nmap.Run, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create YAML file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}
