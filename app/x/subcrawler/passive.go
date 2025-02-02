package subcrawler

import (
	"fmt"
	"sync"
)

// PassiveSource represents a source for passive subdomain enumeration
type PassiveSource interface {
	Name() string
	Enumerate(domain string, results chan<- string) error
}

// EnumeratePassiveSources runs passive enumeration across multiple sources
func EnumeratePassiveSources(sources []PassiveSource, domain string) ([]string, error) {
	results := make(chan string)
	var wg sync.WaitGroup

	// Start enumeration for each source
	for _, source := range sources {
		source := source
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := source.Enumerate(domain, results); err != nil {
				fmt.Printf("Error with source %s: %v\n", source.Name(), err)
			}
		}()
	}

	// Close results channel when all sources are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect unique results
	uniqueResults := make(map[string]bool)
	for result := range results {
		uniqueResults[result] = true
	}

	// Convert map keys to a slice
	var finalResults []string
	for subdomain := range uniqueResults {
		finalResults = append(finalResults, subdomain)
	}

	return finalResults, nil
}
