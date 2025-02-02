package httpcrawler

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// RunIntegrationTest runs an integration test with provided inputs and expected outputs
func RunIntegrationTest(t *testing.T, inputTargets []string, expectedOutputs []string, probeFunc func(string) (Result, error)) {
	var actualOutputs []string

	// Probe each target and collect results
	for _, target := range inputTargets {
		result, err := probeFunc(target)
		if err != nil {
			t.Errorf("Error probing target %s: %v", target, err)
			continue
		}
		actualOutputs = append(actualOutputs, fmt.Sprintf("%s: %d", result.URL, result.Status))
	}

	// Compare actual outputs with expected outputs
	assert.ElementsMatch(t, expectedOutputs, actualOutputs, "Integration test failed")
}

// LoadTestCases loads test cases from a file
func LoadTestCases(filePath string) ([]string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read test cases file: %w", err)
	}

	testCases := strings.Split(strings.TrimSpace(string(data)), "\n")
	return testCases, nil
}
