package core

import (
	"errors"
	"fmt"
	"os"

	llama "github.com/go-skynet/go-llama.cpp"
)

type LanguageModel struct {
	modelPath     string
	contextWindow int
	temperature   float32
	maxTokens     int

	model   *llama.LLama
	loading bool
	ready   bool
}

func NewLanguageModel(modelPath string) (*LanguageModel, error) {
	if _, err := os.Stat(modelPath); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("model file not found: %s", modelPath)
	}

	return &LanguageModel{
		modelPath:     modelPath,
		contextWindow: 2048,
		temperature:   0.7,
		maxTokens:     1024,
	}, nil
}

func (lm *LanguageModel) LoadModel() error {
	if lm.loading {
		return errors.New("model is already loading")
	}
	if lm.ready {
		return errors.New("model is already loaded")
	}

	lm.loading = true
	defer func() { lm.loading = false }()

	var err error
	lm.model, err = llama.New(lm.modelPath, llama.SetContext(lm.contextWindow))
	if err != nil {
		return fmt.Errorf("failed to load model: %v", err)
	}

	lm.ready = true
	return nil
}

func (lm *LanguageModel) RunInference(prompt string) (string, error) {
	if !lm.ready {
		return "", errors.New("model is not loaded or ready")
	}

	options := []llama.PredictOption{
		llama.SetTemperature(lm.temperature),
		llama.SetMaxTokens(lm.maxTokens),
		llama.SetTopP(0.95),
	}

	output, err := lm.model.Predict(prompt, options...)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %v", err)
	}

	return output, nil
}

func (lm *LanguageModel) IsReady() bool {
	return lm.ready
}

func (lm *LanguageModel) SetParameters(temperature float32, contextWindow int, maxTokens int) {
	lm.temperature = temperature
	lm.contextWindow = contextWindow
	lm.maxTokens = maxTokens
}
