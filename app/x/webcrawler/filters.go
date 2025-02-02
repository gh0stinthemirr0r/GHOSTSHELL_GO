package webcrawler

import (
	"regexp"
)

// Filter defines criteria for filtering crawl results
type Filter struct {
	IncludePatterns []string
	ExcludePatterns []string
	MaxDepth        int
}

// Matches checks if a URL matches the include/exclude patterns
func (f *Filter) Matches(url string, depth int) bool {
	if depth > f.MaxDepth {
		return false
	}

	// Check exclude patterns
	for _, pattern := range f.ExcludePatterns {
		if matched, _ := regexp.MatchString(pattern, url); matched {
			return false
		}
	}

	// Check include patterns
	if len(f.IncludePatterns) > 0 {
		for _, pattern := range f.IncludePatterns {
			if matched, _ := regexp.MatchString(pattern, url); matched {
				return true
			}
		}
		return false
	}

	// Default to true if no include patterns are specified
	return true
}
