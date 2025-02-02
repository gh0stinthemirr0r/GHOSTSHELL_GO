package ai

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// SetupRoutes initializes the server routes for AI functionality.
// We pass in:
//   - app: *fiber.App
//   - loader: *ModelLoader
//   - logger: *zap.SugaredLogger
func SetupRoutes(app *fiber.App, loader *ModelLoader, logger *zap.SugaredLogger) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		logger.Info("Health check endpoint hit")
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Server is running",
		})
	})

	// Load model from current config.ModelPath
	app.Post("/model/load", func(c *fiber.Ctx) error {
		logger.Info("Load model request received")
		if err := loader.LoadModel(); err != nil {
			logger.Errorw("Failed to load model", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
		logger.Info("Model loaded successfully")
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Model loaded successfully",
		})
	})

	// Unload model
	app.Post("/model/unload", func(c *fiber.Ctx) error {
		logger.Info("Unload model request received")
		if err := loader.UnloadModel(); err != nil {
			logger.Errorw("Failed to unload model", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
		logger.Info("Model unloaded successfully")
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Model unloaded successfully",
		})
	})

	// Eject model
	app.Post("/model/eject", func(c *fiber.Ctx) error {
		logger.Info("Eject model request received")
		if err := loader.Eject(); err != nil {
			logger.Errorw("Failed to eject model", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
		logger.Info("Model ejected successfully")
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Model ejected successfully",
		})
	})

	// Select a new model path (unload current, update config, load new)
	// POST /model/select => { "model_path": "path/to/model.gguf" }
	app.Post("/model/select", func(c *fiber.Ctx) error {
		var req struct {
			ModelPath string `json:"model_path"`
		}
		if err := c.BodyParser(&req); err != nil {
			logger.Errorw("Failed to parse JSON for model path", "error", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid request payload",
			})
		}

		logger.Infow("Select model request", "model_path", req.ModelPath)
		if err := loader.LoadSelectedModel(req.ModelPath); err != nil {
			logger.Errorw("Failed to select/load new model", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Model path updated & model loaded",
		})
	})

	// Update control parameters
	// POST /model/control => { "temperature": 0.7, "max_tokens": 1024, "reload": true }
	app.Post("/model/control", func(c *fiber.Ctx) error {
		var req struct {
			Temperature float64 `json:"temperature"`
			MaxTokens   int     `json:"max_tokens"`
			Reload      bool    `json:"reload"`
		}
		if err := c.BodyParser(&req); err != nil {
			logger.Errorw("Failed to parse JSON for control params", "error", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid request payload",
			})
		}

		logger.Infow("Update control params request",
			"temp", req.Temperature, "max_tokens", req.MaxTokens, "reload", req.Reload)

		params := ControlParameters{
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		}

		if err := loader.SetControlParameters(params, req.Reload); err != nil {
			logger.Errorw("Failed to update control parameters", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Control parameters updated",
		})
	})

	// Add an endpoint to fetch the current model status
	app.Get("/model/status", func(c *fiber.Ctx) error {
		logger.Info("Model status request received")
		if loader.model == nil {
			return c.JSON(fiber.Map{
				"status":  "idle",
				"message": "No model currently loaded",
			})
		}

		return c.JSON(fiber.Map{
			"status":         "active",
			"message":        "Model is loaded and operational",
			"model_path":     loader.config.ModelPath,
			"control_params": loader.config.ControlParams,
		})
	})
}
