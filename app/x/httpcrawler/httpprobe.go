package httpcrawler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// HTTPProbe handles HTTP probing for targets
type HTTPProbe struct {
	client *http.Client
	config *Config
}

// NewHTTPProbe creates a new HTTPProbe instance
func NewHTTPProbe(config *Config) *HTTPProbe {
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	return &HTTPProbe{
		client: client,
		config: config,
	}
}

// Probe performs an HTTP request to the given target and returns the result
func (h *HTTPProbe) Probe(target string) (Result, error) {
	request, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set custom headers
	request.Header.Set("User-Agent", h.config.UserAgent)

	response, err := h.client.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return Result{}, fmt.Errorf("failed to read response body: %w", err)
	}

	result := Result{
		URL:    target,
		Status: response.StatusCode,
		Body:   string(body),
	}

	return result, nil
}
