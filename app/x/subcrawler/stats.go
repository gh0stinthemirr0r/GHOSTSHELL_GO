package subcrawler

import (
	"fmt"
	"sync"
)

// SourceStats keeps track of statistics for subdomain enumeration sources
type SourceStats struct {
	mu      sync.Mutex
	Stats   map[string]int
	Errors  map[string]int
	Success map[string]int
}

// NewSourceStats creates a new instance of SourceStats
func NewSourceStats() *SourceStats {
	return &SourceStats{
		Stats:   make(map[string]int),
		Errors:  make(map[string]int),
		Success: make(map[string]int),
	}
}

// Increment increments the count for a source in the Stats map
func (s *SourceStats) Increment(source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Stats[source]++
}

// RecordError increments the error count for a source
func (s *SourceStats) RecordError(source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Errors[source]++
}

// RecordSuccess increments the success count for a source
func (s *SourceStats) RecordSuccess(source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Success[source]++
}

// PrintStats prints the collected statistics
func (s *SourceStats) PrintStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Println("Source Statistics:")
	for source, count := range s.Stats {
		success := s.Success[source]
		errors := s.Errors[source]
		fmt.Printf("%s: Total=%d, Success=%d, Errors=%d\n", source, count, success, errors)
	}
}
