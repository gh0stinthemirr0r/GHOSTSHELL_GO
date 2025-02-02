package input

import (
	"bufio"
	"os"
	"strings"
)

// Handler is responsible for reading and sanitizing input for the TLD crawler.
type Handler struct {
	InputLines []string
}

// NewHandler initializes a new input handler.
func NewHandler() *Handler {
	return &Handler{
		InputLines: make([]string, 0),
	}
}

// ReadFromFile reads input lines from a specified file.
func (h *Handler) ReadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			h.InputLines = append(h.InputLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// ReadFromStdin reads input lines from standard input.
func (h *Handler) ReadFromStdin() error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			h.InputLines = append(h.InputLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// GetInputLines returns the sanitized list of input lines.
func (h *Handler) GetInputLines() []string {
	return h.InputLines
}
