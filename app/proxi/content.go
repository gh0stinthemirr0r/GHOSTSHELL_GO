package proxi

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// Content represents HTTP request and response payloads
type Content struct {
	RequestHeaders  map[string][]string `json:"request_headers"`
	RequestBody     string              `json:"request_body"`
	ResponseHeaders map[string][]string `json:"response_headers"`
	ResponseBody    string              `json:"response_body"`
	StatusCode      int                 `json:"status_code"`
}

// NewContent creates a new Content instance from HTTP request and response
func NewContent(req *http.Request, resp *http.Response) (*Content, error) {
	requestBody, err := readBody(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	responseBody, err := readBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Content{
		RequestHeaders:  req.Header,
		RequestBody:     requestBody,
		ResponseHeaders: resp.Header,
		ResponseBody:    responseBody,
		StatusCode:      resp.StatusCode,
	}, nil
}

// readBody reads and returns the content of an HTTP body
func readBody(body io.ReadCloser) (string, error) {
	if body == nil {
		return "", nil
	}
	defer body.Close()

	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToJSON serializes the Content into JSON format
func (c *Content) ToJSON() (string, error) {
	jsonData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal content to JSON: %w", err)
	}
	return string(jsonData), nil
}
