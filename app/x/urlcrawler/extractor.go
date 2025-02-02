package extractor

import (
	"regexp"
)

// Extractor provides methods to extract URLs from raw input.
type Extractor struct {
	urlRegex *regexp.Regexp
}

// NewExtractor initializes a new Extractor instance.
func NewExtractor() *Extractor {
	return &Extractor{
		urlRegex: regexp.MustCompile(`https?://[\w.-]+(?:\.[\w.-]+)+(?:[/?#]\S*)?`),
	}
}

// ExtractURLs extracts all URLs from the given input text.
func (e *Extractor) ExtractURLs(input string) []string {
	return e.urlRegex.FindAllString(input, -1)
}
