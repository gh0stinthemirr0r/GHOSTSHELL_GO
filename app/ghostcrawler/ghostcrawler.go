package ghostcrawler

import (
	"context"
	"errors"
	"sync"
)

// Crawler defines the interface for all crawlers.
type Crawler interface {
	Name() string
	Start(ctx context.Context, input []string, output chan<- Result) error
}

// Manager manages all crawlers and orchestrates their execution.
type Manager struct {
	crawlers map[string]Crawler
	results  chan Result
	mu       sync.Mutex
}

// Result represents the output of a crawler.
type Result struct {
	CrawlerName string
	Data        interface{}
	Error       error
}

// NewManager initializes and returns a new Manager instance.
func NewManager() *Manager {
	return &Manager{
		crawlers: make(map[string]Crawler),
		results:  make(chan Result, 100),
	}
}

// RegisterCrawler adds a new crawler to the manager.
func (m *Manager) RegisterCrawler(crawler Crawler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.crawlers[crawler.Name()] = crawler
}

// StartCrawler starts a specific crawler by name.
func (m *Manager) StartCrawler(ctx context.Context, name string, input []string) error {
	m.mu.Lock()
	crawler, exists := m.crawlers[name]
	m.mu.Unlock()
	if !exists {
		return errors.New("crawler not found")
	}

	go func() {
		if err := crawler.Start(ctx, input, m.results); err != nil {
			m.results <- Result{CrawlerName: name, Error: err}
		}
	}()
	return nil
}

// GetResults retrieves results from all running crawlers.
func (m *Manager) GetResults() []Result {
	m.mu.Lock()
	defer m.mu.Unlock()

	var results []Result
	for {
		select {
		case res := <-m.results:
			results = append(results, res)
		default:
			return results
		}
	}
}

// ListCrawlers lists all registered crawlers.
func (m *Manager) ListCrawlers() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var names []string
	for name := range m.crawlers {
		names = append(names, name)
	}
	return names
}
