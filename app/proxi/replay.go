package proxi

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

// ReplayRequest replays a captured HTTP request and returns the response
func ReplayRequest(method, url string, headers map[string][]string, body string) (*http.Response, error) {
	reqBody := bytes.NewBufferString(body)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to the request
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// ReadResponse reads and returns the response body as a string
func ReadResponse(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}
