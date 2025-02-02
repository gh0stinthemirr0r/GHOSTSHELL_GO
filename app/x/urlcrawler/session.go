package session

import (
	"net/http"
	"sync"
	"time"
)

// Session provides an HTTP client with shared settings for reuse across sources.
type Session struct {
	Client *http.Client
	Once   sync.Once
}

var defaultSession *Session

// GetDefaultSession initializes and returns the default session.
func GetDefaultSession() *Session {
	if defaultSession == nil {
		defaultSession = &Session{}
		defaultSession.Once.Do(func() {
			defaultSession.Client = &http.Client{
				Timeout: 10 * time.Second,
			}
		})
	}
	return defaultSession
}
