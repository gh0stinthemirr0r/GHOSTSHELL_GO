package nmap

import "fmt"

// NmapError represents an error specific to Nmap operations
type NmapError struct {
	Message string
	Code    int
}

// NewNmapError creates a new NmapError instance
func NewNmapError(message string, code int) *NmapError {
	return &NmapError{
		Message: message,
		Code:    code,
	}
}

// Error implements the error interface for NmapError
func (e *NmapError) Error() string {
	return fmt.Sprintf("NmapError (Code %d): %s", e.Code, e.Message)
}

// IsCritical determines if an error is critical
func (e *NmapError) IsCritical() bool {
	return e.Code >= 500
}
