package ghostcrawler

import (
	"net/http"
	"sync"
	"time"
)

// SessionManager manages shared HTTP client sessions.
type SessionManager struct {
	client *http.Client
	once   sync.Once
}

var defaultSessionManager *SessionManager

// GetSessionManager returns a singleton instance of SessionManager.
func GetSessionManager() *SessionManager {
	if defaultSessionManager == nil {
		defaultSessionManager = &SessionManager{}
		defaultSessionManager.once.Do(func() {
			defaultSessionManager.client = &http.Client{
				Timeout: 10 * time.Second,
			}
		})
	}
	return defaultSessionManager
}

// GetClient retrieves the shared HTTP client.
func (s *SessionManager) GetClient() *http.Client {
	return s.client
}

// SetTimeout allows updating the timeout for the HTTP client.
func (s *SessionManager) SetTimeout(timeout time.Duration) {
	s.client.Timeout = timeout
}

// SetCustomTransport allows updating the HTTP transport for custom configurations.
func (s *SessionManager) SetCustomTransport(transport http.RoundTripper) {
	s.client.Transport = transport
}
