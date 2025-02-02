// Package cdnscanner handles data sources and provides logic for accessing CDN-related information.
package cdncrawler

import (
	"encoding/json"
	"fmt"
	"os"
)

// Source represents a single data source with relevant metadata.
type Source struct {
	Name        string   `json:"name"`        // Name of the source
	Description string   `json:"description"` // Description of the source
	CDNs        []string `json:"cdns"`        // List of CDNs associated with the source
}

// DataSourceManager handles loading and accessing data sources.
type DataSourceManager struct {
	sources []Source
}

// NewDataSourceManager creates and returns a new instance of DataSourceManager.
func NewDataSourceManager() *DataSourceManager {
	return &DataSourceManager{
		sources: []Source{},
	}
}

// LoadFromFile loads data sources from a JSON file.
func (dsm *DataSourceManager) LoadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open sources file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&dsm.sources); err != nil {
		return fmt.Errorf("failed to decode sources file: %w", err)
	}

	return nil
}

// GetSources returns all loaded data sources.
func (dsm *DataSourceManager) GetSources() []Source {
	return dsm.sources
}

// FindSourceByName finds and returns a data source by its name.
func (dsm *DataSourceManager) FindSourceByName(name string) (*Source, error) {
	for _, source := range dsm.sources {
		if source.Name == name {
			return &source, nil
		}
	}
	return nil, fmt.Errorf("source not found: %s", name)
}

// DisplaySources prints all available sources to the console.
func (dsm *DataSourceManager) DisplaySources() {
	fmt.Println("Available Data Sources:")
	fmt.Println("───────────────────────")
	for _, source := range dsm.sources {
		fmt.Printf("Name: %s\n", source.Name)
		fmt.Printf("Description: %s\n", source.Description)
		fmt.Printf("CDNs: %v\n", source.CDNs)
		fmt.Println("───────────────────────")
	}
}

// Example usage:
// func main() {
//     manager := NewDataSourceManager()
//     err := manager.LoadFromFile("sources_data.json")
//     if err != nil {
//         fmt.Println("Error:", err)
//         return
//     }
//
//     manager.DisplaySources()
//
//     source, err := manager.FindSourceByName("ExampleSource")
//     if err != nil {
//         fmt.Println("Error:", err)
//     } else {
//         fmt.Printf("Found Source: %+v\n", source)
//     }
// }
