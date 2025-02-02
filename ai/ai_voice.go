package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

const (
	elevenLabsAPIURL = "https://api.elevenlabs.io/v1/text-to-speech"
	freeTTSAPIURL    = "https://api.freetts.com/v1/tts"
)

type ElevenLabsConfig struct {
	APIKey      string
	VoiceID     string
	OutputDir   string
	DefaultLang string
}

type FreeTTSConfig struct {
	OutputDir   string
	DefaultLang string
}

type ElevenLabs struct {
	config ElevenLabsConfig
	logger *zap.Logger
	mu     sync.Mutex
}

// NewElevenLabs initializes the ElevenLabs TTS client.
func NewElevenLabs(config ElevenLabsConfig, logger *zap.Logger) (*ElevenLabs, error) {
	if config.APIKey == "" || config.VoiceID == "" {
		return nil, fmt.Errorf("invalid ElevenLabs configuration")
	}

	// Ensure the output directory exists
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &ElevenLabs{
		config: config,
		logger: logger,
	}, nil
}

// SynthesizeSpeech sends a request to ElevenLabs API and saves the resulting audio.
func (el *ElevenLabs) SynthesizeSpeech(text, filename string) (string, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	if text == "" {
		return "", fmt.Errorf("text cannot be empty")
	}

	// Prepare the request body
	requestBody := map[string]interface{}{
		"text":      text,
		"voice_id":  el.config.VoiceID,
		"lang":      el.config.DefaultLang,
		"stability": 0.75, // Optional TTS parameters
		"clarity":   0.8,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", elevenLabsAPIURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", el.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("received non-OK response: %d, %s", resp.StatusCode, string(body))
	}

	// Save the audio file
	outputPath := filepath.Join(el.config.OutputDir, filename)
	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save audio file: %w", err)
	}

	el.logger.Info("Audio file saved successfully", zap.String("path", outputPath))
	return outputPath, nil
}

// FreeTTS handles free text-to-speech synthesis.
type FreeTTS struct {
	config FreeTTSConfig
	logger *zap.Logger
}

// NewFreeTTS initializes a FreeTTS client.
func NewFreeTTS(config FreeTTSConfig, logger *zap.Logger) (*FreeTTS, error) {
	// Ensure the output directory exists
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &FreeTTS{
		config: config,
		logger: logger,
	}, nil
}

// SynthesizeSpeech sends a request to FreeTTS API and saves the resulting audio.
func (ft *FreeTTS) SynthesizeSpeech(text, filename string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("text cannot be empty")
	}

	// Prepare the request body
	requestBody := map[string]interface{}{
		"text": text,
		"lang": ft.config.DefaultLang,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", freeTTSAPIURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("received non-OK response: %d, %s", resp.StatusCode, string(body))
	}

	// Save the audio file
	outputPath := filepath.Join(ft.config.OutputDir, filename)
	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save audio file: %w", err)
	}

	ft.logger.Info("Audio file saved successfully", zap.String("path", outputPath))
	return outputPath, nil
}

// DetermineTTS determines which TTS service to use based on the configuration.
func DetermineTTS(aiConfigPath string, logger *zap.Logger) (interface{}, error) {
	config, err := LoadConfig(aiConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load AI configuration: %w", err)
	}

	if config.ControlParams != nil && config.ControlParams["elevenlabs_api_key"] != nil {
		elevenLabsConfig := ElevenLabsConfig{
			APIKey:      config.ControlParams["elevenlabs_api_key"].(string),
			VoiceID:     config.ControlParams["elevenlabs_voice_id"].(string),
			OutputDir:   "./ai/audio",
			DefaultLang: "en-US",
		}
		return NewElevenLabs(elevenLabsConfig, logger)
	}

	// Default to FreeTTS if ElevenLabs is not configured
	freeTTSConfig := FreeTTSConfig{
		OutputDir:   "./ai/audio",
		DefaultLang: "en-US",
	}
	return NewFreeTTS(freeTTSConfig, logger)
}

// Example usage in the GhostShell system.
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ttsService, err := DetermineTTS("./ghostshell/config/ai.yaml", logger)
	if err != nil {
		logger.Fatal("Failed to initialize TTS service", zap.Error(err))
	}

	var speechFile string
	switch tts := ttsService.(type) {
	case *ElevenLabs:
		speechFile, err = tts.SynthesizeSpeech("Welcome to GhostShell, your AI-driven terminal.", "welcome_message.mp3")
	case *FreeTTS:
		speechFile, err = tts.SynthesizeSpeech("Welcome to GhostShell, your AI-driven terminal.", "welcome_message.mp3")
	}

	if err != nil {
		logger.Fatal("Failed to synthesize speech", zap.Error(err))
	}

	logger.Info("Speech synthesis complete", zap.String("file", speechFile))
}
