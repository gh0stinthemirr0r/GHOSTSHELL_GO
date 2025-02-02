package httpcrawler

import (
	"regexp"
	"strings"
)

// Filter represents a filter for HTTP probe results
type Filter struct {
	StatusCodes []int
	Regex       *regexp.Regexp
	Content     string
}

// Matches checks if a result matches the filter criteria
func (f *Filter) Matches(result *Result) bool {
	// Check status codes
	if len(f.StatusCodes) > 0 {
		matched := false
		for _, code := range f.StatusCodes {
			if result.Status == code {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check content regex
	if f.Regex != nil {
		if !f.Regex.MatchString(result.Body) {
			return false
		}
	}

	// Check content substring
	if f.Content != "" {
		if !strings.Contains(result.Body, f.Content) {
			return false
		}
	}

	return true
}
