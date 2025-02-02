package proxi

import (
	"fmt"
	"log"
)

// Verbosity levels
const (
	VerboseLevelNone    = 0
	VerboseLevelDefault = 1
	VerboseLevelDebug   = 2
)

// VerboseLogger controls the verbosity of log messages
type VerboseLogger struct {
	Level int
}

// Log logs a message if the verbosity level is sufficient
func (v *VerboseLogger) Log(level int, format string, args ...interface{}) {
	if v.Level >= level {
		log.Printf(format, args...)
	}
}

// Debug logs debug messages
func (v *VerboseLogger) Debug(format string, args ...interface{}) {
	v.Log(VerboseLevelDebug, format, args...)
}

// Info logs informational messages
func (v *VerboseLogger) Info(format string, args ...interface{}) {
	v.Log(VerboseLevelDefault, format, args...)
}

// Error logs error messages
func (v *VerboseLogger) Error(format string, args ...interface{}) {
	fmt.Printf("ERROR: "+format+"\n", args...)
}
