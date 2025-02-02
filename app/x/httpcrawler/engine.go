package httpcrawler

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Runner orchestrates the crawling process
type Runner struct {
	options   *Options
	config    *Config
	httpProbe *HTTPProbe
	logger    *zap.SugaredLogger
}

// NewRunner initializes a new Runner instance
func NewRunner(options *Options, config *Config) (*Runner, error) {
	httpProbe := NewHTTPProbe(config)

	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	sugar := logger.Sugar()

	return &Runner{
		options:   options,
		config:    config,
		httpProbe: httpProbe,
		logger:    sugar,
	}, nil
}

// Run executes the HTTP probing workflow
func (r *Runner) Run(ctx context.Context) error {
	results := make(chan Result, r.options.Concurrency)
	var wg sync.WaitGroup

	// Start probing targets
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, target := range r.options.Targets {
			wg.Add(1)
			go func(target string) {
				defer wg.Done()
				result, err := r.httpProbe.Probe(target)
				if err != nil {
					r.logger.Errorf("Error probing target %s: %v", target, err)
					return
				}
				results <- result
			}(target)
		}
		close(results)
	}()

	// Output results
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := writeResults(results, r.options.OutputFile)
		if err != nil {
			r.logger.Errorf("Error writing results: %v", err)
		}
	}()

	wg.Wait()
	return nil
}
