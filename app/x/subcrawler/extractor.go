package subcrawler

import (
	"regexp"
)

// Extractor handles subdomain extraction from raw data
// Supports regex-based parsing and normalization

type Extractor struct {
	regex *regexp.Regexp
}

// NewExtractor creates a new instance of Extractor with a default regex
func NewExtractor() *Extractor {
	return &Extractor{
		regex: regexp.MustCompile(`[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
	}
}

// Extract parses subdomains from the input string
func (e *Extractor) Extract(data string) []string {
	matches := e.regex.FindAllString(data, -1)
	unique := make(map[string]bool)
	for _, match := range matches {
		unique[match] = true
	}

	var results []string
	for subdomain := range unique {
		results = append(results, subdomain)
	}

	return results
}
