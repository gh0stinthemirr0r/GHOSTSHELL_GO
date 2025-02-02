package nmap

import (
	"regexp"

	"github.com/Ullaakut/nmap/v3"
)

// Filter represents a filter for Nmap results
type Filter struct {
	Port       int
	State      string
	Service    string
	OS         string
	RegexMatch *regexp.Regexp
}

// Matches checks if a given Nmap result matches the filter criteria
func (f *Filter) Matches(result *nmap.Port) bool {
	// Check port number
	if f.Port != 0 && f.Port != result.ID {
		return false
	}

	// Check port state
	if f.State != "" && f.State != result.State {
		return false
	}

	// Check service name
	if f.Service != "" && result.Service != nil && f.Service != result.Service.Name {
		return false
	}

	// Check regex match on service name
	if f.RegexMatch != nil && result.Service != nil {
		if !f.RegexMatch.MatchString(result.Service.Name) {
			return false
		}
	}

	return true
}
