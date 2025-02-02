package ai

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
)

// SanitizeInput removes potentially harmful characters from a user-provided string.
func SanitizeInput(input string) string {
	// Remove harmful or unsupported characters
	cleanInput := strings.ReplaceAll(input, "\n", " ")
	cleanInput = strings.ReplaceAll(cleanInput, "\t", " ")
	cleanInput = strings.TrimSpace(cleanInput)

	// Ensure only alphanumeric characters, spaces, and basic punctuation remain
	reg, _ := regexp.Compile(`[^a-zA-Z0-9 .,!?'-]`)
	cleanInput = reg.ReplaceAllString(cleanInput, "")

	// Limit length to avoid excessive processing
	const maxInputLength = 500
	if len(cleanInput) > maxInputLength {
		cleanInput = cleanInput[:maxInputLength]
	}

	return cleanInput
}

// FormatTimestamp returns the current timestamp in RFC3339 format.
func FormatTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// IsValidModelPath checks if a model path is valid based on its extension.
func IsValidModelPath(path string) error {
	if !strings.HasSuffix(path, ".ggml") && !strings.HasSuffix(path, ".gguf") {
		return errors.New("invalid model path: only .ggml or .gguf files are supported")
	}
	return nil
}

// GenerateID creates a simple unique identifier based on the current time and random bytes.
func GenerateID() string {
	randomBytes := make([]byte, 4) // Generate 4 random bytes
	_, err := rand.Read(randomBytes)
	if err != nil {
		return time.Now().Format("20060102150405") // Fallback to time-only ID
	}
	return time.Now().Format("20060102150405") + hex.EncodeToString(randomBytes)
}

// TruncateString truncates a string to the specified length, adding ellipsis if truncated.
func TruncateString(input string, maxLength int) string {
	if len(input) <= maxLength {
		return input
	}
	return input[:maxLength-3] + "..."
}

// ValidateEmail checks if the given string is a valid email address.
func ValidateEmail(email string) error {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	if matched, _ := regexp.MatchString(emailRegex, email); !matched {
		return errors.New("invalid email address")
	}
	return nil
}

// ParseDuration parses a duration string and returns the time.Duration.
// Supported formats: "1h", "30m", "10s".
func ParseDuration(durationStr string) (time.Duration, error) {
	return time.ParseDuration(durationStr)
}
