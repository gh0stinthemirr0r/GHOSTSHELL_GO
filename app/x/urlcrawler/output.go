package urlcrawler

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Writer manages output operations for the URL crawler.
type Writer struct {
	useJSON bool
	mutex   sync.Mutex
}

// NewWriter initializes a new Writer instance.
func NewWriter(useJSON bool) *Writer {
	return &Writer{
		useJSON: useJSON,
	}
}

// WriteLine writes a single line of output, handling plain text or JSON formats.
func (w *Writer) WriteLine(data interface{}) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.useJSON {
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println(data)
	}
	return nil
}

// WriteToFile writes output to a specified file, handling plain text or JSON formats.
func (w *Writer) WriteToFile(filePath string, data interface{}) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if w.useJSON {
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to write JSON data to file: %w", err)
		}
	} else {
		if _, err := fmt.Fprintln(file, data); err != nil {
			return fmt.Errorf("failed to write data to file: %w", err)
		}
	}

	return nil
}
