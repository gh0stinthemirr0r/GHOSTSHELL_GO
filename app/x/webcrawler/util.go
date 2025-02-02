package webcrawler

import (
	"net/url"
)

// NormalizeURL ensures a URL is in a standardized format
func NormalizeURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	return parsedURL.String(), nil
}

// IsInternalURL checks if a URL belongs to the same domain as the base URL
func IsInternalURL(baseURL, targetURL string) (bool, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return false, err
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		return false, err
	}

	return base.Hostname() == target.Hostname(), nil
}

// JoinURL combines a base URL and a relative path into a full URL
func JoinURL(baseURL, relativePath string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	joined, err := base.Parse(relativePath)
	if err != nil {
		return "", err
	}

	return joined.String(), nil
}
