package discovery

// Query represents a search query for an agent
type Query struct {
	Query string // The search query string
	Limit int    // The maximum number of results to retrieve
}

// Agent is an interface that must be implemented by all source agents
type Agent interface {
	// Query executes a search query and returns a channel of results or an error
	Query(*Session, *Query) (chan Result, error)
	// Name returns the name of the agent/source
	Name() string
}
