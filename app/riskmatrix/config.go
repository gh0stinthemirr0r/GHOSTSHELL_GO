package riskmatrix

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the configuration for the Risk Matrix application
// Includes OpenAI API keys and other settings

type RiskMatrix struct {
	apiKEY