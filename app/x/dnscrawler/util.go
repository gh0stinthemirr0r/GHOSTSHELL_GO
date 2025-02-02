package dnscrawler

import (
	"bufio"
	"net/url"
	"os"
)

// linesInFile reads all lines from a file
func linesInFile(fileName string) ([]string, error) {
	result := []string{}
	file, err := os.Open(fileName)
	if err != nil {
		return result, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}

	return result, scanner.Err()
}

// isURL checks if a string is a valid URL
func isURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

// extractDomain extracts the hostname from a URL
func extractDomain(URL string) string {
	u, err := url.Parse(URL)
	if err != nil {
		return ""
	}

	return u.Hostname()
}
