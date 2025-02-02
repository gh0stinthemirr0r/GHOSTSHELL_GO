package source

import "context"

// Source defines the interface that all data sources must implement.
type Source interface {
	// Fetch retrieves data from the source based on the given input.
	Fetch(ctx context.Context, input string) ([]Result, error)

	// Name returns the name of the source.
	Name() string
}

// Result represents the output of a source fetch operation.
type Result struct {
	URL       string // The URL retrieved from the source
	Source    string // The name of the source
	Extracted bool   // Whether the URL was extracted or directly fetched
}
