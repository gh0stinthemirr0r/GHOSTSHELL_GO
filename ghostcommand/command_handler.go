package ghostcommand

import (
	"fmt"

	"ghostshell/ai"

	"go.uber.org/zap"
)

type AIErrorHandler struct {
	aiLoader *ai.ModelLoader
	logger   *zap.Logger
}

func NewAIErrorHandler(aiLoader *ai.ModelLoader, logger *zap.Logger) *AIErrorHandler {
	return &AIErrorHandler{
		aiLoader: aiLoader,
		logger:   logger,
	}
}

func (eh *AIErrorHandler) HandleError(context, message string) {
	eh.logger.Error("Error occurred", zap.String("context", context), zap.String("message", message))
}

func (eh *AIErrorHandler) HandleErrorWithAI(context, message string) string {
	eh.logger.Info("Generating AI response for error", zap.String("context", context), zap.String("message", message))

	if !eh.aiLoader.ModelLoaded() {
		errorMsg := "AI model is not loaded. Unable to generate an intelligent response."
		eh.logger.Warn(errorMsg)
		return errorMsg
	}

	// Use the AI model to generate an error response
	prompt := fmt.Sprintf("Context: %s\nError: %s\nProvide a detailed resolution and potential fixes.", context, message)
	response, err := eh.aiLoader.RunInference(prompt)
	if err != nil {
		eh.logger.Error("Failed to generate AI response", zap.Error(err))
		return "Failed to generate an AI response due to an internal error."
	}

	eh.logger.Info("AI response generated successfully", zap.String("response", response))
	return response
}
