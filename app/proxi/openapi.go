package proxi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// OpenAPISpec represents the OpenAPI specification structure
type OpenAPISpec struct {
	OpenAPI string                 `json:"openapi"`
	Info    map[string]interface{} `json:"info"`
	Paths   map[string]interface{} `json:"paths"`
}

// NewOpenAPISpec creates a new OpenAPISpec with default values
func NewOpenAPISpec() *OpenAPISpec {
	return &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: map[string]interface{}{
			"title":       "Proxi API",
			"description": "Automatically generated OpenAPI spec from Proxi",
			"version":     "1.0.0",
		},
		Paths: make(map[string]interface{}),
	}
}

// AddPath adds a new path to the OpenAPI spec
func (spec *OpenAPISpec) AddPath(path string, method string, details map[string]interface{}) {
	if _, exists := spec.Paths[path]; !exists {
		spec.Paths[path] = make(map[string]interface{})
	}
	methods := spec.Paths[path].(map[string]interface{})
	methods[method] = details
}

// Save writes the OpenAPI spec to a JSON file
func (spec *OpenAPISpec) Save(filePath string) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OpenAPI spec: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write OpenAPI spec to file: %w", err)
	}

	fmt.Printf("OpenAPI spec saved to %s\n", filePath)
	return nil
}

// LoadOpenAPISpec loads an OpenAPI spec from a JSON file
func LoadOpenAPISpec(filePath string) (*OpenAPISpec, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open OpenAPI spec file: %w", err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI spec file: %w", err)
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAPI spec: %w", err)
	}

	return &spec, nil
}
